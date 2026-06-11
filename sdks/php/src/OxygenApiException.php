<?php

declare(strict_types=1);

namespace OxygenPay;

use RuntimeException;

final class OxygenApiException extends RuntimeException
{
    public function __construct(
        public readonly int $statusCode,
        string $message,
        public readonly mixed $body = null,
    ) {
        parent::__construct($message, $statusCode);
    }
}

