# Backend

Go + Gin + GORM backend scaffold for the sing-box server management SaaS.

## Run

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

The health check is available at:

```text
GET /healthz
```

Most API routes are scaffolded and currently return `501 Not Implemented`.
