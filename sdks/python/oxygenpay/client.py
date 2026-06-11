from __future__ import annotations

import base64
import hashlib
import hmac
import json
from typing import Any, Mapping
from urllib import error, parse, request

DEFAULT_BASE_URL = "https://api.o2pay.co/api/merchant/v1"


class OxygenAPIError(Exception):
    def __init__(self, status_code: int, message: str, body: Any | None = None):
        super().__init__(message)
        self.status_code = status_code
        self.body = body


class OxygenClient:
    def __init__(
        self,
        api_token: str,
        merchant_id: str,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = 10.0,
    ) -> None:
        if not api_token:
            raise ValueError("api_token is required")
        if not merchant_id:
            raise ValueError("merchant_id is required")

        self.api_token = api_token
        self.merchant_id = merchant_id
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def create_payment(self, payment: Mapping[str, Any] | None = None, **fields: Any) -> dict[str, Any]:
        return self._request("POST", "/payment", body=self._payload(payment, fields))

    def list_payments(
        self,
        limit: int | None = None,
        cursor: str | None = None,
        reverse_order: bool | None = None,
        payment_type: str | None = None,
    ) -> dict[str, Any]:
        return self._request(
            "GET",
            "/payment",
            query={
                "limit": limit,
                "cursor": cursor,
                "reverseOrder": reverse_order,
                "type": payment_type,
            },
        )

    def get_payment(self, payment_id: str) -> dict[str, Any]:
        return self._request("GET", f"/payment/{parse.quote(payment_id, safe='')}")

    def create_payment_link(self, payment_link: Mapping[str, Any] | None = None, **fields: Any) -> dict[str, Any]:
        return self._request("POST", "/payment-link", body=self._payload(payment_link, fields))

    def list_payment_links(self) -> dict[str, Any]:
        return self._request("GET", "/payment-link")

    def get_payment_link(self, payment_link_id: str) -> dict[str, Any]:
        return self._request("GET", f"/payment-link/{parse.quote(payment_link_id, safe='')}")

    def delete_payment_link(self, payment_link_id: str) -> None:
        self._request("DELETE", f"/payment-link/{parse.quote(payment_link_id, safe='')}")

    def list_balances(self) -> dict[str, Any]:
        return self._request("GET", "/balance")

    def list_customers(
        self,
        limit: int | None = None,
        cursor: str | None = None,
        reverse_order: bool | None = None,
    ) -> dict[str, Any]:
        return self._request(
            "GET",
            "/customer",
            query={"limit": limit, "cursor": cursor, "reverseOrder": reverse_order},
        )

    def get_customer(self, customer_id: str) -> dict[str, Any]:
        return self._request("GET", f"/customer/{parse.quote(customer_id, safe='')}")

    def _request(
        self,
        method: str,
        path: str,
        body: Mapping[str, Any] | None = None,
        query: Mapping[str, Any] | None = None,
    ) -> Any:
        url = self._url(path, query)
        payload = None
        headers = {
            "Accept": "application/json",
            "X-O2PAY-TOKEN": self.api_token,
        }

        if body is not None:
            payload = json.dumps(_compact(body), separators=(",", ":")).encode("utf-8")
            headers["Content-Type"] = "application/json"

        req = request.Request(url=url, data=payload, method=method, headers=headers)

        try:
            with request.urlopen(req, timeout=self.timeout) as response:
                if response.status == 204:
                    return None
                return _decode_json(response.read())
        except error.HTTPError as exc:
            body_data = exc.read()
            decoded = _decode_json(body_data)
            message = _error_message(decoded, exc.reason)
            raise OxygenAPIError(exc.code, message, decoded) from exc
        except error.URLError as exc:
            raise OxygenAPIError(0, str(exc.reason), None) from exc

    def _url(self, path: str, query: Mapping[str, Any] | None = None) -> str:
        merchant_path = f"/merchant/{parse.quote(self.merchant_id, safe='')}{path}"
        url = f"{self.base_url}{merchant_path}"
        query_string = _query_string(query)
        if query_string:
            url = f"{url}?{query_string}"
        return url

    @staticmethod
    def _payload(payload: Mapping[str, Any] | None, fields: Mapping[str, Any]) -> dict[str, Any]:
        data = dict(payload or {})
        data.update(fields)
        return _compact(data)


def verify_webhook_signature(payload: bytes | str, secret: str, signature: str | None) -> bool:
    if not signature:
        return False
    body = payload.encode("utf-8") if isinstance(payload, str) else payload
    digest = hmac.new(secret.encode("utf-8"), body, hashlib.sha512).digest()
    expected = base64.b64encode(digest).decode("ascii")
    return hmac.compare_digest(expected, signature)


def parse_webhook(payload: bytes | str, secret: str | None = None, signature: str | None = None) -> dict[str, Any]:
    if secret is not None and not verify_webhook_signature(payload, secret, signature):
        raise ValueError("Invalid OxygenPay webhook signature")
    body = payload.decode("utf-8") if isinstance(payload, bytes) else payload
    return json.loads(body)


def _compact(data: Mapping[str, Any]) -> dict[str, Any]:
    return {key: value for key, value in data.items() if value is not None}


def _query_string(query: Mapping[str, Any] | None) -> str:
    if not query:
        return ""
    params = {
        key: str(value).lower() if isinstance(value, bool) else value
        for key, value in _compact(query).items()
    }
    return parse.urlencode(params)


def _decode_json(raw: bytes) -> Any:
    if not raw:
        return None
    try:
        return json.loads(raw.decode("utf-8"))
    except json.JSONDecodeError:
        return raw.decode("utf-8", errors="replace")


def _error_message(decoded: Any, fallback: str) -> str:
    if isinstance(decoded, dict):
        message = decoded.get("message")
        if isinstance(message, str) and message:
            return message
    return fallback
