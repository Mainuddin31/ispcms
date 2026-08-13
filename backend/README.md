# IBMS Backend

Go REST API powering the ISP Business Management System.

**Stack:** Go 1.22 · Fiber v2 · GORM v2 · PostgreSQL · go-routeros · gosnmp

---

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                  Entry point — connects DB, runs migrate+seed, starts Fiber + OLT scheduler
├── internal/
│   ├── config/
│   │   └── config.go                Reads env vars into Config struct
│   ├── database/
│   │   ├── database.go              Opens GORM PostgreSQL connection
│   │   └── migrate.go               AutoMigrate all models + seed roles/permissions/super-admin
│   ├── handlers/                    HTTP handlers (one file per module)
│   │   ├── auth_handler.go
│   │   ├── user_handler.go
│   │   ├── role_handler.go
│   │   ├── router_handler.go
│   │   ├── pppoe_handler.go
│   │   ├── internet_account_handler.go
│   │   ├── olt_handler.go           OLT + SNMP profile + ONU + SNMP probe endpoints
│   │   ├── dashboard_handler.go
│   │   ├── package_handler.go
│   │   ├── profile_mapping_handler.go
│   │   ├── bill_handler.go
│   │   ├── report_handler.go
│   │   ├── visit_handler.go
│   │   └── notification_handler.go
│   ├── middleware/
│   │   ├── auth.go                  JWT validation, sets userID/roles in Locals
│   │   └── rbac.go                  RequirePermission, RequireRole
│   ├── models/                      GORM models (BeforeCreate sets uuid.New())
│   │   ├── user.go
│   │   ├── role.go
│   │   ├── router.go
│   │   ├── pppoe.go
│   │   ├── internet_account.go
│   │   ├── olt.go                   SNMPProfile, OLT, PONPort, OLTSyncLog
│   │   ├── log.go                   SyncLog, ActivityLog
│   │   ├── package.go
│   │   ├── profile_mapping.go
│   │   ├── subscription.go
│   │   ├── bill.go
│   │   ├── notification.go
│   │   ├── visit.go
│   │   └── expense.go
│   ├── repositories/                DB query layer — one interface + struct per model
│   │   └── onu_repo.go              Includes AutoLinkByMAC + LinkFromMACTable
│   ├── router/
│   │   └── router.go                Fiber app setup, all repos/services/handlers wired here
│   └── services/
│       ├── auth_service.go          Login, JWT issue/refresh
│       ├── user_service.go
│       ├── role_service.go
│       ├── router_service.go
│       ├── sync_service.go          Syncs PPPoE secrets + sessions from MikroTik; assigns subscriptions
│       ├── olt_service.go           OLT CRUD + OLTScheduler (background auto-sync)
│       ├── olt_sync_service.go      SNMP ONU sync + FDB MAC walk + CLI MAC scrape + auto-link
│       ├── dashboard_service.go
│       ├── billing_service.go
│       ├── activity_service.go
│       ├── visiting_service.go
│       ├── expense_service.go
│       └── notification_service.go
└── pkg/
    ├── mikrotik/
    │   ├── client.go                go-routeros TCP connection wrapper
    │   └── pppoe.go                 GetPPPoESecrets, GetActiveSessions
    ├── snmp/
    │   └── client.go                SNMP v2c/v3 walk client (GETBULK → GETNEXT fallback)
    ├── telnet/
    │   ├── client.go                Telnet client with IAC negotiation + enable-mode login
    │   └── mac_table.go             Richerlink "show mac-address-table" parser
    └── utils/
        ├── jwt.go                   Generate/parse JWT
        ├── response.go              Fiber response helpers
        └── encrypt.go              AES-256-GCM encrypt/decrypt (stored passwords)
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
go run ./cmd/server/main.go
```

API available at `http://localhost:8082/api/v1`

---

## API Overview

All routes are prefixed `/api/v1`. Protected routes require `Authorization: Bearer <token>`.

