# Codex Progress

## Repository Assessment

- Inferred project type: mixed system; Go backend plus two Vite/React frontends for a self-hosted crypto payment gateway (`OxygenPay`).
- Inferred product scope: merchant dashboard, hosted payment pages, payment links, balances, withdrawals, wallets/KMS, webhooks, and scheduler-backed processing.
- Intended users: merchants/operators accepting crypto payments and administrators running the service self-hosted.
- Current maturity: substantial implementation exists; the repo is archived upstream, and the remaining justified work is operational confidence rather than missing major product surfaces.

## Environment / Tooling Understanding

- Languages: Go (repo targets `1.20`; local toolchain reports `go1.26.2`) and TypeScript/React.
- Backend framework/tooling: Cobra CLI, Echo server, Go modules, sql-migrate, sqlc, go-swagger, Makefile-driven workflows.
- Frontend tooling: Vite, TypeScript, ESLint, Prettier, npm with committed lockfiles.
- Package/build systems: single Go module plus two frontend packages in `ui-dashboard/` and `ui-payment/`.
- Runtime/deployment: Dockerfile plus `docker-compose.local.yml`; Postgres required; KMS can run embedded or standalone.
- Workspace constraints:
  - A baseline git commit for this autonomous session was created at `85ce575` (`chore: baseline before autonomous work`).
  - Docker is unavailable here.
  - `golangci-lint` is not installed, and the repo-pinned `v1.53.3` binary must be bootstrapped with a Go `1.20` toolchain on newer hosts.
  - Local Postgres is not reachable on `127.0.0.1:5432`.
  - Direct `go test` invocations still need an executable temp directory (`TMPDIR=/workspace/oxygen/tmp/go` here), but the repo-native `make build` and `make test` commands now handle that automatically.
  - Further ad hoc Go-based environment bootstrap in this workspace now intermittently stalls in uninterruptible I/O (`D`) state even for simple helper-module commands like `go env` and `go build`, which limits additional backend validation attempts that are not already covered by repo-native commands.
  - Real blockchain/provider flows still require external credentials (`TATUM_*`, `TRONGRID_*`).

## Likely Validation Commands

- Backend build: `make build`
- Backend tests: `make test`
- Backend lint: `make lint`
- Dashboard UI:
  - `cd ui-dashboard && npm ci`
  - `cd ui-dashboard && make lint`
  - `cd ui-dashboard && make build`
- Payment UI:
  - `cd ui-payment && npm ci --ignore-scripts`
  - `cd ui-payment && make lint`
  - `cd ui-payment && make build`
- Local runtime:
  - `docker-compose -f docker-compose.local.yml up`
  - or `./bin/oxygen serve-web --config=$(pwd)/config/oxygen.yml`
- Backend test DB override:
  - `OXYGEN_TEST_DB_DATA_SOURCE=postgres://... TMPDIR=/workspace/oxygen/tmp/go go test ./...`

## Backlog

