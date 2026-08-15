# IBMS — ISP Business Management System

A full-stack management platform for ISPs running MikroTik routers and PON/OLT equipment. Manages PPPoE customers, billing packages, monthly invoicing, payment collection, expenses, OLT/ONU network inventory, and role-based access control with account prefix scoping.

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
| **Profile Mappings** | Link MikroTik PPPoE profiles → billing packages (drives auto-subscription). Controlled by its own `profile_mappings` permission |
| **Subscriptions** | One active subscription per customer; auto-assigned on sync or manually set. Controlled by `subscriptions` permission |
| **Bills** | Monthly invoices — generate, view, collect payments. The **Collect** button shows all outstanding bills across months and applies payment oldest-first (carry-forward) |
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
| **SNMP Profiles** | Vendor-specific OID maps — define once, reuse across OLTs |

### Reports
| Page | Purpose |
|------|---------|
| **Collection Report** | Active User Collection — monthly billing report per customer showing bill, paid, due, status, collector. Includes summary cards, staff collection cards, per-package breakdown, daily bar chart, CSV export, and customer detail drawer. Prefix-scoped: billing officers see only their own accounts |

### Admin
| Page | Purpose |
|------|---------|
| **Users** | Staff accounts |
| **Roles & Permissions** | RBAC with granular module permissions. Non-admin roles support Account Prefix filtering |

> **Sidebar visibility:** Each page in the sidebar is shown only if the logged-in user has the required permission for that module. Users only see what they are allowed to access.

---

## Billing Flow

```
1. Create Packages          → define price, speed, VAT
2. Create Profile Mappings  → map MikroTik profile name → Package
3. Sync Routers             → auto-assigns subscriptions based on each account's profile
4. Generate Bills           → Bills page → "Generate Bills" → generates for current month
5. Collect Payments         → Bills page → click "Collect" on any row → pay all outstanding bills at once
6. View Customer Profile    → Internet Accounts → click any username
```

### How Bill Generation Works

Bills → **Generate Bills** → click Generate (current month only).

The system finds every non-archived, non-disabled account that had an active subscription on the **1st of the billing month**, creates one bill, and skips duplicates. A generation log records how many were created vs skipped and why.

**Package change rules:**
- If a customer changed packages on July 18, the July bill still uses the old package (billing date = July 1).
- August bill uses the new package.
- Historical bills are never recalculated.

### Collecting Payments (Carry-Forward)

Bills → find any bill for a customer → click **Collect**. The dialog shows:

- All unpaid/partial bills for that customer, ordered oldest first
- Per-row: Bill amount · Already paid · Still due
- **Total Outstanding** pre-filled in the amount field

Enter the amount (can be partial), select payment method and optional receipt number, click **Collect Payment**. The system distributes oldest-bill-first:

| Staff collects | Result |
|----------------|--------|
| ৳700 (Jul ৳200 + Aug ৳500 owed) | Jul → paid, Aug → paid |
| ৳200 | Jul → paid, Aug → still pending |
| ৳500 | Jul → paid, Aug → ৳300 applied (partial) |

Each disbursement creates a separate `PaymentRecord` per bill. When paid ≥ total amount, bill moves to `paid` automatically.

### Customer Profile (Billing History)

Internet Accounts → click any username. Shows:
- Customer info: username, router, current package, monthly charge
- Summary: total bills, total paid, total outstanding
- Full billing history: Month · Bill # · Amount · Paid · Due · Status · Payment Method · Receipt # · Collected By · Date

---

## Dashboard

The Dashboard shows real-time financial metrics. **Each section is visible only if the logged-in user has the required permission.** All financial data is automatically scoped to the user's account prefix.

### Collections (requires `billing: view`)

| Card | Shows |
|------|-------|
| Today's Collection | Payments received today |
| **Total Collected** | This calendar month's total (sub-text shows last month for comparison) |
| Outstanding Due | Sum of `due_amount` on all unpaid/partial bills |

### Expenses (requires `expenses: view`)

| Card | Shows |
|------|-------|
| Today's Expense | Expenses recorded today |
| **Total Expense** | This calendar month's expenses (sub-text shows last month) |
| Cash in Hand | This month's collection − this month's expense |

> Billing officers without `expenses: view` still see a **Cash in Hand** card.

### Other Dashboard Sections

