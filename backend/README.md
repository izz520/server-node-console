# Backend

Go + Gin + GORM backend for the sing-box server management SaaS.

## Run

```bash
cp .env.example .env
go run ./cmd/api
```

The server automatically loads `.env` from the current directory. If it is started from the repository root, it also attempts to load `backend/.env`. Shell environment variables take precedence over values in `.env`.

## Database

The PostgreSQL database must already exist before the API starts. The API runs GORM auto migration to create or update tables, but it does not create the database itself.

Docker Compose default:

```bash
docker compose up -d postgres
```

Local PostgreSQL example:

```bash
createdb -h 127.0.0.1 -p 5432 -U <your-db-user> singbox_manager
```

Then update `DATABASE_DSN` in `.env`.

## Endpoints

Health check:

```text
GET /healthz
```

Main API prefix:

```text
/api/v1
```

Public subscription links:

```text
GET /sub/{token}
```

## Verify

```bash
go test ./...
```
