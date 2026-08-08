# IBMS Frontend

Next.js 14 App Router frontend for the ISP Business Management System.

**Stack:** Next.js 14 · TypeScript · Tailwind CSS · shadcn/ui · TanStack React Query · Axios

---

## Project Structure

```
frontend/src/
├── app/
│   ├── layout.tsx               Root layout (QueryClient provider, Toaster)
│   ├── page.tsx                 Redirect → /dashboard
│   ├── login/
│   │   └── page.tsx             Login form
│   └── (dashboard)/             Auth-gated layout with Sidebar + Topbar
│       ├── layout.tsx           Checks localStorage for token, redirects if missing
│       ├── dashboard/page.tsx   Stats overview (routers, accounts, billing widgets)
│       ├── routers/page.tsx     Router CRUD + sync trigger
│       ├── internet-accounts/   PPPoE account list (online/offline, caller ID, IPs)
│       ├── pppoe/               PPPoE secrets (legacy view)
│       ├── sessions/            Active PPPoE sessions
│       ├── packages/            Billing package CRUD
│       ├── profile-mappings/    MikroTik profile → package mapping + unmapped warning
│       ├── subscriptions/       Customer subscriptions (auto + manual assign)
│       ├── bills/               Monthly bills — generate, filter, record payments
│       ├── notifications/       System alerts, mark read
│       ├── users/               Staff management
│       └── roles/               Role & permission management
├── components/
│   ├── layout/
│   │   ├── Sidebar.tsx          Collapsible nav with section dividers
│   │   └── Topbar.tsx           Page title + breadcrumbs
│   └── ui/                      shadcn/ui components (button, table, dialog, etc.)
├── lib/
│   ├── api.ts                   Axios instance + all API methods grouped by module
│   └── utils.ts                 cn(), formatDate(), formatRelative()
└── types/
    └── index.ts                 TypeScript interfaces for all backend models
```

---

## Environment Variables

Create `frontend/.env.local`:

```env
NEXT_PUBLIC_API_URL=/api/v1
```

In production the Next.js rewrite proxies `/api/v1/*` → `http://ispcms_backend:8080/api/v1/*` (configured in `next.config.js`). In local dev with the backend running on port 8081, set:

```env
NEXT_PUBLIC_API_URL=http://localhost:8081/api/v1
```

---

## Running Locally

```bash
cd frontend
npm install
npm run dev        # http://localhost:3000
```

Or via Docker Compose from the project root:

```bash
docker compose up -d
```

---

## Pages

### Infrastructure
| Route | Description |
|-------|-------------|
| `/dashboard` | Stats cards: routers, internet accounts, billing summary |
| `/routers` | Add/edit MikroTik routers, test connection, sync |
| `/internet-accounts` | All PPPoE accounts with online status, IPs, caller ID |
| `/pppoe` | Raw PPPoE secrets from RouterOS |
| `/sessions` | Live active sessions |

### Billing
| Route | Description |
|-------|-------------|
| `/packages` | Create/edit billing packages (price, speed, VAT) |
| `/profile-mappings` | Map MikroTik profile names to billing packages. Shows unmapped profiles as clickable chips — click to open the create dialog pre-filled with that profile name |
| `/subscriptions` | View all customer subscriptions. Manual assign available for accounts not auto-assigned by sync |
| `/bills` | Filter bills by month/year/status. **Generate Bills** button creates bills for all subscribed accounts. **Record Payment** per bill |
| `/notifications` | System alerts — unmapped profiles, bill generation results. Unread badge in sidebar |

### Admin
| Route | Description |
|-------|-------------|
| `/users` | Staff accounts |
| `/roles` | Roles and permission assignment |

---

## API Client (`src/lib/api.ts`)

All API groups:

```ts
authApi         // login, logout, refresh, profile, changePassword
dashboardApi    // stats
usersApi        // CRUD + role assignment
rolesApi        // CRUD + permissions
routersApi      // CRUD + test + sync
internetAccountsApi  // list, get, stats, profiles, syncAll
pppoeApi        // secrets, sessions
packagesApi     // CRUD + listActive
profileMappingsApi   // CRUD + unmapped
subscriptionsApi     // list, assign, getActiveForAccount
billsApi        // list, get, generate, updateStatus, status, generationLogs
notificationsApi     // list, unreadCount, markRead, markAllRead
```

---

## Auth Flow

1. `POST /auth/login` → stores `access_token` + `refresh_token` in `localStorage`
2. Axios interceptor attaches `Authorization: Bearer <token>` to every request
3. On 401 → tries `POST /auth/refresh` → if it fails, clears storage and redirects to `/login`
4. Dashboard layout checks `localStorage` on mount, redirects to `/login` if no token