| Section | Required Permission |
|---------|-------------------|
| Charts (Collection vs Expense, Monthly bar) | `billing: view` or `expenses: view` |
| Billing Stats (packages, subscriptions, bills) | `billing: view` |
| Network / Routers | `routers: view` |
| Recent Syncs | `routers: view` |
| Internet Accounts | `accounts: view` |
| Activity Timeline | `dashboard: view` (all users) |

---

## Collection Report

**Reports → Collection Report** — full payment picture for any billing month.

**Default month:** Opens on the current running month.

**Filters:** Month/Year · Payment Status (All/Paid/Partial/Unpaid/No Bill) · Router · Package · OLT · PON Port · Collector · Search

**Summary cards:** Total Active · Collected · Uncollected · Collection Amount · Total Bill · Total Due · Collection Rate %

### Staff Collection Cards

A card per billing officer who collected payments in the selected month. Each shows: staff name, total collected (৳), client count, share of total collection (progress bar).

- **Admin view:** Click any staff card to filter the table to bills collected by that person. Blue banner: *"Showing bills collected by [Name]"* with clear button.
- **Billing Officer view:** Sees only their own card and their own bills. Report is automatically scoped to their prefix accounts.

### Active Filter Banner

When a collector filter is active, a blue banner shows above the search bar: *"Showing bills collected by [Name] — ✕ Clear"*.

### Table & Export

Columns: Username · Package · Bill · Paid · Due · Status · Last Payment · Collected By

Click any row → **Customer Detail Drawer** (account info, ONU/network details, bill breakdown, last payment).

**CSV Export** downloads all rows in the current filtered view.

---

## Expense Tracking

Expenses → **Add Expense**. Each expense records:
- Date, category, amount, payment method
- Vendor and reference number (optional)
- Notes

Expense categories are managed separately (activate/deactivate). The Dashboard expense pie chart shows spending by category for the current month. Activity is logged for every create/update/delete.

---

## OLT / Network Module

### SNMP Profiles

Before adding an OLT, create (or use the seeded) SNMP profile for its vendor:

| Profile | Vendor | Technology |
|---------|--------|------------|
| BDCOM_EPON | BDCOM | EPON |
| VSOL_EPON | VSOL | EPON |
| CDATA_EPON | C-Data | EPON |
| Photon_EPON | Shenzhen Photon / C-Data (enterprise OID 12170) | EPON |
| HUAWEI_GPON | Huawei | GPON |
| ZTE_GPON | ZTE | GPON |
| RICHERLINK_EPON | Richerlink | EPON |
| RICHERLINK_EPON_V2 | Richerlink | EPON (firmware V1.0.0.32715+) |

**Special OID map keys:**

| Key | Purpose |
|-----|---------|
| `index_port_pos` | Position in OID suffix for port number (default `"0"`) |
| `index_onu_pos` | Position in OID suffix for ONU slot (default `"1"`) |
| `index_packed` | `"true"` — packed index encoding (Richerlink) |
| `index_cdata` | `"true"` — C-Data/Photon index encoding: suffix `A.B.C` maps to port=A, onu=B |
| `status_online_string` | Substring match for string-based online status |
| `power_divisor` | Divide raw power int by this to get dBm (default `"10"`) |
| `rx_power_negate` | `"true"` — negate RX power (VSOL absolute value) |
| `distance_unit` | `"m"` or `"cm"` — auto-converts cm to metres |
| `use_getnext` | `"true"` — force GETNEXT walk (no GETBULK support) |
| `link_by_name` | `"true"` — after MAC linking, also match ONU `serial_number` to account `username` (case-insensitive). Used for C-Data/Photon OLTs where the ONU label OID stores the PPPoE username |
| `cdata_port_onu_count_oid` | OID prefix for per-port ONU count table (C-Data/Photon only) — used to enumerate valid port/slot combos before walking ONU subtables |
| `onu_tx_power` | OID for TX optical power column (some vendors expose this in a separate subtable) |

### Photon_EPON Profile (C-Data / Shenzhen Photon, enterprise OID 12170)

This profile is **not seeded automatically**. Create it via API after deployment:

