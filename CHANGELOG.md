## [0.4.1](https://github.com/stuttgart-things/homerun2-scout/compare/v0.4.0...v0.4.1) (2026-03-12)


### Bug Fixes

* force RESP2 protocol for RediSearch FT.AGGREGATE compatibility ([4df847f](https://github.com/stuttgart-things/homerun2-scout/commit/4df847f17b3ade01ddbd7bcbf87053d26d3ddfa3)), closes [#30](https://github.com/stuttgart-things/homerun2-scout/issues/30)

# [0.4.0](https://github.com/stuttgart-things/homerun2-scout/compare/v0.3.0...v0.4.0) (2026-03-12)


### Features

* ensure RediSearch index exists on startup ([5217ef4](https://github.com/stuttgart-things/homerun2-scout/commit/5217ef45a86f501bdd806af6c6baa8a4193737c3)), closes [#28](https://github.com/stuttgart-things/homerun2-scout/issues/28)

# [0.3.0](https://github.com/stuttgart-things/homerun2-scout/compare/v0.2.1...v0.3.0) (2026-03-11)


### Features

* expose analytics as Prometheus metrics for Grafana integration ([86daf90](https://github.com/stuttgart-things/homerun2-scout/commit/86daf90807f66faed173891de6f3be9b3b758295)), closes [#23](https://github.com/stuttgart-things/homerun2-scout/issues/23)

## [0.2.1](https://github.com/stuttgart-things/homerun2-scout/compare/v0.2.0...v0.2.1) (2026-03-11)


### Bug Fixes

* allow main branch to deploy github-pages and opt into Node.js 24 ([0b637e1](https://github.com/stuttgart-things/homerun2-scout/commit/0b637e1c1c6d42a5986eb0dfad77723296eed448))

# [0.2.0](https://github.com/stuttgart-things/homerun2-scout/compare/v0.1.0...v0.2.0) (2026-03-11)


### Features

* ScoutProfile CRD for business logic configuration ([444b037](https://github.com/stuttgart-things/homerun2-scout/commit/444b037ea870be59ca84bd68007a50980c2e4ca3)), closes [#19](https://github.com/stuttgart-things/homerun2-scout/issues/19)

# [0.1.0](https://github.com/stuttgart-things/homerun2-scout/compare/v0.0.0...v0.1.0) (2026-03-11)


### Bug Fixes

* add dagger module go.mod to fix CI lint ([f9ed22e](https://github.com/stuttgart-things/homerun2-scout/commit/f9ed22e0e80171282e4e825ccb56303f27439f84))
* downgrade go directive to 1.25.0 for Dagger compatibility ([f5abf54](https://github.com/stuttgart-things/homerun2-scout/commit/f5abf54e597fc4608d417e183acaf10be491edad))
* remove remaining unnecessary blank identifiers in queries.go ([c9c19b5](https://github.com/stuttgart-things/homerun2-scout/commit/c9c19b5f7176a08357776f01f67588641dd30846))
* resolve golangci-lint errors (errcheck, staticcheck) ([c305774](https://github.com/stuttgart-things/homerun2-scout/commit/c3057745cc0a4d188f832eef08eeac399b8c50ce))


### Features

* aggregator with periodic FT.AGGREGATE queries ([20992c0](https://github.com/stuttgart-things/homerun2-scout/commit/20992c060878e59c868374467bee8308d40bf529)), closes [#3](https://github.com/stuttgart-things/homerun2-scout/issues/3)
* config loading, logging setup, and analytics models ([5f0936c](https://github.com/stuttgart-things/homerun2-scout/commit/5f0936c2d0cae76625fac18ae9268df033a9b016)), closes [#2](https://github.com/stuttgart-things/homerun2-scout/issues/2)
* KCL Kubernetes manifests for homerun2-scout ([e03deb7](https://github.com/stuttgart-things/homerun2-scout/commit/e03deb7fbdfaf3a129ab5a6caf16ce75fbb4dba4)), closes [#6](https://github.com/stuttgart-things/homerun2-scout/issues/6)
* main.go entrypoint wiring server and aggregator ([4c969b1](https://github.com/stuttgart-things/homerun2-scout/commit/4c969b13695fa9a53c518465d55afbf8d280783e)), closes [#5](https://github.com/stuttgart-things/homerun2-scout/issues/5)
* Prometheus metrics exposition endpoint ([feefa80](https://github.com/stuttgart-things/homerun2-scout/commit/feefa80a43f60e85f30422b681818c2e98c69781)), closes [#10](https://github.com/stuttgart-things/homerun2-scout/issues/10)
* REST API handlers and auth middleware ([7cfff9e](https://github.com/stuttgart-things/homerun2-scout/commit/7cfff9e3d4f19aefd83d098c5c30e172986a7033)), closes [#4](https://github.com/stuttgart-things/homerun2-scout/issues/4)
* retention/cleanup — prune old RediSearch entries by TTL ([ec21173](https://github.com/stuttgart-things/homerun2-scout/commit/ec21173304863c9562f9d482f505ae0bcb868a0a)), closes [#9](https://github.com/stuttgart-things/homerun2-scout/issues/9)
* threshold alerting — detect anomalies and pitch meta-alerts ([a7ebda9](https://github.com/stuttgart-things/homerun2-scout/commit/a7ebda95b425c1d3e0959bb32cbcc62bf77a461d)), closes [#8](https://github.com/stuttgart-things/homerun2-scout/issues/8) [#9](https://github.com/stuttgart-things/homerun2-scout/issues/9)

# 1.0.0 (2026-03-11)


### Bug Fixes

* add dagger module go.mod to fix CI lint ([f9ed22e](https://github.com/stuttgart-things/homerun2-scout/commit/f9ed22e0e80171282e4e825ccb56303f27439f84))
* downgrade go directive to 1.25.0 for Dagger compatibility ([f5abf54](https://github.com/stuttgart-things/homerun2-scout/commit/f5abf54e597fc4608d417e183acaf10be491edad))
* remove remaining unnecessary blank identifiers in queries.go ([c9c19b5](https://github.com/stuttgart-things/homerun2-scout/commit/c9c19b5f7176a08357776f01f67588641dd30846))
* resolve golangci-lint errors (errcheck, staticcheck) ([c305774](https://github.com/stuttgart-things/homerun2-scout/commit/c3057745cc0a4d188f832eef08eeac399b8c50ce))


### Features

* aggregator with periodic FT.AGGREGATE queries ([20992c0](https://github.com/stuttgart-things/homerun2-scout/commit/20992c060878e59c868374467bee8308d40bf529)), closes [#3](https://github.com/stuttgart-things/homerun2-scout/issues/3)
* config loading, logging setup, and analytics models ([5f0936c](https://github.com/stuttgart-things/homerun2-scout/commit/5f0936c2d0cae76625fac18ae9268df033a9b016)), closes [#2](https://github.com/stuttgart-things/homerun2-scout/issues/2)
* KCL Kubernetes manifests for homerun2-scout ([e03deb7](https://github.com/stuttgart-things/homerun2-scout/commit/e03deb7fbdfaf3a129ab5a6caf16ce75fbb4dba4)), closes [#6](https://github.com/stuttgart-things/homerun2-scout/issues/6)
* main.go entrypoint wiring server and aggregator ([4c969b1](https://github.com/stuttgart-things/homerun2-scout/commit/4c969b13695fa9a53c518465d55afbf8d280783e)), closes [#5](https://github.com/stuttgart-things/homerun2-scout/issues/5)
* project scaffolding with infra boilerplate from omni-pitcher ([b59cd32](https://github.com/stuttgart-things/homerun2-scout/commit/b59cd329e507aa4429c76fa7abe4b42b53d01b4b)), closes [#1](https://github.com/stuttgart-things/homerun2-scout/issues/1)
* Prometheus metrics exposition endpoint ([feefa80](https://github.com/stuttgart-things/homerun2-scout/commit/feefa80a43f60e85f30422b681818c2e98c69781)), closes [#10](https://github.com/stuttgart-things/homerun2-scout/issues/10)
* REST API handlers and auth middleware ([7cfff9e](https://github.com/stuttgart-things/homerun2-scout/commit/7cfff9e3d4f19aefd83d098c5c30e172986a7033)), closes [#4](https://github.com/stuttgart-things/homerun2-scout/issues/4)
* retention/cleanup — prune old RediSearch entries by TTL ([ec21173](https://github.com/stuttgart-things/homerun2-scout/commit/ec21173304863c9562f9d482f505ae0bcb868a0a)), closes [#9](https://github.com/stuttgart-things/homerun2-scout/issues/9)
* threshold alerting — detect anomalies and pitch meta-alerts ([a7ebda9](https://github.com/stuttgart-things/homerun2-scout/commit/a7ebda95b425c1d3e0959bb32cbcc62bf77a461d)), closes [#8](https://github.com/stuttgart-things/homerun2-scout/issues/8) [#9](https://github.com/stuttgart-things/homerun2-scout/issues/9)

# 1.0.0 (2026-03-11)


### Bug Fixes

* add dagger module go.mod to fix CI lint ([f9ed22e](https://github.com/stuttgart-things/homerun2-scout/commit/f9ed22e0e80171282e4e825ccb56303f27439f84))
* downgrade go directive to 1.25.0 for Dagger compatibility ([f5abf54](https://github.com/stuttgart-things/homerun2-scout/commit/f5abf54e597fc4608d417e183acaf10be491edad))
* remove remaining unnecessary blank identifiers in queries.go ([c9c19b5](https://github.com/stuttgart-things/homerun2-scout/commit/c9c19b5f7176a08357776f01f67588641dd30846))
* resolve golangci-lint errors (errcheck, staticcheck) ([c305774](https://github.com/stuttgart-things/homerun2-scout/commit/c3057745cc0a4d188f832eef08eeac399b8c50ce))


### Features

* aggregator with periodic FT.AGGREGATE queries ([20992c0](https://github.com/stuttgart-things/homerun2-scout/commit/20992c060878e59c868374467bee8308d40bf529)), closes [#3](https://github.com/stuttgart-things/homerun2-scout/issues/3)
* config loading, logging setup, and analytics models ([5f0936c](https://github.com/stuttgart-things/homerun2-scout/commit/5f0936c2d0cae76625fac18ae9268df033a9b016)), closes [#2](https://github.com/stuttgart-things/homerun2-scout/issues/2)
* KCL Kubernetes manifests for homerun2-scout ([e03deb7](https://github.com/stuttgart-things/homerun2-scout/commit/e03deb7fbdfaf3a129ab5a6caf16ce75fbb4dba4)), closes [#6](https://github.com/stuttgart-things/homerun2-scout/issues/6)
* main.go entrypoint wiring server and aggregator ([4c969b1](https://github.com/stuttgart-things/homerun2-scout/commit/4c969b13695fa9a53c518465d55afbf8d280783e)), closes [#5](https://github.com/stuttgart-things/homerun2-scout/issues/5)
* project scaffolding with infra boilerplate from omni-pitcher ([b59cd32](https://github.com/stuttgart-things/homerun2-scout/commit/b59cd329e507aa4429c76fa7abe4b42b53d01b4b)), closes [#1](https://github.com/stuttgart-things/homerun2-scout/issues/1)
* Prometheus metrics exposition endpoint ([feefa80](https://github.com/stuttgart-things/homerun2-scout/commit/feefa80a43f60e85f30422b681818c2e98c69781)), closes [#10](https://github.com/stuttgart-things/homerun2-scout/issues/10)
* REST API handlers and auth middleware ([7cfff9e](https://github.com/stuttgart-things/homerun2-scout/commit/7cfff9e3d4f19aefd83d098c5c30e172986a7033)), closes [#4](https://github.com/stuttgart-things/homerun2-scout/issues/4)
* retention/cleanup — prune old RediSearch entries by TTL ([ec21173](https://github.com/stuttgart-things/homerun2-scout/commit/ec21173304863c9562f9d482f505ae0bcb868a0a)), closes [#9](https://github.com/stuttgart-things/homerun2-scout/issues/9)
* threshold alerting — detect anomalies and pitch meta-alerts ([a7ebda9](https://github.com/stuttgart-things/homerun2-scout/commit/a7ebda95b425c1d3e0959bb32cbcc62bf77a461d)), closes [#8](https://github.com/stuttgart-things/homerun2-scout/issues/8) [#9](https://github.com/stuttgart-things/homerun2-scout/issues/9)
