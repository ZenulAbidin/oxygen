# OxygenPay SDKs

This directory contains lightweight SDKs for integrating server-side applications with the OxygenPay Merchant API.

Available SDKs:

- `python`
- `javascript`
- `php`

All SDKs support:

- `X-O2PAY-TOKEN` authentication
- Creating payments
- Listing and fetching payments
- Creating, listing, fetching, and deleting payment links
- Listing merchant balances
- Listing and fetching customers
- Webhook HMAC-SHA512 signature verification

The Merchant API base URL defaults to:

```text
https://api.o2pay.co/api/merchant/v1
```

For self-hosted Oxygen deployments, pass your own base URL, for example:

```text
http://localhost:8080/api/merchant/v1
```

API tokens are created from the Oxygen dashboard.

