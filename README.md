<p align="center">
  <a href="https://o2pay.co">
    <img src="./.github/static/cover.svg" height="200" alt="cover">
  </a>
</p>

**OxygenPay** is a cloud or self-hosted crypto payment gateway.
Receive crypto including stablecoins with ease. Open new opportunities for your product by accepting cryptocurrency.

<img src="./.github/static/demo.jpg" alt="demo">

## Supported Currencies 🔗

<table>
    <tr>
        <td align="center">
            <img src="./ui-dashboard/src/assets/icons/crypto/btc.svg" height="64" alt="btc">
            <div>Bitcoin</div>
        </td>
        <td align="center">
            <img src="./ui-dashboard/src/assets/icons/crypto/eth.svg" height="64" alt="eth">
            <div>Ethereum</div>
        </td>
        <td>
            <img src="./ui-dashboard/src/assets/icons/crypto/matic.svg" height="64" alt="matic">
            <div>Polygon</div>
        </td>
        <td align="center">
            <img src="./ui-dashboard/src/assets/icons/crypto/tron.svg" height="64" alt="tron">
            <div>TRON</div>
        </td>
        <td align="center">
            <img src="./ui-dashboard/src/assets/icons/crypto/bnb.svg" height="64" alt="bnb">
            <div>BNB</div>
        </td>
        <td align="center">
            <img src="./ui-dashboard/src/assets/icons/crypto/usdt.svg" height="64" alt="usdt">
            <div>USDT</div>
        </td>
        <td align="center">
            <img src="./ui-dashboard/src/assets/icons/crypto/usdc.svg" height="64" alt="usdc">
            <div>USDC</div>
        </td>
    </tr>
</table>

## Features ✨

- Self-hosted
- Non-custodial
- Built-in multi-tenancy
- Create payment links for predefined invoices and donations
- Automatic hot wallets management
- Built-in KMS (Key Management Service) for securely storing wallet keys
- Nice and simple merchant dashboard; sleek payment UI
- Easy integration via the local docs service: [API](http://localhost:8081/api/merchant.html) or [webhooks](http://localhost:8081/#webhooks)
- No need to setup full-nodes
- Support for testnets
- It's only 1 binary!

## Development

### Stack

- Go `1.20`
- Node.js + npm for the two Vite frontends in `ui-dashboard/` and `ui-payment/`
- Postgres for the main application database

### Local Config

- The repo ships `config/oxygen.example.yml` as the base config.
- Local `make run`, `make run-kms`, and `make run-scheduler` commands will use `config/oxygen.yml` when present and otherwise fall back to `config/oxygen.example.yml`.
- `docker-compose.local.yml` expects a `docker.env` file; start from `docker.env.example`. Provider credentials are optional for local startup.
- The local compose stack exposes Oxygen at `http://localhost:8080`.
- The local compose stack also exposes bundled documentation and API specs at `http://localhost:8081`.

### Validation

- Backend: `make build`, `make test`, `make lint`
- Install backend CLI tooling with `make require-deps`; `make lint` and the lint bootstrap use the repo-targeted Go `1.20.14` toolchain for `golangci-lint v1.53.3`, so newer hosts still match CI.
- Dashboard UI: `cd ui-dashboard && npm ci && make lint && make build`
- Payment UI: `cd ui-payment && npm ci --ignore-scripts && make lint && make build`
- Backend tests expect Postgres to be reachable for integration tests. By default they use `127.0.0.1` as `postgres`; override with `OXYGEN_TEST_DB_DATA_SOURCE` when needed.


## Roadmap 🛣️

- [x] Support for USDC
- [x] Support for Binance Smart Chain (BNB, BUSD)
- [x] Donations feature
- [ ] Support for [WalletConnect](https://walletconnect.com/)
- [ ] SDKs for (Python, JavaScript, PHP, etc...)
- [ ] Support for all major ETH Layer 2 Chains
- [ ] Support for blockchain notification providers other than Tatum
- [ ] Integration with DEXes for automatic swaps: convert incoming crypto to stablecoins

## License 📑

This software is licensed under [Apache License 2.0](./LICENSE).