- [done] developer experience issue affecting completion: local run commands fall back to the shipped `config/oxygen.example.yml`, and README documents the config path.
- [done] developer experience issue affecting completion: top-level Makefile no longer eagerly runs `go list` for unrelated targets like `make help` and `make run`.
- [done] developer experience issue affecting completion: frontend Makefiles work without a committed `.env` by falling back to `.env.example`.
- [done] developer experience issue affecting completion: added `docker.env.example` and documented the Docker env-file workflow.
- [done] test/build/lint/type failure: both frontends pass repo-native `make lint` and `make build`.
- [done] developer experience issue affecting completion: backend test DB setup now supports `OXYGEN_TEST_DB_DATA_SOURCE`, uses a fast connection timeout, reports clear Postgres connection errors, and exposes a focused config-resolution unit test.
- [done] developer experience issue affecting completion: backend `make build` and `make test` now use a repo-local Go temp directory, so validations are not blocked by a `noexec` host `/tmp`.
- [done] unfinished-work search: targeted search found only a UI workaround TODO and a BTC broadcast stub; BTC broadcast is out of current scope because shipped currencies do not include BTC.
- [done] test/build/lint/type failure: re-ran repo-native backend/frontend build validation on the current baseline and all locally runnable build/lint checks passed.
- [done] unfinished-work search: repeated route/manifests/code scanning confirmed no remaining in-scope local work beyond environment-blocked integration/lint/runtime validation.
- [done] developer experience issue affecting completion: `make require-deps` and `make lint` now export the repo-targeted Go `1.20.14` toolchain plus `GOSUMDB=sum.golang.org` for `golangci-lint v1.53.3`, and README documents that repo-native lint setup on newer hosts.
- [blocked] test/build/lint/type failure: full backend integration tests require reachable Postgres or a valid `OXYGEN_TEST_DB_DATA_SOURCE`.
- [blocked] test/build/lint/type failure: full backend lint still requires a local `golangci-lint v1.53.3` binary and a long analysis window; the toolchain/panic issue is fixed, but a bounded 10-minute local run did not finish in this workspace.
- [blocked] developer experience issue affecting completion: an isolated embedded-Postgres bootstrap attempt via a temporary nested Go module was not viable in this workspace because subsequent `go env` / `go build` commands stalled in host-level I/O wait before a local Postgres could be launched.
- [blocked] broken flow: full end-to-end payment processing and blockchain provider flows require real provider credentials/services.
- [blocked] broken flow: Docker-based runtime validation is unavailable because Docker is not installed here.
- [out_of_scope] missing in-scope feature: README roadmap items that are not already represented by code or broken product paths.
- [out_of_scope] polish: generated/OpenAPI enum surfaces still mention `BTC`, but runtime supported currencies come from `internal/service/blockchain/currencies.json` and the active dashboard/payment flows expose only `ETH`, `TRON`, `MATIC`, `BNB`, `USDT`, `USDC`, and deprecated `BUSD`.

## Validations Attempted

