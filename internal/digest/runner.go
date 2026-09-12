package digest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/stuttgart-things/homerun2-scout/internal/aggregator"
)

// claimTTL is how long the key claiming a window's digest lives: longer than
// any window is still pitched, so a restarted scout finds it.
const claimTTL = 8 * 24 * time.Hour

// Querier returns the groups of messages in [start, end).
type Querier interface {
	Window(ctx context.Context, start, end time.Time) ([]Row, error)
}

// Store claims a window's digest, so it is pitched once however many scout
// replicas run and however often scout restarts.
type Store interface {
	// Claim reports whether key was free and is now taken.
	Claim(ctx context.Context, key string, ttl time.Duration) (bool, error)
	// Release frees key again, for a digest that could not be pitched.
	Release(ctx context.Context, key string) error
}

// Pitcher delivers a digest message.
type Pitcher interface {
	Pitch(ctx context.Context, msg Message) error
}

// Config is what the runner pitches, and when.
type Config struct {
	Schedules []Schedule
	Location  *time.Location
	// Exclude are systems left out of the counts. The digest's own System
	// is always left out.
	Exclude    []string
	TopSystems int
	// System is the system the digest is pitched as.
	System string
	// Retention is how long scout keeps messages. The window before is only
	// compared when retention still holds it: all of it, give or take the
	// schedule's lateness, whose oldest minutes retention may already have
	// deleted by the time the digest runs. 0 means retention is off.
	Retention time.Duration
}

// Runner pitches the digests of cfg as their windows end.
type Runner struct {
	cfg      Config
	querier  Querier
	store    Store
	pitcher  Pitcher
	now      func() time.Time
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

// NewRunner returns a runner; Start begins pitching.
func NewRunner(cfg Config, querier Querier, store Store, pitcher Pitcher) *Runner {
	if cfg.Location == nil {
		cfg.Location = time.UTC
	}
	return &Runner{
		cfg: cfg, querier: querier, store: store, pitcher: pitcher,
		now: time.Now, interval: time.Minute, done: make(chan struct{}),
	}
}

// Start checks every minute whether a window has ended whose digest is due.
func (r *Runner) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)
	go func() {
		defer close(r.done)
		r.RunOnce(ctx)
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.RunOnce(ctx)
			}
		}
	}()
	var names []string
	for _, s := range r.cfg.Schedules {
		names = append(names, s.Name)
	}
	slog.Info("digest started", "schedules", names, "timezone", r.cfg.Location.String(), "system", r.cfg.System)
}

// Stop ends the loop and waits for it.
func (r *Runner) Stop() {
	if r.cancel != nil {
		r.cancel()
		<-r.done
	}
}

// RunOnce pitches every digest that is due now.
func (r *Runner) RunOnce(ctx context.Context) {
	for _, s := range r.cfg.Schedules {
		if err := r.pitchDue(ctx, s); err != nil {
			slog.Warn("digest not pitched, retrying next minute", "schedule", s.Name, "error", err)
		}
	}
}

// pitchDue pitches the digest of s's latest window unless it is already
// claimed or too late.
func (r *Runner) pitchDue(ctx context.Context, s Schedule) error {
	now := r.now()
	end := s.LastEnd(now, r.cfg.Location)
	if now.Sub(end) > s.MaxLateness() {
		return nil
	}

	key := "scout:digest:" + s.Name + ":" + strconv.FormatInt(end.Unix(), 10)
	claimed, err := r.store.Claim(ctx, key, claimTTL)
	if err != nil {
		return fmt.Errorf("claim %s: %w", key, err)
	}
	if !claimed {
		return nil
	}

	msg, _, err := r.Build(ctx, s, end)
	if err == nil {
		err = r.pitcher.Pitch(ctx, msg)
	}
	if err != nil {
		if rerr := r.store.Release(ctx, key); rerr != nil {
			slog.Warn("digest: releasing claim failed", "key", key, "error", rerr)
		}
		return err
	}
	slog.Info("digest pitched", "schedule", s.Name, "end", end.Format(time.RFC3339), "title", msg.Title, "severity", msg.Severity)
	return nil
}

// Report is a digest with the summaries it was built from.
type Report struct {
	Schedule string   `json:"schedule"`
	Message  Message  `json:"message"`
	Current  Summary  `json:"current"`
	Previous *Summary `json:"previous,omitempty"`
}

