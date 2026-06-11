<?php

declare(strict_types=1);

namespace OxygenPay;

final class Webhooks
{
    public static function verifySignature(string $payload, string $secret, ?string $signature): bool
    {
        if ($signature === null || $signature === '') {
            return false;
        }

        $expected = base64_encode(hash_hmac('sha512', $payload, $secret, true));

        return hash_equals($expected, $signature);
    }

    public static function parse(string $payload, ?string $secret = null, ?string $signature = null): array
    {
        if ($secret !== null && !self::verifySignature($payload, $secret, $signature)) {
            throw new \InvalidArgumentException('Invalid OxygenPay webhook signature');
        }

        return json_decode($payload, true, flags: JSON_THROW_ON_ERROR);
    }
}