```bash
TOKEN="<your JWT token>"
curl -s -X POST http://<server>:8082/api/v1/snmp-profiles \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Photon_EPON",
    "vendor": "Photon",
    "technology": "EPON",
    "description": "Shenzhen Photon Broadband / C-Data OLT (enterprise OID 1.3.6.1.4.1.12170)",
    "oid_map": {
      "onu_serial":              "1.3.6.1.4.1.12170.2.3.4.1.1.2",
      "onu_mac":                 "1.3.6.1.4.1.12170.2.3.4.1.1.7",
      "onu_status":              "1.3.6.1.4.1.12170.2.3.4.1.1.8",
      "onu_distance":            "1.3.6.1.4.1.12170.2.3.4.1.1.15",
      "onu_rx_power":            "1.3.6.1.4.1.12170.2.3.4.2.1.4",
      "onu_tx_power":            "1.3.6.1.4.1.12170.2.3.4.2.1.5",
      "cdata_port_onu_count_oid":"1.3.6.1.4.1.12170.2.3.3.1.1.8",
      "power_divisor":           "100",
      "index_cdata":             "true",
      "distance_unit":           "m",
      "link_by_name":            "true"
    }
  }' | jq .
```

**How `link_by_name` works for C-Data/Photon:** The `onu_serial` OID (`…4.1.1.2`) stores the ONU's label as set on the OLT — operators typically name ONUs with the customer's PPPoE username. After the MAC-based auto-link pass, the system runs a second pass matching `LOWER(onus.serial_number) = LOWER(internet_accounts.username)`. ONUs that still don't have a match (e.g. never named on the OLT, or prefix mismatch) remain unlinked and can be linked manually via ONU Inventory.

### OLT Sync

Walks OIDs, discovers/upserts ONUs by `(olt_id, port_index, onu_slot)`, archives ONUs no longer visible, updates port counts, logs sync results. Background scheduler checks every minute for overdue syncs.

---

## PPPoE Sync

Idempotent — safe to run multiple times.

1. Orphaned rows cleaned first
2. Secrets upserted by `(router_id, username)`
3. MikroTik-only → created; IBMS-only → archived
4. Sessions fetched → `is_online`, `current_ip`, `uptime` updated
5. Profile mapping found → subscription auto-assigned

---

## Roles & Permissions

### Built-in Roles

| Role | Dashboard Sections | Access |
|------|-------------------|--------|
| `super_admin` | All | Full access to all modules |
| `admin` | All | Full access; cannot delete roles |
| `billing_officer` | Collections · Billing · Accounts · Cash in Hand · Activity | billing, profile_mappings, subscriptions, packages, notifications (view/create/update); accounts (view) |
| `operator` | Network · Accounts · Activity · Recent Syncs | routers, pppoe, accounts, network (view/create/update) |
| `viewer` | Sections matching granted permissions | View-only on assigned modules |

Permissions are module + action pairs assigned via **Roles & Permissions** in the UI. Only `super_admin` can modify permission assignments.

### Permission Modules

| Module | Controls |
|--------|---------|
| `dashboard` | Dashboard page access |
| `accounts` | Internet Accounts page |
| `routers` | Routers page + sync |
| `pppoe` | PPPoE Secrets + Active Sessions |
| `packages` | Packages page |
| `profile_mappings` | Profile Mappings page |
| `subscriptions` | Subscriptions page |
| `billing` | Bills page + payment collection |
| `notifications` | Notifications |
| `expenses` | Expenses + Expense Categories |
| `reports` | Collection Report |
| `network` | OLTs · ONU Inventory · SNMP Profiles |
| `users` | Users management |
| `roles` | Roles & Permissions |

### Sidebar Visibility

Every sidebar item is hidden automatically if the user lacks the required permission. Users see only pages they are allowed to access.

### Account Prefix Filter

Non-admin roles can be restricted to Internet Accounts whose username starts with specific prefixes.

**How to configure:**
1. Admin → Roles & Permissions → click the role card
2. Scroll to **Account Prefix Filter**
3. Type a prefix → press **Enter** (or click **Save Prefixes** directly)
4. Add multiple prefixes — the role sees accounts matching any of them

**Behaviour:**
- Prefixes set (e.g. `SUN`, `AB`) → user sees only accounts starting with those prefixes
- No prefixes set → user sees **no accounts** (fully blocked)
- `admin` / `super_admin` → always unrestricted

**Prefix filter is enforced system-wide:**
- Internet Accounts page
- Bills page
- Collection Report
- Dashboard (collections, outstanding due, bill counts, account stats all scoped to prefix accounts)

**Example:**

