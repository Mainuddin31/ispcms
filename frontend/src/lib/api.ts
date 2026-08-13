import axios, { AxiosError, InternalAxiosRequestConfig } from "axios";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "/api/v1";

export const api = axios.create({
  baseURL: API_URL,
  headers: { "Content-Type": "application/json" },
});

// Attach access token to every request
api.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("access_token");
    if (token) config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// On 401, attempt refresh or redirect to login
api.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as InternalAxiosRequestConfig & { _retry?: boolean };
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true;
      const refresh = typeof window !== "undefined" ? localStorage.getItem("refresh_token") : null;
      if (refresh) {
        try {
          const { data } = await axios.post(`${API_URL}/auth/refresh`, { refresh_token: refresh });
          localStorage.setItem("access_token", data.data.access_token);
          localStorage.setItem("refresh_token", data.data.refresh_token);
          original.headers.Authorization = `Bearer ${data.data.access_token}`;
          return api(original);
        } catch {
          localStorage.clear();
          window.location.href = "/login";
        }
      } else {
        localStorage.clear();
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  }
);

// ─── Auth ────────────────────────────────────────────────────────────────────
export const authApi = {
  login: (username: string, password: string) =>
    api.post("/auth/login", { username, password }),
  logout: () => api.post("/auth/logout"),
  refresh: (refresh_token: string) => api.post("/auth/refresh", { refresh_token }),
  profile: () => api.get("/auth/profile"),
  changePassword: (old_password: string, new_password: string) =>
    api.put("/auth/change-password", { old_password, new_password }),
  resetPassword: (userId: string, new_password: string) =>
    api.post(`/users/${userId}/reset-password`, { new_password }),
};

// ─── Dashboard ────────────────────────────────────────────────────────────────
export const dashboardApi = {
  stats: () => api.get("/dashboard/stats"),
  activities: (params?: { module?: string; period?: string; limit?: number }) =>
    api.get("/dashboard/activities", { params }),
};

// ─── Users ────────────────────────────────────────────────────────────────────
export const usersApi = {
  list: (params?: { page?: number; page_size?: number; search?: string; status?: string }) =>
    api.get("/users", { params }),
  get: (id: string) => api.get(`/users/${id}`),
  create: (data: object) => api.post("/users", data),
  update: (id: string, data: object) => api.put(`/users/${id}`, data),
  delete: (id: string) => api.delete(`/users/${id}`),
  assignRole: (userId: string, role_id: string) =>
    api.post(`/users/${userId}/roles`, { role_id }),
  removeRole: (userId: string, roleId: string) =>
    api.delete(`/users/${userId}/roles/${roleId}`),
  setStatus: (userId: string, status: string) =>
    api.patch(`/users/${userId}/status`, { status }),
};

// ─── Roles ────────────────────────────────────────────────────────────────────
export const rolesApi = {
  list: () => api.get("/roles"),
  get: (id: string) => api.get(`/roles/${id}`),
  create: (data: object) => api.post("/roles", data),
  update: (id: string, data: object) => api.put(`/roles/${id}`, data),
  delete: (id: string) => api.delete(`/roles/${id}`),
  permissions: () => api.get("/roles/permissions"),
  setPermissions: (roleId: string, permission_ids: string[]) =>
    api.put(`/roles/${roleId}/permissions`, { permission_ids }),
  setAccountPrefixes: (roleId: string, prefixes: string[]) =>
    api.put(`/roles/${roleId}/account-prefixes`, { prefixes }),
};

// ─── Routers ──────────────────────────────────────────────────────────────────
export const routersApi = {
  list: (params?: { page?: number; page_size?: number; search?: string }) =>
    api.get("/routers", { params }),
  get: (id: string) => api.get(`/routers/${id}`),
  create: (data: object) => api.post("/routers", data),
  update: (id: string, data: object) => api.put(`/routers/${id}`, data),
  delete: (id: string) => api.delete(`/routers/${id}`),
  testById: (id: string) => api.post(`/routers/${id}/test`),
  testRaw: (data: object) => api.post("/routers/test-connection", data),
  sync: (id: string) => api.post(`/routers/${id}/sync`, {}, { timeout: 120000 }),
  syncLogs: (id: string, limit?: number) =>
    api.get(`/routers/${id}/sync-logs`, { params: { limit } }),
};

// ─── Internet Accounts ────────────────────────────────────────────────────────
export const internetAccountsApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    search?: string;
    router_id?: string;
    profile?: string;
    is_online?: boolean;
    disabled?: boolean;
    archived?: boolean;
  }) => api.get("/internet-accounts", { params }),
  get: (id: string) => api.get(`/internet-accounts/${id}`),
  stats: () => api.get("/internet-accounts/stats"),
  profiles: () => api.get("/internet-accounts/profiles"),
  syncAll: () => api.post("/internet-accounts/sync-all", {}, { timeout: 120000 }),
};

