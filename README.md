# IBMS — ISP Business Management System

A full-stack management platform for ISPs running MikroTik routers and PON/OLT equipment. Manages PPPoE customers, billing packages, monthly invoicing, payment collection, expenses, and OLT/ONU network inventory.

**Stack:** Go 1.22 · Fiber v2 · GORM · PostgreSQL · Next.js 14 · TypeScript · Tailwind · shadcn/ui · Docker Compose

---

## Modules

### Infrastructure
| Page | Purpose |
|------|---------|
| **Routers** | Add MikroTik routers, test connection, trigger per-router or all-router sync |
| **Internet Accounts** | All PPPoE customers synced from routers — online/offline status, IP, uptime. Click username to open the Customer Profile with full billing history |
| **PPPoE Secrets** | Raw PPPoE secret list from RouterOS |
| **Active Sessions** | Live PPPoE sessions |

### Billing
| Page | Purpose |
|------|---------|
| **Packages** | Define billable internet packages (price, speed, VAT) |
| **Profile Mappings** | Link MikroTik PPPoE profiles → billing packages (drives auto-subscription) |
| **Subscriptions** | One active subscription per customer; auto-assigned on sync or manually set |
| **Bills** | Monthly invoices — generate, view, record payments with Payment Method + Receipt Number |
| **Notifications** | System alerts (unmapped profiles, bill generation results) |

### Expenses
| Page | Purpose |
|------|---------|
| **Expenses** | Record operational expenses with category, method, vendor, and reference number |
| **Expense Categories** | Manage expense categories (Office, Rent, Maintenance, etc.) |

### Network (OLT / PON)
| Page | Purpose |
|------|---------|
| **Network Dashboard** | Stats overview — OLT count, PON ports, ONU online/offline/unassigned, port utilization, recent sync activity |
| **OLTs** | Add/edit OLT devices (SNMP v2c or v3), trigger manual sync, view per-OLT sync logs |
| **ONU Inventory** | Full ONU list with filters (OLT, port, status, unlinked-only), optical power, distance, and link-to-account action |
| **SNMP Profiles** | Vendor-specific OID maps — define once, reuse across OLTs. Includes seeded profiles for BDCOM, VSOL, C-Data (EPON) and Huawei, ZTE (GPON) |

### Admin
| Page | Purpose |
|------|---------|
| **Users** | Staff accounts |
| **Roles & Permissions** | RBAC — super_admin, admin, billing_officer, operator, viewer |

---

## Billing Flow

```
1. Create Packages         → define price, speed, VAT
2. Create Profile Mappings → map MikroTik profile name → Package
3. Sync Routers            → auto-assigns subscriptions based on each account's profile
4. Generate Bills          → Bills page → "Generate Bills" → generates for current month
5. Record Payments         → Bills page → "Record Payment" (select method, enter receipt no.)
6. View Customer Profile   → Internet Accounts → click any username
```

### How bill generation works

Bills → **Generate Bills** → click Generate (current month only).

The system finds every non-archived, non-disabled account that had an active subscription on the **1st of the billing month**, creates one bill, and skips duplicates automatically. A generation log records how many were created vs skipped and why.

**Package change rules:**
- If a customer changed packages on July 18, the July bill still uses the old package (billing date = July 1).
- August bill uses the new package (new subscription started before August 1).
- Historical bills are never recalculated.

### Recording Payments

Bills → find the bill → **Record Payment**. The dialog accepts:
- **Amount received** (can be partial — the bill status moves to `partial`)
- **Payment Method** — Cash, bKash, Bank Transfer, Card, Other
- **Receipt Number** — optional reference for your records
- **Notes** — free text

Each payment saves a `PaymentRecord`. A bill can have multiple payment records (partial payments over time). When total paid ≥ total amount, status automatically becomes `paid`.

### Customer Profile (Billing History)

On the **Internet Accounts** page, click any username to open the Customer Profile. It shows:

- Customer info: username, router, current package, monthly charge
- Summary: total bills, total paid, total outstanding
- Full billing history table with: Month, Bill #, Amount, Paid, Due, Status, Payment Method, Receipt #, Collected By, Payment Date

### Dashboard — Financial Stats

The Dashboard shows real-time financial metrics:

| Metric | Source |
|--------|--------|
| **Today's Collection** | Sum of payments received today |
| **This Month** | Sum of payments received this calendar month |
| **Total Collection** | All-time payment sum |
| **Outstanding Due** | Sum of `due_amount` on all non-paid, non-cancelled bills |
| **Today's Expense** | Expenses recorded today |
| **Monthly Expense** | Expenses recorded this calendar month |
| **Cash in Hand** | Total collection minus total expense |

