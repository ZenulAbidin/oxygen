### 2024-07-01

Hello, Internet 🌎 Despite the great journey filled with challenges and learnings, I haven’t found product market fit for this project. Given this, and my current lack of bandwidth and will to continue improving and maintaining it, I’ve arhivie it and move forward. However, I believe this project still holds potential, and I hope someone with the necessary passion and resources might take it forward. 

If you’re interested, feel free to fork the repo and continue its development.

Thank y'all for your support and understanding.

--- 

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
- Create payment links for predefined invoices
- Automatic hot wallets management
- Built-in KMS (Key Management Service) for securely storing wallet keys
- Nice and simple merchant dashboard; sleek payment UI
- Easy integration via [API](https://docs.o2pay.co/specs/merchant/v1/) or [webhooks](https://docs.o2pay.co/webhooks)
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
- `docker-compose.local.yml` expects a `docker.env` file; start from `docker.env.example` and replace the placeholder provider credentials.

### Validation

- Backend: `make build`, `make test`, `make lint`
- Dashboard UI: `cd ui-dashboard && npm ci && make lint && make build`
- Payment UI: `cd ui-payment && npm ci --ignore-scripts && make lint && make build`
- Backend tests expect Postgres to be reachable for integration tests. By default they use `127.0.0.1` as `postgres`; override with `OXYGEN_TEST_DB_DATA_SOURCE` when needed.


## Roadmap 🛣️

- [x] Support for USDC
- [x] Support for Binance Smart Chain (BNB, BUSD)
- [ ] Donations feature
- [ ] Support for [WalletConnect](https://walletconnect.com/)
- [ ] SDKs for (Python, JavaScript, PHP, etc...)
- [ ] Support for all major ETH Layer 2 Chains
- [ ] Support for blockchain notification providers other than Tatum
- [ ] Integration with DEXes for automatic swaps: convert incoming crypto to stablecoins

## License 📑

This software is licensed under [Apache License 2.0](./LICENSE).