### Auth
| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/login` | Returns `{ tokens: { access_token, refresh_token } }` |
| POST | `/auth/refresh` | Exchange refresh token |
| POST | `/auth/logout` | Invalidate session |
| GET  | `/auth/profile` | Current user info |

### OLT Management
| Method | Path | Description |
|--------|------|-------------|
| GET | `/snmp-profiles` | List SNMP profiles |
| POST | `/snmp-profiles` | Create SNMP profile |
| PUT | `/snmp-profiles/:id` | Update SNMP profile (including OID map) |
| DELETE | `/snmp-profiles/:id` | Delete profile |
| GET | `/olts` | List all OLTs |
| POST | `/olts` | Create OLT |
| GET | `/olts/:id` | Get OLT detail |
| PUT | `/olts/:id` | Update OLT (including CLI credentials) |
| DELETE | `/olts/:id` | Soft-delete OLT |
| POST | `/olts/:id/sync` | Trigger manual SNMP sync |
| POST | `/olts/:id/test-connection` | Test SNMP reachability |
| POST | `/olts/:id/snmp-probe` | Walk any OID for diagnostics `{ oid, limit }` |
| GET | `/olts/:id/sync-logs` | Sync history for one OLT |
| GET | `/olts/:id/pon-ports` | PON port list |
| POST | `/olts/:id/auto-link-onus` | Manually trigger MAC-based ONU linking |
| GET | `/olts/stats` | Aggregate OLT counts |
| GET | `/olts/sync-logs/recent` | Recent sync logs across all OLTs |
| GET | `/onus` | List ONUs (filter: olt_id, pon_port_id, status, unlinked, search) |
| GET | `/onus/:id` | Get ONU detail |
| PATCH | `/onus/:id/link` | Manually link/unlink ONU to internet account |
| POST | `/onus/auto-link` | Run MAC auto-link across all OLTs |

### Billing
| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/packages` | List / create packages |
| PUT/DELETE | `/packages/:id` | Update / delete |
| GET/POST | `/profile-mappings` | List / create profile→package mappings |
| GET | `/subscriptions` | List subscriptions |
| POST | `/subscriptions` | Manually assign package to account |
| GET | `/bills` | List bills (filter by month, year, status) |
| POST | `/bills/generate` | Generate bills for a month |
| PATCH | `/bills/:id/status` | Record payment / change status |
| GET | `/notifications` | List notifications |

---

## OLT Sync Engine

### Overview

The sync engine runs in two modes:
- **Manual** — `POST /olts/:id/sync`
- **Automatic** — background goroutine checks every minute; syncs any OLT whose `sync_interval > 0` and whose last sync is overdue

Each sync run:
1. Discovers PON ports and ONUs via SNMP walk
2. Upserts ONUs into the database (new / updated / archived)
3. Auto-links unlinked ONUs to internet accounts via two strategies (see below)
4. Logs the result (`linked_onus`, `onus_found`, `ports_discovered`, etc.)

### SNMP Profile OID Map

Each OLT references an SNMP profile that maps logical names to vendor OIDs:

