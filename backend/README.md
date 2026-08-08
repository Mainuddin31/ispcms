# IBMS Backend

Go REST API powering the ISP Business Management System.

**Stack:** Go 1.22 · Fiber v2 · GORM v2 · PostgreSQL · go-routeros

---

## Project Structure

```
backend/
├── cmd/
│   └── main.go                  Entry point — connects DB, runs migrate+seed, starts Fiber
├── internal/
│   ├── config/
│   │   └── config.go            Reads env vars into Config struct
│   ├── database/
│   │   ├── database.go          Opens GORM PostgreSQL connection
│   │   └── migrate.go           AutoMigrate all models + seed roles/permissions/super-admin
│   ├── handlers/                HTTP handlers (one file per module)
│   │   ├── auth_handler.go
│   │   ├── user_handler.go
│   │   ├── role_handler.go
│   │   ├── router_handler.go
│   │   ├── pppoe_handler.go
│   │   ├── internet_account_handler.go
│   │   ├── dashboard_handler.go
│   │   ├── package_handler.go
│   │   ├── profile_mapping_handler.go
│   │   ├── bill_handler.go
│   │   └── notification_handler.go
│   ├── middleware/
│   │   ├── auth.go              JWT validation, sets userID/roles in Locals
│   │   └── permission.go        RequirePermission, RequireRole
│   ├── models/                  GORM models (BeforeCreate sets uuid.New())
│   │   ├── user.go
│   │   ├── router.go
│   │   ├── pppoe.go
│   │   ├── internet_account.go
│   │   ├── sync_log.go
│   │   ├── activity_log.go
│   │   ├── package.go
│   │   ├── profile_mapping.go
│   │   ├── subscription.go
│   │   ├── bill.go              MonthlyBill + BillGenerationLog
│   │   └── notification.go
│   ├── repositories/            DB query layer — one interface + struct per model
│   ├── router/
│   │   └── router.go            Fiber app setup, all repos/services/handlers wired here
│   └── services/
│       ├── auth_service.go      Login, JWT issue/refresh
│       ├── user_service.go
│       ├── role_service.go
│       ├── router_service.go
│       ├── sync_service.go      Syncs PPPoE secrets + sessions from MikroTik; assigns subscriptions
│       ├── dashboard_service.go Aggregates stats for dashboard
│       ├── billing_service.go   PackageService + ProfileMappingService + BillingService
│       └── notification_service.go
└── pkg/
    ├── mikrotik/
    │   ├── client.go            go-routeros TCP connection wrapper
    │   └── pppoe.go             GetPPPoESecrets, GetActiveSessions
    └── utils/
        ├── jwt.go               Generate/parse JWT
        └── crypto.go            AES-256-GCM encrypt/decrypt (router passwords)
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASSWORD` | *(required)* | PostgreSQL password |
| `DB_NAME` | `ispcms` | Database name |
| `JWT_SECRET` | *(required)* | Secret for JWT signing + AES credential encryption |
| `JWT_EXPIRY` | `15m` | Access token TTL |
| `JWT_REFRESH_EXPIRY` | `7d` | Refresh token TTL |
| `SERVER_PORT` | `8080` | Fiber listen port |
| `CORS_ORIGINS` | `http://localhost:3000` | Allowed CORS origins |
| `SUPER_ADMIN_USERNAME` | `admin` | Created on first run |
| `SUPER_ADMIN_PASSWORD` | `admin123` | Created on first run |
| `SUPER_ADMIN_EMAIL` | `admin@example.com` | Created on first run |

---

## Running Locally

```bash
# With Docker Compose (recommended — starts DB + backend + frontend)
docker compose up -d

# Standalone (needs a running PostgreSQL)
cd backend
go run ./cmd/main.go
```

API available at `http://localhost:8080/api/v1`

Health check: `GET http://localhost:8080/health`

---

## API Overview

All routes are prefixed `/api/v1`. Protected routes require `Authorization: Bearer <token>`.

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/login` | Returns access + refresh tokens |
| POST | `/auth/refresh` | Exchange refresh token |
| POST | `/auth/logout` | Invalidate session |
| GET  | `/auth/profile` | Current user info |

### Billing
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/packages` | List / create packages |
| PUT/DELETE | `/packages/:id` | Update / delete |
| GET | `/packages/active` | Active packages only (for dropdowns) |
| GET/POST | `/profile-mappings` | List / create profile→package mappings |
| GET | `/profile-mappings/unmapped` | Profiles in internet_accounts with no mapping |
| GET | `/subscriptions` | List subscriptions |
| POST | `/subscriptions` | Manually assign package to account |
| GET | `/subscriptions/account/:accountId` | Active subscription for one account |
| GET | `/bills` | List bills (filter by month, year, status) |
| POST | `/bills/generate` | Generate bills for a month |
| PATCH | `/bills/:id/status` | Record payment / change status |
| GET | `/bills/status` | Summary: generated vs pending for a month |
| GET | `/bills/generation-logs` | Past generation runs |
| GET | `/notifications` | List notifications |
| GET | `/notifications/unread-count` | Badge count |
| PATCH | `/notifications/:id/read` | Mark one read |
| POST | `/notifications/mark-all-read` | Mark all read |

---

## Billing Logic

### Subscription assignment (automatic)
1. Sync triggers → for each PPPoE account, looks up `profile` in `profile_mappings`
2. If mapping found → calls `AssignSubscriptionFromSync(accountID, package)`
3. Creates a new `CustomerSubscription` (closes the old one first if it exists and the package changed)

### Bill generation
1. `POST /bills/generate` with `{ month, year }`
2. For each non-archived, non-disabled account:
   - Finds the subscription active on the **last day of the billing month** (capped at today for the current month)
   - Skips if already billed or no subscription found
   - Creates `MonthlyBill` with charge = subscription price + VAT

### Package change rule
If a customer's package changed on the 20th, the current month's bill (already generated or generated later) uses the subscription that was active at the lookup date (end of month). Historical bills are never touched.
