import { createHmac, timingSafeEqual } from "node:crypto";

export const DEFAULT_BASE_URL = "https://api.o2pay.co/api/merchant/v1";

export class OxygenApiError extends Error {
  constructor(statusCode, message, body = null) {
    super(message);
    this.name = "OxygenApiError";
    this.statusCode = statusCode;
    this.body = body;
  }
}

export class OxygenClient {
  constructor({ apiToken, merchantId, baseUrl = DEFAULT_BASE_URL, timeoutMs = 10000, fetchFn = globalThis.fetch }) {
    if (!apiToken) {
      throw new Error("apiToken is required");
    }
    if (!merchantId) {
      throw new Error("merchantId is required");
    }
    if (typeof fetchFn !== "function") {
      throw new Error("fetch is not available; use Node.js 18+ or pass fetchFn");
    }

    this.apiToken = apiToken;
    this.merchantId = merchantId;
    this.baseUrl = baseUrl.replace(/\/+$/, "");
    this.timeoutMs = timeoutMs;
    this.fetchFn = fetchFn;
  }

  createPayment(payment) {
    return this.request("POST", "/payment", { body: payment });
  }

  listPayments({ limit, cursor, reverseOrder, type } = {}) {
    return this.request("GET", "/payment", {
      query: { limit, cursor, reverseOrder, type },
    });
  }

  getPayment(paymentId) {
    return this.request("GET", `/payment/${encodeURIComponent(paymentId)}`);
  }

  createPaymentLink(paymentLink) {
    return this.request("POST", "/payment-link", { body: paymentLink });
  }

  listPaymentLinks() {
    return this.request("GET", "/payment-link");
  }

  getPaymentLink(paymentLinkId) {
    return this.request("GET", `/payment-link/${encodeURIComponent(paymentLinkId)}`);
  }

  deletePaymentLink(paymentLinkId) {
    return this.request("DELETE", `/payment-link/${encodeURIComponent(paymentLinkId)}`);
  }

  listBalances() {
    return this.request("GET", "/balance");
  }

  listCustomers({ limit, cursor, reverseOrder } = {}) {
    return this.request("GET", "/customer", {
      query: { limit, cursor, reverseOrder },
    });
  }

  getCustomer(customerId) {
    return this.request("GET", `/customer/${encodeURIComponent(customerId)}`);
  }

  async request(method, path, { body, query } = {}) {
    const url = this.buildUrl(path, query);
    const headers = {
      Accept: "application/json",
      "X-O2PAY-TOKEN": this.apiToken,
    };
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeoutMs);

    let payload;
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      payload = JSON.stringify(compact(body));
    }

    try {
      const response = await this.fetchFn(url, {
        method,
        headers,
        body: payload,
        signal: controller.signal,
      });

      if (response.status === 204) {
        return null;
      }

      const responseBody = await parseResponse(response);
      if (!response.ok) {
        throw new OxygenApiError(response.status, errorMessage(responseBody, response.statusText), responseBody);
      }

      return responseBody;
    } finally {
      clearTimeout(timeoutId);
    }
  }

  buildUrl(path, query) {
    const merchantId = encodeURIComponent(this.merchantId);
    const url = new URL(`${this.baseUrl}/merchant/${merchantId}${path}`);

    for (const [key, value] of Object.entries(compact(query || {}))) {
      url.searchParams.set(key, String(value));
    }

    return url;
  }
}

export function verifyWebhookSignature(payload, secret, signature) {
  if (!signature) {
    return false;
  }

  const body = toBuffer(payload);
  const expected = createHmac("sha512", secret).update(body).digest("base64");
  const expectedBuffer = Buffer.from(expected);
  const signatureBuffer = Buffer.from(signature);

  return expectedBuffer.length === signatureBuffer.length && timingSafeEqual(expectedBuffer, signatureBuffer);
}

export function parseWebhook(payload, { secret, signature } = {}) {
  if (secret && !verifyWebhookSignature(payload, secret, signature)) {
    throw new Error("Invalid OxygenPay webhook signature");
  }

  return JSON.parse(toBuffer(payload).toString("utf8"));
}

function compact(value) {
  return Object.fromEntries(Object.entries(value).filter(([, item]) => item !== undefined && item !== null));
}

async function parseResponse(response) {
  const text = await response.text();
  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function errorMessage(body, fallback) {
  return body && typeof body === "object" && typeof body.message === "string" ? body.message : fallback;
}

function toBuffer(payload) {
  if (Buffer.isBuffer(payload)) {
    return payload;
  }
  if (payload instanceof Uint8Array) {
    return Buffer.from(payload);
  }
  return Buffer.from(String(payload), "utf8");
}