Charts: 12-month Collection vs Expense vs Cash in Hand (line), expense by category (pie), monthly collection (bar). Activity timeline shows recent actions across all modules.

---

## Expense Tracking

Expenses → **Add Expense**. Each expense records:
- Date, category, amount, payment method
- Vendor and reference number (optional)
- Description / notes

Expense categories are managed separately and can be activated/deactivated. The Dashboard expense pie chart shows spending by category for the current month.

Activity is logged for every expense create/update/delete (visible in the Dashboard activity timeline).

---

## OLT / Network Module

### SNMP Profiles

Before adding an OLT, create (or use the seeded) SNMP profile for its vendor. A profile stores:
- **Vendor** and **Technology** (EPON / GPON)
- **OID Map** — key/value pairs mapping standard attribute names (`onu_mac`, `onu_status`, `onu_rx_power`, `onu_tx_power`, `onu_distance`, `onu_serial`, `onu_model`) to the actual vendor OID strings
- **`index_port_pos`** / **`index_onu_pos`** — which segment of the SNMP walk index suffix encodes the port number and ONU slot (0-based)

Seeded profiles (created on first startup):

| Profile | Vendor | Technology | Notes |
|---------|--------|------------|-------|
| BDCOM_EPON | BDCOM | EPON | |
| VSOL_EPON | VSOL | EPON | Uses `1.3.6.1.4.1.37950.1.1.5.12.1.25.1.*` OID tree; power stored as absolute value |
| CDATA_EPON | C-Data | EPON | |
| HUAWEI_GPON | Huawei | GPON | |
| ZTE_GPON | ZTE | GPON | |
| RICHERLINK_EPON | Richerlink | EPON | Packed index encoding; string float power values |
| RICHERLINK_EPON_V2 | Richerlink | EPON | Firmware V1.0.0.32715+; uses GETNEXT walk (no GETBULK support); no per-ONU optical power |

**Special OID map keys:**

| Key | Type | Purpose |
|-----|------|---------|
| `index_port_pos` | int string | Position in OID suffix that holds port number (default `"0"`) |
| `index_onu_pos` | int string | Position in OID suffix that holds ONU slot (default `"1"`) |
| `index_packed` | `"true"` | ONU index is a single encoded int: `port<<16 \| slot<<8 \| onu` (Richerlink) |
| `status_online_string` | string | Match substring for string-based online status (e.g. `"configuration ok"`) |
| `power_divisor` | float string | Divide raw power integer by this value to get dBm (default `"10"`) |
| `rx_power_negate` | `"true"` | Negate RX power value — for OLTs that store it as an absolute positive |
| `distance_unit` | `"m"` / `"cm"` | Distance unit from the OLT; `"cm"` values are auto-converted to metres |
| `use_getnext` | `"true"` | Force GETNEXT walk instead of GETBULK — for firmware that doesn't respond to GETBULK |

### Adding an OLT

OLTs → **Add OLT**:
- Management IP, vendor, model
- Select SNMP profile
- SNMP version: **v2c** (community string) or **v3** (username + auth/priv protocol + passwords)
- Sync interval in minutes (`0` = manual only)

v3 passwords are encrypted at rest using AES and masked in API responses.

### OLT Sync

Sync walks the OIDs defined in the OLT's SNMP profile and:
1. Discovers PON ports (created lazily on first ONU discovery for that port)
2. Upserts each ONU by `(olt_id, port_index, onu_slot)` — the idempotent key
3. Archives ONUs no longer visible (sets `archived_at`)
4. Updates ONU counts on each PON port
5. Logs a `OLTSyncLog` with counts: ports discovered, ONUs discovered/new/updated/archived

**Background scheduler** checks every minute for OLTs whose `sync_interval > 0` and whose `last_sync_at` is older than `sync_interval` minutes, then runs sync automatically.

Sync can also be triggered manually per-OLT from the OLTs page.

### ONU Inventory

The ONU Inventory page lists all ONUs with filters for OLT, port, status (online/offline), and unlinked-only. Select an OLT to reveal the port filter — pick a specific PON port to see only ONUs on that port. Click any row to open a detail sheet showing:

- MAC address, serial number, vendor/model
- Optical power (Rx/Tx dBm, color-coded: green ≥ −20, yellow −20 to −25, red < −25) and distance
- Last online timestamp
- Linked internet account (with change/unlink action)

**Link ONU → Account**: from the detail sheet, click the link icon to open the Link dialog. Search for an internet account by username or IP, select it, and save. This associates the physical ONU (fiber modem) with the PPPoE customer record.

---

## PPPoE Sync

Sync is idempotent — running it multiple times is safe.

