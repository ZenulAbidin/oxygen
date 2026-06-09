# Merchant webhooks

Oxygen sends an HTTP(S) `POST` request to the merchant webhook URL whenever a payment or withdrawal status changes.

## Headers

| Header | Description |
| --- | --- |
| `Content-Type` | Always `application/json`. |
| `X-Webhook-Id` | Stable event ID. Use it for idempotency and duplicate protection. |
| `X-Webhook-Event` | Event type, such as `payment.status_changed` or `withdrawal.status_changed`. |
| `X-Signature` | Base64-encoded HMAC-SHA512 of the raw request body, present when a webhook secret is configured. |

Verify `X-Signature` with the configured webhook secret before trusting the payload.

## Payment status payload

The top-level `id`, `status`, `customerEmail`, `selectedBlockchain`, `selectedCurrency`, `paymentLinkId`, and `isTest` fields are kept for backward compatibility. New integrations should prefer the nested objects.

```json
{
  "eventId": "payment:d790ec98-823c-11ed-a1eb-0242ac120002:success",
  "eventType": "payment.status_changed",
  "version": "1",
  "id": "d790ec98-823c-11ed-a1eb-0242ac120002",
  "orderId": "order#123",
  "type": "payment",
  "status": "success",
  "customerEmail": "john@doe.com",
  "selectedBlockchain": "ETH",
  "selectedCurrency": "ETH_USDT",
  "paymentLinkId": "8c857501-e67d-4a0b-98d9-46e461b42c09",
  "isTest": false,
  "occurredAt": "2022-12-22T19:12:55.386201Z",
  "payment": {
    "id": "d790ec98-823c-11ed-a1eb-0242ac120002",
    "publicId": "9d0a334f-af6d-4cbb-b0ca-118d3fb738bb",
    "orderId": "order#123",
    "type": "payment",
    "status": "success",
    "createdAt": "2022-12-22T19:02:55.386201Z",
    "updatedAt": "2022-12-22T19:12:55.386201Z",
    "amount": {
      "value": "29.9",
      "raw": "2990",
      "currency": "USD",
      "decimals": 2
    },
    "redirectUrl": "https://my-store.com/success",
    "paymentUrl": "https://pay.example.com/p/9d0a334f-af6d-4cbb-b0ca-118d3fb738bb",
    "description": "White T-shirt size M",
    "isTest": false
  },
  "customer": {
    "email": "john@doe.com"
  },
  "paymentMethod": {
    "blockchain": "ETH",
    "blockchainName": "Ethereum",
    "currency": "ETH_USDT",
    "name": "USDT",
    "displayName": "USDT (Ethereum)",
    "networkId": "ethereum-mainnet",
    "isTest": false
  },
  "paymentLink": {
    "id": "8c857501-e67d-4a0b-98d9-46e461b42c09"
  },
  "transaction": {
    "type": "incoming",
    "status": "completed",
    "hash": "0xdf147859a6e66961326ac91f4bd5e9980432040031e5eb7108603d51b81ae005",
    "explorerLink": "https://etherscan.io/tx/0xdf147859a6e66961326ac91f4bd5e9980432040031e5eb7108603d51b81ae005",
    "networkId": "ethereum-mainnet",
    "senderAddress": "0xsender",
    "recipientAddress": "0xrecipient",
    "amount": {
      "value": "29.9",
      "raw": "29900000",
      "currency": "ETH_USDT",
      "decimals": 6
    },
    "factAmount": {
      "value": "29.9",
      "raw": "29900000",
      "currency": "ETH_USDT",
      "decimals": 6
    },
    "serviceFee": {
      "value": "0.44",
      "raw": "440000",
      "currency": "ETH_USDT",
      "decimals": 6
    },
    "networkFee": {
      "value": "0.001",
      "raw": "1000000000000000",
      "currency": "ETH",
      "decimals": 18
    },
    "createdAt": "2022-12-22T19:05:55.386201Z",
    "updatedAt": "2022-12-22T19:12:55.386201Z",
    "isTest": false
  }
}
```

Withdrawal webhooks use the same envelope with `eventType: "withdrawal.status_changed"`, `type: "withdrawal"`, and `transaction.type: "withdrawal"`. The `payment.amount` value is the requested withdrawal amount, and `transaction.recipientAddress` is the merchant payout address.

## Delivery behavior

The receiver should return any `2xx` status to acknowledge delivery. Oxygen treats non-`2xx` responses and network failures as failed delivery attempts and logs them for operators.