// ─── Packages ─────────────────────────────────────────────────────────────────
export const packagesApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; search?: string }) =>
    api.get("/packages", { params }),
  listActive: () => api.get("/packages/active"),
  get: (id: string) => api.get(`/packages/${id}`),
  create: (data: object) => api.post("/packages", data),
  update: (id: string, data: object) => api.put(`/packages/${id}`, data),
  delete: (id: string) => api.delete(`/packages/${id}`),
};

// ─── Profile Mappings ─────────────────────────────────────────────────────────
export const profileMappingsApi = {
  list: (params?: { page?: number; page_size?: number; package_id?: string; search?: string }) =>
    api.get("/profile-mappings", { params }),
  unmapped: () => api.get("/profile-mappings/unmapped"),
  get: (id: string) => api.get(`/profile-mappings/${id}`),
  create: (data: object) => api.post("/profile-mappings", data),
  update: (id: string, data: object) => api.put(`/profile-mappings/${id}`, data),
  delete: (id: string) => api.delete(`/profile-mappings/${id}`),
};

// ─── Subscriptions ────────────────────────────────────────────────────────────
export const subscriptionsApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    internet_account_id?: string;
    package_id?: string;
    active_only?: boolean;
  }) => api.get("/subscriptions", { params }),
  getActiveForAccount: (accountId: string) =>
    api.get(`/subscriptions/account/${accountId}`),
  assign: (internet_account_id: string, package_id: string) =>
    api.post("/subscriptions", { internet_account_id, package_id }),
  autoAssign: () => api.post("/subscriptions/auto-assign"),
};

// ─── Bills ────────────────────────────────────────────────────────────────────
export const billsApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    status?: string;
    month?: number;
    year?: number;
    internet_account_id?: string;
    package_id?: string;
    search?: string;
  }) => api.get("/bills", { params }),
  get: (id: string) => api.get(`/bills/${id}`),
  generate: (month?: number, year?: number) => api.post("/bills/generate", { month, year }),
  updateStatus: (id: string, data: { status?: string; paid_amount?: number; notes?: string; payment_method?: string; receipt_number?: string }) =>
    api.patch(`/bills/${id}/status`, data),
  status: (month?: number, year?: number) =>
    api.get("/bills/status", { params: { month, year } }),
  generationLogs: (params?: { month?: number; year?: number; limit?: number }) =>
    api.get("/bills/generation-logs", { params }),
  accountDue: (internetAccountId: string) =>
    api.get("/bills/account-due", { params: { internet_account_id: internetAccountId } }),
  collect: (data: {
    internet_account_id: string;
    amount: number;
    payment_method?: string;
    notes?: string;
    receipt_number?: string;
  }) => api.post("/bills/collect", data),
};

// ─── Payment History ──────────────────────────────────────────────────────────
export const paymentHistoryApi = {
  listByAccount: (internetAccountId: string) =>
    api.get(`/internet-accounts/${internetAccountId}/payment-history`),
};

// ─── Billing History ──────────────────────────────────────────────────────────
export const billingHistoryApi = {
  listByAccount: (internetAccountId: string) =>
    api.get(`/internet-accounts/${internetAccountId}/billing-history`),
};

// ─── Notifications ────────────────────────────────────────────────────────────
export const notificationsApi = {
  list: (params?: { page?: number; page_size?: number; unread_only?: boolean }) =>
    api.get("/notifications", { params }),
  unreadCount: () => api.get("/notifications/unread-count"),
  markRead: (id: string) => api.patch(`/notifications/${id}/read`),
  markAllRead: () => api.post("/notifications/mark-all-read"),
};

// ─── Expense Categories ───────────────────────────────────────────────────────
export const expenseCategoriesApi = {
  list: (status?: string) => api.get("/expense-categories", { params: { status: status ?? "all" } }),
  create: (data: { name: string; description?: string }) =>
    api.post("/expense-categories", data),
  update: (id: string, data: { name?: string; description?: string; status?: string }) =>
    api.put(`/expense-categories/${id}`, data),
  delete: (id: string) => api.delete(`/expense-categories/${id}`),
};

// ─── Expenses ─────────────────────────────────────────────────────────────────
export const expensesApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    search?: string;
    category_id?: string;
    payment_method?: string;
    user_id?: string;
    date_from?: string;
    date_to?: string;
    amount_min?: number;
    amount_max?: number;
    sort_by?: string;
    sort_dir?: string;
  }) => api.get("/expenses", { params }),
  get: (id: string) => api.get(`/expenses/${id}`),
  create: (data: {
    expense_date: string;
    category_id: string;
    amount: number;
    payment_method?: string;
    vendor?: string;
    reference_number?: string;
    description?: string;
    attachment_path?: string;
  }) => api.post("/expenses", data),
  update: (id: string, data: {
    expense_date?: string;
    category_id?: string;
    amount?: number;
    payment_method?: string;
    vendor?: string;
    reference_number?: string;
    description?: string;
    attachment_path?: string;
  }) => api.put(`/expenses/${id}`, data),
  delete: (id: string) => api.delete(`/expenses/${id}`),
  summary: () => api.get("/expenses/summary"),
};

