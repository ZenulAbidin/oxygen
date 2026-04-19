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
  - A baseline git commit for this autonomous session was created at `98be4bc` (`chore: baseline before autonomous work`).
  - Docker is unavailable here.
  - `golangci-lint` is not installed.
  - Local Postgres is not reachable on `127.0.0.1:5432`.
  - Direct `go test` invocations still need an executable temp directory (`TMPDIR=/workspace/oxygen/tmp/go` here), but the repo-native `make build` and `make test` commands now handle that automatically.
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
- [blocked] test/build/lint/type failure: full backend integration tests require reachable Postgres or a valid `OXYGEN_TEST_DB_DATA_SOURCE`.
- [blocked] test/build/lint/type failure: backend lint requires `golangci-lint`, which is not installed in this workspace.
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
- Targeted unfinished-work verification:
  - `rg -n 't\\.Skip|Skip\\(|not implemented yet|panic\\(\".*TODO|TODO:' ...` -> only surfaced generated-client TODOs, an upstream Ant Design workaround, and the BTC broadcaster stub.
  - `rg -n 'BTC|bitcoin|btc' ...` plus inspection of `internal/service/blockchain/currencies.json` and `ui-dashboard/src/types/index.ts` -> BTC is present in lower-level wallet/OpenAPI surfaces but not in the active supported-currency set used by merchant/payment flows.
  - `rg -n --glob '!pkg/api-*' --glob '!web/redoc/**' --glob '!**/package-lock.json' --glob '!**/*.test.*' --glob '!internal/test/**' --glob '!*_test.go' 'TODO|FIXME|XXX|HACK|not implemented|placeholder|stub|mock-only|throw new Error|panic\\(' cmd internal ui-dashboard/src ui-payment/src web` -> only surfaced expected guard/setup panics, the known Ant Design workaround TODO, and the BTC broadcaster stub outside the active supported-currency set.

## Unresolved Blockers

- No reachable Postgres service in this workspace.
- Docker is unavailable.
- `golangci-lint` is unavailable.
- Real blockchain/provider credentials are unavailable.

## Scope Notes

- Focus on repository-supported flows and operational completeness, not new roadmap features.
- Remaining work is blocked by missing local services/tools rather than clearly justified in-repo feature or flow gaps.
- Anti-premature-stop check for this iteration:
  - Project type and stack verified from README, Makefiles, workflow files, manifests, and code layout.
  - Intended dev workflow verified from CI and documented commands.
  - Unfinished-work markers searched again after the baseline commit.
  - Major locally runnable flows validated again on the current baseline: backend build plus both frontend lint/build pipelines; backend tests still fail only on the documented Postgres dependency.
  - Remaining gaps are either environment-blocked (`Postgres`, `Docker`, `golangci-lint`, provider credentials) or outside the current supported product scope.
