# OxygenPay PHP SDK

Server-side PHP client for the OxygenPay Merchant API.

Requires PHP 8.1+ with `ext-curl` and `ext-json`.

## Usage

```php
<?php

use OxygenPay\OxygenClient;

$client = new OxygenClient(
    apiToken: 'op_live_...',
    merchantId: 'merchant-uuid',
    baseUrl: 'http://localhost:8080/api/merchant/v1',
);

$payment = $client->createPayment([
    'id' => '123e4567-e89b-12d3-a456-426655440000',
    'currency' => 'USD',
    'price' => 29.90,
    'description' => 'White T-shirt size M',
    'redirectUrl' => 'https://example.com/success',
]);

echo $payment['paymentUrl'];
```

## Webhook Verification

Use the raw request body exactly as received.

```php
<?php

use OxygenPay\Webhooks;

$rawBody = file_get_contents('php://input');
$signature = $_SERVER['HTTP_X_SIGNATURE'] ?? null;

if (!Webhooks::verifySignature($rawBody, 'webhook-secret', $signature)) {
    http_response_code(400);
    exit;
}

$event = Webhooks::parse($rawBody);
```

