package digest

import "strings"

// Excluded reports whether system is left out of the digest: an entry of
// exclude matches it exactly, or, ending in "*", as a prefix (#93) --
// "vm-*" covers vm-sthings-app, vm-app-dev and every VM to come.
func Excluded(system string, exclude []string) bool {
	for _, e := range exclude {
		if prefix, ok := strings.CutSuffix(e, "*"); ok {
			if strings.HasPrefix(system, prefix) {
				return true
			}
			continue
		}
		if system == e {
			return true
		}
	}
	return false
}