**What happens on each sync:**
1. Orphaned rows (nil router_id) are cleaned first
2. Each PPPoE secret is upserted by `(router_id, username)` — the unique key
3. Accounts in MikroTik but not in IBMS → created; accounts in IBMS but not in MikroTik → archived
4. Active sessions are fetched; `is_online`, `current_ip`, `uptime` updated per account
5. If an account's profile has a Package Mapping → subscription auto-assigned

**Sync Summary dialog** appears after "Sync All" showing: new, updated, archived account counts, online/offline users, and any router errors.

---

## Deploy to Ubuntu (Production)

This section covers a full production deployment on Ubuntu 20.04 or 22.04 using Docker Compose and Nginx.

### 1. Prepare the server

```bash
# Update packages
sudo apt update && sudo apt upgrade -y

# Install required tools
sudo apt install -y git curl ufw
```

### 2. Install Docker & Docker Compose

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Add your user to the docker group (so you don't need sudo every time)
sudo usermod -aG docker $USER

# Apply group change (or log out and back in)
newgrp docker

# Verify Docker is running
docker --version
docker compose version
```

### 3. Configure firewall (UFW)

```bash
# Allow SSH (important — do this before enabling UFW)
sudo ufw allow OpenSSH

# Allow HTTP and HTTPS (for Nginx)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable

# Verify
sudo ufw status
```

> **Note:** Do NOT expose ports 8081 (backend) or 5869 (frontend) directly. Nginx will handle all incoming traffic.

### 4. Clone the project

```bash
# Choose your deploy directory
cd /opt
sudo git clone https://github.com/your-org/ispcms.git
sudo chown -R $USER:$USER ispcms
cd ispcms
```

Or transfer from your machine:

```bash
# From your local machine
scp -r ./ispcms user@your-server-ip:/opt/ispcms
```

### 5. Configure environment variables

```bash
cp .env.example .env
nano .env
```

Edit every value — especially these:

```env
# Database — use a strong password
DB_NAME=ispcms
DB_USER=ispcms
DB_PASSWORD=CHANGE_THIS_STRONG_PASSWORD

# JWT secret — minimum 32 characters, random
JWT_SECRET=CHANGE_THIS_TO_A_RANDOM_SECRET_MIN_32_CHARS

# Server
SERVER_ENV=production
CORS_ORIGINS=https://your-domain.com

# Initial super admin account
SUPER_ADMIN_EMAIL=admin@your-domain.com
SUPER_ADMIN_PASSWORD=CHANGE_THIS_STRONG_PASSWORD
SUPER_ADMIN_USERNAME=superadmin
```

> **Security:** Never commit `.env` to git. The `.gitignore` already excludes it.

### 6. Build and start

```bash
cd /opt/ispcms
docker compose up -d --build
```

This will:
- Build the Go backend image
- Build the Next.js frontend image
- Start PostgreSQL, backend, and frontend containers
- Run database migrations and seed default data automatically on first start

Verify all containers are running:

```bash
docker compose ps
```

You should see three containers — `ispcms_db`, `ispcms_backend`, `ispcms_frontend` — all with status `Up`.

Check logs if anything looks wrong:

```bash
docker compose logs -f backend
docker compose logs -f frontend
```

### 7. Install and configure Nginx

```bash
sudo apt install -y nginx
```

Create the site configuration:

```bash
sudo nano /etc/nginx/sites-available/ispcms
```

Paste the following (replace `your-domain.com` with your actual domain or server IP):

```nginx
server {
    listen 80;
    server_name your-domain.com;

    # Increase timeouts for long-running SNMP sync operations
    proxy_read_timeout 180s;
    proxy_connect_timeout 10s;
    proxy_send_timeout 30s;

    # Frontend
    location / {
        proxy_pass http://127.0.0.1:5869;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Backend API — direct pass-through
    location /api/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 180s;
    }
}
```

Enable the site and restart Nginx:

```bash
sudo ln -s /etc/nginx/sites-available/ispcms /etc/nginx/sites-enabled/
sudo nginx -t          # test config — must say "syntax is ok"
sudo systemctl restart nginx
sudo systemctl enable nginx
```

The app is now accessible at `http://your-domain.com`.

### 8. Enable HTTPS with Let's Encrypt (recommended)

> Requires a real domain name pointing to your server's IP.

```bash
sudo apt install -y certbot python3-certbot-nginx

sudo certbot --nginx -d your-domain.com

# Auto-renewal is set up automatically. Test it:
sudo certbot renew --dry-run
```

Certbot will update the Nginx config to redirect HTTP → HTTPS and add the SSL certificate. After this, also update your `.env`:

```env
CORS_ORIGINS=https://your-domain.com
```

Then restart the backend:

```bash
docker compose restart backend
```

### 9. (Optional) Run without a domain — IP only

If you don't have a domain and want to access via IP directly, use this simplified Nginx config:

```nginx
server {
    listen 80;
    server_name _;        # matches any hostname / bare IP

    proxy_read_timeout 180s;

    location / {
        proxy_pass http://127.0.0.1:5869;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 180s;
    }
}
```

Access the app at `http://<server-ip>`.

---

## Managing the Deployment

### Update to a new version

```bash
cd /opt/ispcms

# Pull latest code
git pull

# Rebuild and restart (zero-downtime: Compose restarts containers one by one)
docker compose up -d --build
```

### View logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f postgres
```

### Restart services

```bash
# Restart everything
docker compose restart

# Restart one service
docker compose restart backend
```

### Stop and start

```bash
docker compose down       # stop all containers (data is preserved in volumes)
docker compose up -d      # start again
```

### Backup the database

```bash
# Create a backup
docker exec ispcms_db pg_dump -U ispcms ispcms > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore from backup
docker exec -i ispcms_db psql -U ispcms ispcms < backup_20260101_120000.sql
```

### Check container status

```bash
docker compose ps
docker stats --no-stream    # CPU / memory usage
```

---

## Development

### Prerequisites
- Docker & Docker Compose
- Go 1.22 (for local backend work)
- Node.js 20+ (for local frontend work)

### Run locally

```bash
cp .env.example .env
# Edit .env if needed, then:
docker compose up -d
```

- Backend API: `http://localhost:8081/api/v1`
- Frontend: `http://localhost:5869`

Default login credentials are set via `SUPER_ADMIN_USERNAME` / `SUPER_ADMIN_PASSWORD` in `.env`.

### Rebuild a single service after code changes

```bash
docker compose build --no-cache frontend && docker compose up -d frontend
docker compose build --no-cache backend  && docker compose up -d backend
```

### Database schema

Schema is auto-migrated on startup via GORM `AutoMigrate`. No manual SQL migrations needed.

`PrepareSchema` runs before AutoMigrate on every startup to:
- Remove orphaned internet_account rows (old sync bug cleanup)
- Deduplicate rows by `(router_id, username)` if any exist
- Drop old indexes before recreating them
- Remove phantom router rows (created by an old save-cascade bug — now fixed)

---

## Architecture

```
backend/
  cmd/main.go                    Entry point; starts OLT sync scheduler
  internal/
    config/                      Env config
    database/                    DB connect + PrepareSchema + migrate + seed
    handlers/                    HTTP handlers (one file per module)
    middleware/                  JWT auth, RBAC permission checks
    models/                      GORM models
    repositories/                DB query layer
    router/                      Fiber app setup + route registration
    services/                    Business logic + OLT background scheduler
  pkg/
    snmp/                        SNMP client (gosnmp) — BulkWalk with GETNEXT fallback

frontend/
  src/
    app/(dashboard)/             Page routes (Next.js App Router)
    components/
      layout/                    Sidebar, Topbar
      ui/                        shadcn/ui components
    lib/
      api.ts                     Axios API client (all endpoints)
      utils.ts                   Date formatting, cn()
    types/index.ts               TypeScript interfaces for all models
```

---

## API Endpoints (key ones)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/dashboard/stats` | All stats including financial + activity timeline |
| `GET` | `/dashboard/activities` | Activity log with module + period filters |
| `GET` | `/internet-accounts/:id/billing-history` | Bills + last payment info for a customer |
| `PATCH` | `/bills/:id/status` | Record payment (amount, method, receipt, notes) |
| `POST` | `/internet-accounts/sync-all` | Sync all routers, returns SyncSummary |
| `GET` | `/snmp-profiles` | List SNMP vendor profiles |
| `POST` | `/olts` | Add OLT device |
| `POST` | `/olts/:id/sync` | Trigger manual OLT sync |
| `POST` | `/olts/:id/test` | Test SNMP connectivity |
| `GET` | `/olts/:id/pon-ports` | List PON ports for an OLT |
| `GET` | `/onus` | List ONUs (filterable by OLT, port, status, unlinked) |
| `PATCH` | `/onus/:id/link` | Link ONU to an internet account |
| `GET` | `/expenses` | List expenses with filters |
| `GET` | `/expenses/summary` | Expense totals by period and category |

---

## Roles & Permissions

| Role | Access |
|------|--------|
| `super_admin` | Full access to everything |
| `admin` | Full access except cannot delete roles |
| `billing_officer` | View+create+update on billing, packages, subscriptions, expenses; view-only everywhere else |
| `operator` | View+create+update on routers, PPPoE, internet accounts, and network (OLTs/ONUs); view-only on dashboard and packages |
| `viewer` | View-only across all modules |

Permissions are module + action pairs (`accounts.view`, `network.update`, etc.) assigned via **Roles & Permissions** in the UI. Only `super_admin` can modify role permission assignments.
