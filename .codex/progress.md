# Codex Progress

## Repository Assessment

- Inferred project type: mixed system; Go backend plus two Vite/React frontends for a self-hosted crypto payment gateway (`OxygenPay`).
- Inferred product scope: merchant dashboard, hosted payment pages, payment links, balances, withdrawals, wallets/KMS, webhooks, and scheduler-backed processing.
- Intended users: merchants/operators accepting crypto payments and administrators running the service self-hosted.
- Current maturity: substantial implementation exists, but the repo is archived and has some operational/documentation rough edges.

## Environment / Tooling Understanding

- Languages: Go 1.20, TypeScript/React.
- Backend framework/tooling: Cobra CLI, Echo server, sql-migrate, sqlc, go-swagger, golangci-lint.
- Frontend tooling: Vite, TypeScript, ESLint, Prettier, npm with `package-lock.json`.
- Package/build systems: Go modules, npm, Makefiles.
- Runtime/deployment: Dockerfile plus `docker-compose.local.yml`; Postgres required; KMS may run embedded or standalone.
- Monorepo shape: single Go module with two frontend packages (`ui-dashboard`, `ui-payment`).
- Config/runtime blockers detected:
  - Backend validation is blocked in this workspace because neither `go` nor Docker is installed.
  - External provider secrets are needed for real blockchain operations (`TATUM_*`, `TRONGRID_*`), though many tests appear to use fakes.

## Likely Validation Commands

- Backend build: `make build`
- Backend tests: `make test`
- Backend lint: `make lint`
- Dashboard install/lint/typecheck/build:
  - `cd ui-dashboard && npm ci`
  - `cd ui-dashboard && make lint`
  - `cd ui-dashboard && make build`
- Payment install/lint/typecheck/build:
  - `cd ui-payment && npm ci --ignore-scripts`
  - `cd ui-payment && make lint`
  - `cd ui-payment && make build`
- Local runtime:
  - `docker-compose -f docker-compose.local.yml up`
  - or `./bin/oxygen serve-web --config=$(pwd)/config/oxygen.yml` after preparing config

## Backlog

- [done] developer experience issue affecting completion: local run commands now fall back to the shipped `config/oxygen.example.yml`, and README documents the config path.
- [done] developer experience issue affecting completion: top-level Makefile no longer eagerly runs `go list` for unrelated targets like `make help` and `make run`.
- [done] developer experience issue affecting completion: frontend Makefiles now work without a committed `.env` by falling back to `.env.example`.
- [done] test/build/lint/type failure: both frontends pass repo-native `make lint` and `make build`.
- [done] developer experience issue affecting completion: added `docker.env.example` and documented the Docker env-file workflow.
- [done] unfinished-work search: targeted search found only a UI workaround TODO and BTC broadcast stub; BTC broadcast is out of current scope because the shipped currency set/README do not advertise BTC support.
- [blocked] test/build/lint/type failure: backend `make build` / `make test` / `make lint` cannot run here because `go` is unavailable and Docker is unavailable.
- [blocked] broken flow: full end-to-end payment processing and blockchain provider flows require real provider credentials/services.
- [out_of_scope] missing in-scope feature: roadmap items from README that are not already represented by code or broken paths.

## Validations Attempted

- `go test ./...` -> blocked: `go` command is not installed in this workspace.
- `go build -o /tmp/oxygen-testbuild ./main.go` -> blocked: `go` command is not installed in this workspace.
- `cd ui-dashboard && npm ci` -> passed.
- `cd ui-payment && npm ci --ignore-scripts` -> passed.
- `cd ui-dashboard && npx eslint src --ext .js,.jsx,.ts,.tsx` -> passed.
- `cd ui-dashboard && npx tsc --noEmit --skipLibCheck -p ./tsconfig.json` -> passed.
- `cd ui-payment && npx eslint src --ext .js,.jsx,.ts,.tsx` -> passed.
- `cd ui-payment && npx tsc --noEmit --skipLibCheck -p ./tsconfig.json` -> passed.
- `cd ui-payment && VITE_BACKEND_HOST='//' VITE_SUPPORT_EMAIL='help@site.com' VITE_ROOTPATH='/p/' VITE_SHOW_BRANDING='true' npx tsc && npx vite build --base=/p/` -> passed.
- `cd ui-dashboard && VITE_BACKEND_HOST='//' VITE_ROOTPATH='/dashboard/' npx tsc && npx vite build --base=/dashboard/` -> initially failed with Node heap OOM when the env was not exported to `vite`; passed after applying the Makefile/workflow fix.
- `cd ui-dashboard && make lint` -> passed.
- `cd ui-payment && make lint` -> passed.
- `cd ui-dashboard && make build` -> passed.
- `cd ui-payment && make build` -> passed.
- `make -n run` -> passed and resolves to `config/oxygen.example.yml` without triggering `go list`.
- `make help` -> passed without requiring `go`.

## Unresolved Blockers

- Local backend validation still requires a Go toolchain.
- Docker-based runtime validation still requires Docker.
- Real end-to-end blockchain flows may be blocked by absent provider credentials.

## Scope Notes

- Focus on repository-supported flows and operational completeness, not new roadmap features.
