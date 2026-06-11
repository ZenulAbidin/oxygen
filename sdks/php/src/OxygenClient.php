<?php

declare(strict_types=1);

namespace OxygenPay;

final class OxygenClient
{
    public const DEFAULT_BASE_URL = 'https://api.o2pay.co/api/merchant/v1';

    public function __construct(
        private readonly string $apiToken,
        private readonly string $merchantId,
        private readonly string $baseUrl = self::DEFAULT_BASE_URL,
        private readonly int $timeoutSeconds = 10,
    ) {
        if ($this->apiToken === '') {
            throw new \InvalidArgumentException('apiToken is required');
        }
        if ($this->merchantId === '') {
            throw new \InvalidArgumentException('merchantId is required');
        }
    }

    public function createPayment(array $payment): array
    {
        return $this->request('POST', '/payment', body: $payment);
    }

    public function listPayments(
        ?int $limit = null,
        ?string $cursor = null,
        ?bool $reverseOrder = null,
        ?string $type = null,
    ): array {
        return $this->request('GET', '/payment', query: [
            'limit' => $limit,
            'cursor' => $cursor,
            'reverseOrder' => $reverseOrder,
            'type' => $type,
        ]);
    }

    public function getPayment(string $paymentId): array
    {
        return $this->request('GET', '/payment/' . rawurlencode($paymentId));
    }

    public function createPaymentLink(array $paymentLink): array
    {
        return $this->request('POST', '/payment-link', body: $paymentLink);
    }

    public function listPaymentLinks(): array
    {
        return $this->request('GET', '/payment-link');
    }

    public function getPaymentLink(string $paymentLinkId): array
    {
        return $this->request('GET', '/payment-link/' . rawurlencode($paymentLinkId));
    }

    public function deletePaymentLink(string $paymentLinkId): void
    {
        $this->request('DELETE', '/payment-link/' . rawurlencode($paymentLinkId));
    }

    public function listBalances(): array
    {
        return $this->request('GET', '/balance');
    }

    public function listCustomers(?int $limit = null, ?string $cursor = null, ?bool $reverseOrder = null): array
    {
        return $this->request('GET', '/customer', query: [
            'limit' => $limit,
            'cursor' => $cursor,
            'reverseOrder' => $reverseOrder,
        ]);
    }

    public function getCustomer(string $customerId): array
    {
        return $this->request('GET', '/customer/' . rawurlencode($customerId));
    }

    private function request(string $method, string $path, ?array $query = null, ?array $body = null): mixed
    {
        $url = $this->buildUrl($path, $query);
        $headers = [
            'Accept: application/json',
            'X-O2PAY-TOKEN: ' . $this->apiToken,
        ];

        $ch = curl_init($url);
        if ($ch === false) {
            throw new OxygenApiException(0, 'Unable to initialize HTTP client');
        }

        curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $method);
        curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
        curl_setopt($ch, CURLOPT_TIMEOUT, $this->timeoutSeconds);

        if ($body !== null) {
            $headers[] = 'Content-Type: application/json';
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode(self::compact($body), JSON_THROW_ON_ERROR));
        }

        curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);

        $response = curl_exec($ch);
        $statusCode = (int) curl_getinfo($ch, CURLINFO_RESPONSE_CODE);

        if ($response === false) {
            $message = curl_error($ch);
            curl_close($ch);
            throw new OxygenApiException(0, $message);
        }

        curl_close($ch);

        if ($statusCode === 204) {
            return null;
        }

        $decoded = $response === '' ? null : json_decode($response, true);
        $bodyValue = json_last_error() === JSON_ERROR_NONE ? $decoded : $response;

        if ($statusCode < 200 || $statusCode >= 300) {
            throw new OxygenApiException($statusCode, self::errorMessage($bodyValue, 'OxygenPay API request failed'), $bodyValue);
        }

        return $bodyValue;
    }

    private function buildUrl(string $path, ?array $query = null): string
    {
        $url = rtrim($this->baseUrl, '/') . '/merchant/' . rawurlencode($this->merchantId) . $path;
        $query = self::compact($query ?? []);

        if ($query !== []) {
            $url .= '?' . http_build_query($query);
        }

        return $url;
    }

    private static function compact(array $value): array
    {
        return array_filter($value, static fn (mixed $item): bool => $item !== null);
    }

    private static function errorMessage(mixed $body, string $fallback): string
    {
        if (is_array($body) && isset($body['message']) && is_string($body['message'])) {
            return $body['message'];
        }

        return $fallback;
    }
}