| Key | Description |
|-----|-------------|
| `onu_mac` | Base OID for ONU MAC walk |
| `onu_status` | Base OID for ONU operational status |
| `onu_rx_power` | RX optical power |
| `onu_tx_power` | TX optical power |
| `onu_distance` | Distance from OLT |
| `onu_serial` | Serial number |
| `onu_model` | ONU model/type |
| `index_port_pos` | Position in OID suffix holding port number (default `"0"`) |
| `index_onu_pos` | Position in OID suffix holding ONU slot (default `"1"`) |
| `power_divisor` | Raw integer divisor to convert to dBm (e.g. `"-10"`) |
| `mac_table_oid` | FDB port table — `dot1qTpFdbPort` or `dot1dTpFdbPort` |
| `mac_table_port_ifindex_oid` | Bridge port → ifIndex — `dot1dBasePortIfIndex` |
| `mac_table_ifdescr_oid` | ifIndex → interface name — `ifDescr` |
| `use_getnext` | `"true"` forces GETNEXT walk (for devices that don't support GETBULK) |
| `index_packed` | `"true"` for Richerlink packed index format (`port<<16\|slot<<8\|onu`) |

### ONU Auto-Linking

After each sync, unlinked ONUs are matched to internet accounts using two strategies applied in order:

**Strategy 1 — ONU Hardware MAC match (`AutoLinkByMAC`)**
Matches `onus.mac_address` against `internet_accounts.caller_id` (both normalized to bare lowercase hex). Works when the ONU's hardware MAC is set as the PPPoE caller-id.

**Strategy 2 — Bridge MAC table (FDB or CLI)**
Identifies which customer CPE devices are reachable through each ONU, then matches those MACs against `internet_accounts.caller_id`.

The engine picks the source automatically per OLT:

| OLT type | Source | How |
|----------|--------|-----|
| VSOL EPON | SNMP FDB walk | `dot1qTpFdbPort` → `ifDescr` (`EPON0/P:S` format) |
| Richerlink EPON | Telnet CLI scrape | `show mac-address-table` → parse `epon<P> onu<S>` |

**CLI scrape is preferred** — if `cli_protocol` is set on the OLT, the engine always uses Telnet/SSH and skips the SNMP FDB walk.

### OLT CLI Credentials (Telnet / SSH)

Add these fields when creating or updating an OLT to enable CLI-based MAC scraping:

| Field | Description |
|-------|-------------|
| `cli_protocol` | `"telnet"` or `"ssh"` (empty = disabled) |
| `cli_port` | Port number (0 = use default: 23 for Telnet, 22 for SSH) |
| `cli_username` | Login username |
| `cli_password` | Login password (stored AES-encrypted) |
| `cli_enable_password` | Privileged-mode password (empty = same as `cli_password`) |

Example — enable Telnet scraping on a Richerlink OLT:
```bash
curl -X PUT /api/v1/olts/<id> \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "cli_protocol": "telnet",
    "cli_port": 23,
    "cli_username": "admin",
    "cli_password": "yourpassword",
    "cli_enable_password": "enablepassword",
    ... (other OLT fields)
  }'
```

The Telnet client handles:
- IAC option negotiation (all options refused for plain byte stream)
- Two-stage login: user-mode (`EPON>`) → `enable` → privileged-mode (`EPON#`)
- Automatic fallback: if no enable password required, skips that step

### SNMP Probe API

Useful for testing OIDs on a real OLT before adding them to a profile:

```bash
curl -X POST /api/v1/olts/<id>/snmp-probe \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"oid": "1.3.6.1.2.1.17.7.1.2.2.1.2", "limit": 20}'
```

Returns `{ count, entries: [{ oid_suffix, value }] }`.

### MikroTik caller-id Backfill

For ONUs to auto-link, each internet account's `caller_id` must contain the customer CPE MAC. If `caller_id` is empty in MikroTik, run this RouterOS script to backfill from active sessions:

```routeros
/ppp secret
:foreach s in=[find where caller-id=""] do={
    :local username [get $s name]
    :local session [/ppp active find where name=$username]
    :if ([:len $session] > 0) do={
        :local mac [/ppp active get ($session->0) caller-id]
        :if ($mac != "") do={
            set $s caller-id=$mac
            :log info ("Set caller-id for " . $username . " = " . $mac)
        }
    }
}
```

Schedule it to run nightly:
```routeros
/system scheduler add name=fill-caller-id interval=1d \
  on-event="/system script run fill-caller-id"
```

---

## Billing Logic

### Subscription assignment (automatic)
1. Router sync triggers → for each PPPoE account, looks up `profile` in `profile_mappings`
2. If mapping found → calls `AssignSubscriptionFromSync(accountID, package)`
3. Creates a new `CustomerSubscription` (closes the old one if the package changed)

### Bill generation
1. `POST /bills/generate` with `{ month, year }`
2. For each non-archived, non-disabled account:
   - Finds the subscription active on the last day of the billing month
   - Skips if already billed or no subscription found
   - Creates `MonthlyBill` with charge = subscription price + VAT
