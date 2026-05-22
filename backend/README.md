# Backend

Go + Gin + GORM backend scaffold for the sing-box server management SaaS.

## Run

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

The server automatically loads `.env` from the current directory. Shell
environment variables take precedence over values in `.env`.

The health check is available at:

```text
GET /healthz
```

Most API routes are scaffolded and currently return `501 Not Implemented`.
