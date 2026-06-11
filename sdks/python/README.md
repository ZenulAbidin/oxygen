# OxygenPay Python SDK

Server-side Python client for the OxygenPay Merchant API.

## Usage

```python
from uuid import uuid4
from oxygenpay import OxygenClient

client = OxygenClient(
    api_token="op_live_...",
    merchant_id="merchant-uuid",
    base_url="http://localhost:8080/api/merchant/v1",
)

payment = client.create_payment({
    "id": str(uuid4()),
    "currency": "USD",
    "price": 29.90,
    "description": "White T-shirt size M",
    "redirectUrl": "https://example.com/success",
})

print(payment["paymentUrl"])
```

## Webhook Verification

Use the raw request body exactly as received.

```python
from oxygenpay import verify_webhook_signature

is_valid = verify_webhook_signature(
    payload=raw_body,
    secret="webhook-secret",
    signature=request.headers.get("X-Signature"),
)
```