| User | Prefix | Sees |
|------|--------|------|
| Hanif (billing_officer) | `SUN` | SUN_001, SUN_Karim, … |
| Sany (billing_officer) | `AB` | AB_Sany, AB_Rahman, … |

---

## Deploy to Ubuntu (Production)

### 1. Install Docker

```bash
sudo apt update && sudo apt install -y git curl
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER && newgrp docker
```

### 2. Clone & Configure

```bash
cd ~
git clone https://github.com/your-org/ispcms.git
cd ispcms
cp .env.example .env
nano .env   # set DB_PASSWORD, JWT_SECRET, SUPER_ADMIN_PASSWORD, CORS_ORIGINS
```

### 3. Build & Start

```bash
docker compose up -d --build
docker compose ps   # all three containers should be Up
```

### 4. Nginx (IP-only access)

```bash
sudo apt install -y nginx
sudo nano /etc/nginx/sites-available/ispcms
```

```nginx
server {
    listen 80;
    server_name _;
    proxy_read_timeout 180s;

    location / {
        proxy_pass http://127.0.0.1:5869;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8082;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 180s;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/ispcms /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl restart nginx && sudo systemctl enable nginx
```

### 5. Update to a New Version

```bash
cd ~/ispcms

# Fetch and reset to origin/main (safer than git pull if local state is dirty)
git fetch origin
git reset --hard origin/main

# Rebuild only the backend (faster than rebuilding everything)
sudo docker compose build --no-cache backend
sudo docker compose up -d
```

> **Note:** If the project is managed via a systemd service (`sudo systemctl restart ispcms`), that service typically runs `docker compose up -d`. You still need to run `docker compose build --no-cache backend` first so the new image is ready before restarting.

After update, check logs to confirm the migration completed without errors:

```bash
docker compose logs backend | grep -E "PrepareSchema|AutoMigrate|Seed|ERROR" | head -30
```

### Useful Commands

```bash
# Logs
docker compose logs -f backend
docker compose logs -f frontend

# Restart
docker compose restart backend

# Database backup
docker exec ispcms_db pg_dump -U ispcms ispcms > backup_$(date +%Y%m%d).sql
```

---

## Development

```bash
cp .env.example .env
docker compose up -d
```

- Backend API: `http://localhost:8082/api/v1`
- Frontend: `http://localhost:5869`

Rebuild a single service after code changes:

```bash
docker compose build --no-cache backend && docker compose up -d backend
docker compose build --no-cache frontend && docker compose up -d frontend
```

Schema is auto-migrated on every startup — no manual SQL needed. The startup sequence is:

1. `PrepareSchema` — idempotent raw SQL fixups (e.g. `ALTER TABLE … ADD COLUMN IF NOT EXISTS`) for columns that GORM AutoMigrate cannot reliably add (custom Valuer/Scanner types like `oid_map`)
2. `AutoMigrate` — GORM schema sync for all models
3. `Seed` — inserts/updates built-in SNMP profiles and default roles

If you see `column "oid_map" of relation "snmp_profiles" does not exist` errors, the container is running old code — rebuild with `--no-cache`.

---

## Architecture

```
backend/
  cmd/server/                    Entry point; starts OLT sync scheduler
  internal/
    config/                      Env config
    database/                    DB connect, PrepareSchema, migrate, seed
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
      layout/                    Sidebar (permission-filtered), Topbar
      roles/                     PermissionMatrix, AccountPrefixEditor
      ui/                        shadcn/ui components
    contexts/AuthContext.tsx     Auth state + hasPermission() + hasRole()
    lib/api.ts                   Axios API client (all endpoints)
    types/index.ts               TypeScript interfaces for all models
```

---

## Key API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/dashboard/stats` | Financial stats + activity — prefix-scoped per user |
| `GET` | `/reports/active-user-collection` | Collection report — prefix-scoped per user |
| `GET` | `/bills/account-due?internet_account_id=` | All outstanding bills for a customer |
| `POST` | `/bills/collect` | Collect payment across all unpaid bills oldest-first |
| `GET` | `/internet-accounts` | Account list — prefix-scoped |
| `GET` | `/bills` | Bill list — prefix-scoped |
| `PUT` | `/roles/:id/account-prefixes` | Set account prefix filter for a role |
| `PUT` | `/roles/:id/permissions` | Set all permissions for a role |
| `POST` | `/olts/:id/sync` | Trigger manual OLT sync |
| `PATCH` | `/onus/:id/link` | Link ONU to an internet account |