- `make help` -> passed.
- `make -n run` -> passed and resolves to `config/oxygen.example.yml` without triggering `go list`.
- `cd ui-dashboard && make lint` -> passed.
- `cd ui-payment && make lint` -> passed.
- `cd ui-dashboard && make build` -> passed.
- `cd ui-payment && make build` -> passed.
- `go version` -> passed (`go version go1.26.2 linux/amd64`).
- `golangci-lint version` -> failed (`golangci-lint: command not found`).
- `TMPDIR=/workspace/.tmp-go go test ./internal/test -run TestTestDatabaseConfig -count=1 -timeout=120s` -> passed.
- `TMPDIR=/workspace/.tmp-go go test ./internal/test -run TestNewDB -count=1 -timeout=30s` -> failed quickly with a clear `127.0.0.1:5432` connection-refused error and guidance to use `OXYGEN_TEST_DB_DATA_SOURCE`.
- `TMPDIR=/workspace/.tmp-go make build` -> passed.
- `TMPDIR=/workspace/.tmp-go make test` -> started a full `go test -race` sweep; not useful to keep waiting once the focused Postgres-dependent failure mode had already been confirmed.
- `git commit -m "chore: baseline before autonomous work"` -> passed (`98be4bc`).
- `TMPDIR=/workspace/.tmp-go make build` -> passed on the current baseline.
- `cd ui-dashboard && make lint && make build` -> passed on the current baseline.
- `cd ui-payment && make lint && make build` -> passed on the current baseline.
- `go test ./internal/server/http/paymentapi -run Test -count=1` -> failed in this workspace before execution because `/tmp` is `noexec` (`fork/exec /tmp/go-build.../paymentapi.test: permission denied`).
- `go test ./internal/server/http/merchantapi -run Test -count=1` -> failed in this workspace before execution because `/tmp` is `noexec` (`fork/exec /tmp/go-build.../merchantapi.test: permission denied`).
- `go test ./internal/server/http/webhook -run Test -count=1` -> failed in this workspace before execution because `/tmp` is `noexec` (`fork/exec /tmp/go-build.../webhook.test: permission denied`).
- `make help` -> passed after the Makefile temp-dir change.
- `make build` -> passed after the Makefile temp-dir change and now uses `TMPDIR="/workspace/oxygen/tmp/go"` internally.
- `TMPDIR=/workspace/oxygen/tmp/go go test ./internal/server/http/paymentapi -run Test -count=1` -> failed quickly with the expected Postgres connection-refused error.
- `TMPDIR=/workspace/oxygen/tmp/go go test ./internal/server/http/merchantapi -run Test -count=1` -> failed quickly with the expected Postgres connection-refused error.
- `TMPDIR=/workspace/oxygen/tmp/go go test ./internal/server/http/webhook -run Test -count=1` -> failed quickly with the expected Postgres connection-refused error.
- `make test` -> now runs with `TMPDIR="/workspace/oxygen/tmp/go"`; focused package tests confirm the remaining failure mode is missing Postgres rather than temp-binary execution.
- `make help` -> passed on the current baseline.
- `make build` -> passed on the current baseline.
- `make lint` -> failed because `golangci-lint` is not installed in this workspace (`make: golangci-lint: No such file or directory`).
- `docker --version` -> failed because Docker is not installed in this workspace (`docker: command not found`).
- `cd ui-dashboard && make lint && make build` -> passed again on the current baseline.
- `cd ui-payment && make lint && make build` -> passed again on the current baseline.
- `TMPDIR=/workspace/oxygen/tmp/go go test ./internal/test -run TestTestDatabaseConfig -count=1 -timeout=120s` -> passed.
- `TMPDIR=/workspace/oxygen/tmp/go go test ./internal/server/http/paymentapi -run Test -count=1 -timeout=60s` -> failed quickly with the expected Postgres connection-refused error and the improved guidance to use `OXYGEN_TEST_DB_DATA_SOURCE`.
- `git commit -m "chore: baseline before autonomous work"` -> passed for this session (`85ce575`).
- `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3` -> failed under host `go1.26.2` with `invalid array length -delta * delta` from `golang.org/x/tools/internal/tokeninternal`.
- `GOTOOLCHAIN=go1.20.14 go version` -> passed (`go version go1.20.14 linux/amd64`).
- `GOTOOLCHAIN=go1.20.14 TMPDIR=/workspace/oxygen/tmp/go GOBIN=/workspace/oxygen/tmp/bin go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3` -> started successfully under the repo-targeted toolchain but remained long-running in this workspace, so the README was updated to document the required toolchain instead of treating lint bootstrap as a product bug.
- `timeout 300s bash -lc 'TMPDIR=/workspace/oxygen/tmp/go GOBIN=/workspace/oxygen/tmp/bin GOTOOLCHAIN=go1.20.14 go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3'` -> passed and produced `/workspace/oxygen/tmp/bin/golangci-lint`.
- `/workspace/oxygen/tmp/bin/golangci-lint version` -> passed (`v1.53.3` built with `go1.20.14`).
- `PATH=/workspace/oxygen/tmp/bin:$PATH make lint` -> failed with `unsupported version: 2`, which confirmed the lint runtime also needed the repo-targeted Go toolchain.
- `PATH=/workspace/oxygen/tmp/bin:$PATH GOTOOLCHAIN=go1.20.14 make lint` -> failed quickly because this workspace had no `GOSUMDB` configured for toolchain verification.
- `make -n lint` -> passed and now expands to `GOTOOLCHAIN=go1.20.14 GOSUMDB=sum.golang.org golangci-lint run -v ./...`.
- `make -n require-deps` -> passed and now expands the repo-native lint bootstrap command as `GOTOOLCHAIN=go1.20.14 GOSUMDB=sum.golang.org go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.53.3`.
- `timeout 600s bash -lc 'PATH=\"/workspace/oxygen/tmp/bin:$PATH\" GOTOOLCHAIN=go1.20.14 GOSUMDB=sum.golang.org make lint'` -> no longer panicked and reached real analysis, but timed out after 10 minutes with no lint findings emitted in this workspace.
- `PATH=/workspace/oxygen/tmp/bin:$PATH GOTOOLCHAIN=go1.20.14 GOSUMDB=sum.golang.org timeout 1200s make lint` -> progressed past config/package loading, then stalled in host-level I/O wait without emitting findings; the stuck process was terminated rather than left running indefinitely.
- Temporary nested-module environment bootstrap under `.codex/embedded-pg-helper/` -> attempted to launch an isolated embedded Postgres for backend tests, but `go run`, `go env`, and `go build` in that helper module stalled in uninterruptible I/O state before any local Postgres process or bootstrap artifacts were created, so the helper was removed and the attempt was recorded as an environment blocker.
- Targeted unfinished-work verification:
  - `rg -n 't\\.Skip|Skip\\(|not implemented yet|panic\\(\".*TODO|TODO:' ...` -> only surfaced generated-client TODOs, an upstream Ant Design workaround, and the BTC broadcaster stub.
  - `rg -n 'BTC|bitcoin|btc' ...` plus inspection of `internal/service/blockchain/currencies.json` and `ui-dashboard/src/types/index.ts` -> BTC is present in lower-level wallet/OpenAPI surfaces but not in the active supported-currency set used by merchant/payment flows.
  - `rg -n --glob '!pkg/api-*' --glob '!web/redoc/**' --glob '!**/package-lock.json' --glob '!**/*.test.*' --glob '!internal/test/**' --glob '!*_test.go' 'TODO|FIXME|XXX|HACK|not implemented|placeholder|stub|mock-only|throw new Error|panic\\(' cmd internal ui-dashboard/src ui-payment/src web` -> only surfaced expected guard/setup panics, the known Ant Design workaround TODO, and the BTC broadcaster stub outside the active supported-currency set.
