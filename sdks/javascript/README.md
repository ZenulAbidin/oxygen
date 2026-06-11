# OxygenPay JavaScript SDK

Server-side JavaScript client for the OxygenPay Merchant API.

Requires Node.js 18+ for global `fetch`.

## Usage

```js
import { randomUUID } from "node:crypto";
import { OxygenClient } from "@oxygenpay/sdk";

const client = new OxygenClient({
  apiToken: "op_live_...",
  merchantId: "merchant-uuid",
  baseUrl: "http://localhost:8080/api/merchant/v1",
});

const payment = await client.createPayment({
  id: randomUUID(),
  currency: "USD",
  price: 29.9,
  description: "White T-shirt size M",
  redirectUrl: "https://example.com/success",
});

console.log(payment.paymentUrl);
```

## Webhook Verification

Use the raw request body exactly as received.

```js
import { verifyWebhookSignature } from "@oxygenpay/sdk";

const ok = verifyWebhookSignature(rawBody, "webhook-secret", request.headers["x-signature"]);
```