// Build computes the digest of s for the window ending at end.
func (r *Runner) Build(ctx context.Context, s Schedule, end time.Time) (Message, Report, error) {
	exclude := append([]string{r.cfg.System}, r.cfg.Exclude...)

	start := s.Start(end)
	rows, err := r.querier.Window(ctx, start, end)
	if err != nil {
		return Message{}, Report{}, err
	}
	cur := Summarize(rows, start, end, exclude, r.cfg.TopSystems)

	var previous *Summary
	prevStart := s.Start(start)
	if r.cfg.Retention <= 0 || !prevStart.Before(r.now().Add(-r.cfg.Retention-s.MaxLateness())) {
		prevRows, err := r.querier.Window(ctx, prevStart, start)
		if err != nil {
			return Message{}, Report{}, err
		}
		prev := Summarize(prevRows, prevStart, start, exclude, r.cfg.TopSystems)
		previous = &prev
	}

	msg := Format(s, cur, previous, r.cfg.System, r.cfg.Location)
	return msg, Report{Schedule: s.Name, Message: msg, Current: cur, Previous: previous}, nil
}

// Preview builds the digest of s for the window ending now, without pitching
// it.
func (r *Runner) Preview(ctx context.Context, s Schedule) (Report, error) {
	_, report, err := r.Build(ctx, s, r.now())
	return report, err
}

// ErrNoNumericTimestamp is returned for an index that cannot answer a time
// window: RediSearch answers a range query on an undeclared field with no
// results rather than an error, which would read as a quiet window.
var ErrNoNumericTimestamp = errors.New("redisearch index has no NUMERIC " + aggregator.TimestampUnixField +
	": recreate it (FT.DROPINDEX without DD, then restart scout)")

// RedisQuerier answers windows with FT.AGGREGATE over the messages index.
type RedisQuerier struct {
	Client *redis.Client
	Index  string
}

// Window implements Querier.
func (q RedisQuerier) Window(ctx context.Context, start, end time.Time) ([]Row, error) {
	info, err := q.Client.Do(ctx, "FT.INFO", q.Index).Result()
	if err != nil {
		return nil, fmt.Errorf("FT.INFO %s: %w", q.Index, err)
	}
	if !aggregator.HasNumericAttribute(info, aggregator.TimestampUnixField) {
		return nil, ErrNoNumericTimestamp
	}

	// LIMIT caps the reply explicitly: a window has one group per system and
	// severity, far below it.
	result, err := q.Client.Do(ctx,
		"FT.AGGREGATE", q.Index,
		fmt.Sprintf("@%s:[%d (%d]", aggregator.TimestampUnixField, start.Unix(), end.Unix()),
		"GROUPBY", "2", "@system", "@severity",
		"REDUCE", "COUNT", "0", "AS", "count",
		"LIMIT", "0", "10000",
		"TIMEOUT", "30000",
	).Result()
	if err != nil {
		return nil, fmt.Errorf("FT.AGGREGATE %s: %w", q.Index, err)
	}
	return parseRows(result), nil
}

// parseRows reads a RESP2 FT.AGGREGATE reply: [count, [field, value, ...], ...].
func parseRows(result any) []Row {
	arr, ok := result.([]any)
	if !ok {
		return nil
	}
	var rows []Row
	for _, item := range arr[min(1, len(arr)):] {
		fields, ok := item.([]any)
		if !ok {
			continue
		}
		var row Row
		for i := 0; i+1 < len(fields); i += 2 {
			name, _ := fields[i].(string)
			value, _ := fields[i+1].(string)
			switch name {
			case "system":
				row.System = value
			case "severity":
				row.Severity = value
			case "count":
				row.Count, _ = strconv.ParseInt(value, 10, 64)
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// RedisStore claims digests with SET NX.
type RedisStore struct {
	Client *redis.Client
}

// Claim implements Store.
func (s RedisStore) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return s.Client.SetNX(ctx, key, time.Now().UTC().Format(time.RFC3339), ttl).Result()
}

// Release implements Store.
func (s RedisStore) Release(ctx context.Context, key string) error {
	return s.Client.Del(ctx, key).Err()
}

// HTTPPitcher posts digests to omni-pitcher's /pitch.
type HTTPPitcher struct {
	URL    string // omni-pitcher base URL; /pitch is appended
	Token  string
	Client *http.Client
}

// Pitch implements Pitcher.
func (p HTTPPitcher) Pitch(ctx context.Context, msg Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL+"/pitch", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.Token)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pitch digest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("pitch digest: omni-pitcher answered %d", resp.StatusCode)
	}
	return nil
}