- `make help` -> passed again on this iteration.
- `make build` -> started from the repo-native command on this iteration but entered the same long-running no-output state seen in prior workspace I/O stalls, so it was terminated rather than left hanging indefinitely.
- `cd ui-dashboard && make lint && make build` -> started on this iteration but was terminated after the same long-running no-output workspace behavior; earlier successful baseline runs still stand as the last completed local frontend validation.
- `cd ui-payment && make lint && make build` -> started on this iteration but was terminated after the same long-running no-output workspace behavior; earlier successful baseline runs still stand as the last completed local frontend validation.
- `TMPDIR=/workspace/oxygen/tmp/go go test ./internal/test -run TestTestDatabaseConfig -count=1 -timeout=120s` -> passed again on this iteration.
- `TMPDIR=/workspace/oxygen/tmp/go go test ./internal/server/http/paymentapi -run Test -count=1 -timeout=60s` -> failed again in `0.296s` with the expected Postgres connection-refused panic and guidance to use `OXYGEN_TEST_DB_DATA_SOURCE`.
- `make -n lint` -> passed again on this iteration and still expands to `GOTOOLCHAIN=go1.20.14 GOSUMDB=sum.golang.org golangci-lint run -v ./...`.
- `docker-compose.local.yml` and `config/oxygen.example.yml` inspection -> confirmed the documented local topology still matches the repo: one app container, Postgres, file-based config fallback, and provider credentials supplied via env/config.

## Unresolved Blockers

- No reachable Postgres service in this workspace.
- Docker is unavailable.
- Full backend lint remains blocked by workspace behavior even after the toolchain fix; a longer follow-up run progressed further but then stalled in host-level I/O wait without producing findings.
- Additional local Postgres bootstrap attempts are blocked by the same workspace-level Go I/O stalls, so I could not safely turn the backend integration suite into a fully local validation flow from this container.
- Real blockchain/provider credentials are unavailable.

## Scope Notes

- Focus on repository-supported flows and operational completeness, not new roadmap features.
- Remaining work is blocked by missing local services/tools rather than clearly justified in-repo feature or flow gaps.
- A fresh iteration on `2026-04-19 13:37:03Z` revalidated the repo assessment, package manifests, CI commands, and unfinished-work markers; it did not surface a new in-scope product bug or missing feature, only deeper confirmation that the remaining backend gaps are environmental.
- A fresh iteration on `2026-04-19 13:41:31Z` rechecked the route surfaces, stack manifests, local config/runtime docs, unfinished-work markers, and focused Go tests; it again surfaced no new in-scope product bug or missing feature.
- Anti-premature-stop check for this iteration:
  - Project type and stack verified from README, Makefiles, workflow files, manifests, and code layout.
  - Intended dev workflow verified from CI and documented commands.
  - Unfinished-work markers searched again after the baseline commit.
  - Major locally runnable flows were rechecked on the current baseline, and no new in-scope repo change was justified beyond earlier completed DX fixes.
  - Remaining gaps are either environment-blocked (`Postgres`, `Docker`, workspace Go I/O stalls, provider credentials) or outside the current supported product scope.