// ─── SNMP Profiles ────────────────────────────────────────────────────────────
export const snmpProfilesApi = {
  list: () => api.get("/snmp-profiles"),
  get: (id: string) => api.get(`/snmp-profiles/${id}`),
  create: (data: { name: string; vendor: string; technology: string; oid_map: Record<string, string>; description?: string }) =>
    api.post("/snmp-profiles", data),
  update: (id: string, data: { name: string; vendor: string; technology: string; oid_map: Record<string, string>; description?: string }) =>
    api.put(`/snmp-profiles/${id}`, data),
  delete: (id: string) => api.delete(`/snmp-profiles/${id}`),
};

// ─── OLTs ────────────────────────────────────────────────────────────────────
export const oltsApi = {
  list: (params?: { search?: string; status?: string }) => api.get("/olts", { params }),
  get: (id: string) => api.get(`/olts/${id}`),
  stats: () => api.get("/olts/stats"),
  recentSyncLogs: (limit?: number) => api.get("/olts/sync-logs", { params: { limit } }),
  testConnectionRaw: (data: { ip: string; community: string; port?: number; version?: string }) =>
    api.post("/olts/test-connection", data),
  create: (data: object) => api.post("/olts", data),
  update: (id: string, data: object) => api.put(`/olts/${id}`, data),
  delete: (id: string) => api.delete(`/olts/${id}`),
  sync: (id: string) => api.post(`/olts/${id}/sync`, {}, { timeout: 120000 }),
  testConnection: (id: string) => api.post(`/olts/${id}/test`),
  syncLogs: (id: string, limit?: number) => api.get(`/olts/${id}/sync-logs`, { params: { limit } }),
  ponPorts: (id: string) => api.get(`/olts/${id}/pon-ports`),
};

// ─── ONUs ────────────────────────────────────────────────────────────────────
export const onusApi = {
  list: (params?: {
    page?: number;
    page_size?: number;
    search?: string;
    olt_id?: string;
    pon_port_id?: string;
    status?: string;
    unlinked?: boolean;
  }) => api.get("/onus", { params }),
  get: (id: string) => api.get(`/onus/${id}`),
  link: (id: string, internet_account_id: string | null) =>
    api.patch(`/onus/${id}/link`, { internet_account_id }),
  // Auto-link all unlinked ONUs to internet accounts via MAC/caller_id matching
  autoLink: (oltId?: string) =>
    oltId
      ? api.post(`/olts/${oltId}/auto-link-onus`)
      : api.post("/onus/auto-link"),
};

// ─── Reports ──────────────────────────────────────────────────────────────────
export const reportsApi = {
  activeUserCollection: (params: {
    billing_month?: number;
    billing_year?: number;
    payment_status?: string;
    package_id?: string;
    router_id?: string;
    olt_id?: string;
    pon_port_id?: string;
    collector_id?: string;
    search?: string;
    page?: number;
    page_size?: number;
  }) => api.get("/reports/active-user-collection", { params }),
};

// ─── Visiting ────────────────────────────────────────────────────────────────
export const visitingApi = {
  pendingCustomers: (params?: { month?: number; year?: number }) =>
    api.get("/visits/pending-customers", { params }),
  today: () => api.get("/visits/today"),
  list: (params?: {
    page?: number;
    page_size?: number;
    status?: string;
    date_from?: string;
    date_to?: string;
    date_preset?: "today" | "tomorrow" | "this_week";
    assigned_staff_id?: string;
    search?: string;
  }) => api.get("/visits", { params }),
  get: (id: string) => api.get(`/visits/${id}`),
  create: (data: {
    internet_account_id: string;
    bill_id: string;
    billing_month: number;
    billing_year: number;
    assigned_staff_id: string;
    scheduled_date: string;
    scheduled_time: string;
    notes?: string;
  }) => api.post("/visits", data),
  update: (id: string, data: {
    scheduled_date?: string;
    scheduled_time?: string;
    assigned_staff_id?: string;
    notes?: string;
  }) => api.put(`/visits/${id}`, data),
  complete: (id: string) => api.post(`/visits/${id}/complete`, {}),
  reschedule: (id: string, data: {
    scheduled_date: string;
    scheduled_time: string;
    assigned_staff_id?: string;
    notes?: string;
  }) => api.post(`/visits/${id}/reschedule`, data),
  cancel: (id: string) => api.post(`/visits/${id}/cancel`, {}),
  byAccount: (internetAccountId: string) =>
    api.get(`/internet-accounts/${internetAccountId}/visits`),
};

// ─── PPPoE ────────────────────────────────────────────────────────────────────
export const pppoeApi = {
  secrets: (params?: {
    page?: number;
    page_size?: number;
    search?: string;
    router_id?: string;
    disabled?: boolean;
  }) => api.get("/pppoe/secrets", { params }),
  secret: (id: string) => api.get(`/pppoe/secrets/${id}`),
  sessions: (params?: { router_id?: string }) =>
    api.get("/pppoe/sessions", { params }),
};
