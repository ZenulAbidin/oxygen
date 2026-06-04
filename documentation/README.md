# OxygenPay Documentation

This folder contains the local documentation site served by the `documentation`
service in `docker-compose.local.yml`.

The API spec source of truth stays in `api/proto`. The documentation Dockerfile
copies those specs into the docs image under `/specs`, and copies the checked-in
ReDoc runtime from `web/redoc/redoc.standalone.js`.

Local URLs:

- Documentation home: `http://localhost:8081`
- Merchant API: `http://localhost:8081/api/merchant.html`
- Dashboard API: `http://localhost:8081/api/dashboard.html`
- Payment API: `http://localhost:8081/api/payment.html`
- Internal Admin API: `http://localhost:8081/api/admin.html`
- KMS API: `http://localhost:8081/api/kms.html`
