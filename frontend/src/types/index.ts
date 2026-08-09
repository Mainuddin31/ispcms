export interface User {
  id: string;
  full_name: string;
  username: string;
  email: string;
  phone?: string;
  status: "active" | "disabled";
  avatar?: string;
  last_login?: string;
  created_at: string;
  updated_at: string;
  roles?: Role[];
}

export interface Role {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  account_prefixes?: string[];
  permissions?: Permission[];
  created_at: string;
  updated_at: string;
}

export interface Permission {
  id: string;
  name: string;
  module: string;
  action: string;
}

export interface Router {
  id: string;
  name: string;
  ip_address: string;
  api_port: number;
  username: string;
  location?: string;
  pop_name?: string;
  description?: string;
  status: "active" | "disabled";
  connection_status: "connected" | "disconnected";
  last_connected?: string;
  last_sync_time?: string;
  created_at: string;
  updated_at: string;
}

export interface PPPoESecret {
  id: string;
  router_id: string;
  routeros_id: string;
  username: string;
  password?: string;
  profile?: string;
  service?: string;
  local_address?: string;
  remote_address?: string;
  caller_id?: string;
  disabled: boolean;
  comment?: string;
  last_seen?: string;
  sync_time: string;
  created_at: string;
  updated_at: string;
  router?: Router;
}

export interface PPPoESession {
  id: string;
  router_id: string;
  routeros_id: string;
  username: string;
  current_ip?: string;
  uptime?: string;
  session_id?: string;
  connected_since?: string;
  sync_time: string;
  created_at: string;
}

export interface SyncLog {
  id: string;
  router_id: string;
  status: "running" | "success" | "failed";
  secrets_total: number;
  secrets_created: number;
  secrets_updated: number;
  secrets_deleted: number;
  sessions_total: number;
  new_accounts: number;
  updated_accounts: number;
  archived_accounts: number;
  online_count: number;
  offline_count: number;
  error_message?: string;
  duration_ms: number;
  started_at: string;
  completed_at?: string;
  router?: Router;
}

export interface SyncSummary {
  routers_processed: number;
  routers_succeeded: number;
  routers_failed: number;
  total_secrets: number;
  new_accounts: number;
  updated_accounts: number;
  archived_accounts: number;
  online_users: number;
  offline_users: number;
  duration_ms: number;
  logs: SyncLog[];
  errors: string[];
}

export interface ActivityLog {
  id: string;
  user_id?: string;
  module: string;
  activity_type: string;
  title: string;
  description: string;
  reference_type?: string;
  reference_id?: string;
  // legacy fields
  action: string;
  resource_id?: string;
  details?: string;
  ip_address?: string;
  created_at: string;
  user?: {
    id: string;
    full_name: string;
    username: string;
  };
}

export interface InternetAccount {
  id: string;
  router_id: string;
  mikrotik_secret_id: string;
  username: string;
  password?: string;
  service?: string;
  profile?: string;
  local_address?: string;
  remote_address?: string;
  caller_id?: string;
  comment?: string;
  disabled: boolean;
  is_online: boolean;
  current_ip?: string;
  session_id?: string;
  uptime?: string;
  connected_since?: string;
  last_seen?: string;
  sync_status: "synced" | "missing";
  last_sync_at?: string;
  archived_at?: string;
  created_at: string;
  updated_at: string;
  router?: Router;
  onu?: ONU;
}

export interface InternetAccountStats {
  total: number;
  enabled: number;
  disabled: number;
  online: number;
  offline: number;
  archived: number;
}

// ─── Billing Module ───────────────────────────────────────────────────────────

export interface Package {
  id: string;
  package_name: string;
  display_name: string;
  speed?: string;
  monthly_price: number;
  vat_percent: number;
  installation_charge: number;
  description?: string;
  status: "active" | "inactive";
  created_at: string;
  updated_at: string;
}

export interface ProfileMapping {
  id: string;
  mikrotik_profile: string;
  package_id: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  package?: Package;
}

export interface CustomerSubscription {
  id: string;
  internet_account_id: string;
  package_id: string;
  monthly_price: number;
  effective_from: string;
  effective_until?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  internet_account?: InternetAccount;
  package?: Package;
}

export interface MonthlyBill {
  id: string;
  bill_number: string;
  internet_account_id: string;
  package_id: string;
  subscription_id: string;
  billing_month: number;
  billing_year: number;
  monthly_charge: number;
  discount: number;
  fine: number;
  vat: number;
  total_amount: number;
  paid_amount: number;
  due_amount: number;
  status: "pending" | "due" | "partial" | "paid" | "cancelled";
  due_date?: string;
  notes?: string;
  generated_at: string;
  created_at: string;
  updated_at: string;
  internet_account?: InternetAccount;
  package?: Package;
}

export interface PaymentRecord {
  id: string;
  bill_id: string;
  internet_account_id: string;
  amount: number;
  notes?: string;
  payment_method: string; // cash | bkash | bank | card | other
  receipt_number?: string;
  received_by_id?: string;
  paid_at: string;
  created_at: string;
  bill?: MonthlyBill;
  received_by?: { id: string; full_name: string; username: string };
}

export interface BillingHistoryEntry extends MonthlyBill {
  last_payment_method?: string;
  last_receipt_number?: string;
  last_collected_by?: string;
  last_paid_at?: string;
}

export interface BillSkipDetail {
  account_id: string;
  username: string;
  reason: string;
}

export interface BillGenerationLog {
  id: string;
  billing_month: number;
  billing_year: number;
  total_accounts: number;
  bills_generated: number;
  bills_skipped: number;
  skip_details?: BillSkipDetail[];
  status: "completed" | "partial" | "failed";
  generated_by_id?: string;
  generated_at: string;
  created_at: string;
}

export interface Notification {
  id: string;
  type: string;
  title: string;
  message: string;
  severity: "info" | "warning" | "error" | "success";
  entity_type?: string;
  entity_id?: string;
  recipient_roles?: string[];
  is_read: boolean;
  read_at?: string;
  created_at: string;
  updated_at: string;
}

// ─── Dashboard ────────────────────────────────────────────────────────────────

export interface MonthlyPoint {
  year: number;
  month: number;
  label: string;
  collection: number;
  expense: number;
  cash_in_hand: number;
}

export interface DashboardStats {
  total_routers: number;
  online_routers: number;
  offline_routers: number;
  total_pppoe_secrets: number;
  active_pppoe_users: number;
  disabled_pppoe_users: number;
  active_sessions: number;
  last_sync_time?: string;
  // Internet account stats
  total_accounts: number;
  enabled_accounts: number;
  disabled_accounts: number;
  online_accounts: number;
  offline_accounts: number;
  archived_accounts: number;
  // Billing stats
  total_packages: number;
  active_packages: number;
  active_subscriptions: number;
  unmapped_profiles: number;
  bills_this_month: number;
  bills_pending_generate: number;
  // Financial — collection
  today_collection: number;
  monthly_collection: number;
  last_month_collection: number;
  total_collection: number;
  total_outstanding_due: number;
  total_bills_generated: number;
  bills_paid: number;
  bills_billing_pending: number;
  // Financial — expenses
  today_expense: number;
  monthly_expense: number;
  total_expense: number;
  // Derived
  cash_in_hand: number;
  // Charts
  monthly_chart: MonthlyPoint[];
  expense_category_pie: CategoryTotal[];
  // Activity
  recent_sync_logs: SyncLog[];
  recent_activities: ActivityLog[];
}

// ─── Expense Module ───────────────────────────────────────────────────────────

export interface ExpenseCategory {
  id: string;
  name: string;
  description?: string;
  status: "active" | "inactive";
  created_at: string;
  updated_at: string;
}

export interface Expense {
  id: string;
  expense_number: string;
  expense_date: string;
  category_id: string;
  amount: number;
  payment_method: "cash" | "bank" | "mobile" | "cheque" | "card" | "other";
  vendor?: string;
  reference_number?: string;
  description?: string;
  attachment_path?: string;
  created_by_id?: string;
  updated_by_id?: string;
  deleted_by_id?: string;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  category?: ExpenseCategory;
  created_by?: { id: string; full_name: string; username: string };
  updated_by?: { id: string; full_name: string; username: string };
  deleted_by?: { id: string; full_name: string; username: string };
}

export interface CategoryTotal {
  category_id: string;
  category_name: string;
  total: number;
}

export interface ExpenseSummary {
  today_total: number;
  week_total: number;
  month_total: number;
  year_total: number;
  all_time_total: number;
  category_totals: CategoryTotal[];
}

// ─── OLT / Network Module ─────────────────────────────────────────────────────

export type OIDMap = Record<string, string>;

export interface SNMPProfile {
  id: string;
  name: string;
  vendor: string;
  technology: "EPON" | "GPON";
  oid_map: OIDMap;
  description?: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface OLT {
  id: string;
  name: string;
  vendor?: string;
  model?: string;
  snmp_profile_id: string;
  snmp_profile?: SNMPProfile;
  management_ip: string;
  snmp_version: "v2c" | "v3";
  snmp_port: number;
  timeout: number;
  retries: number;
  community?: string;
  v3_username?: string;
  v3_auth_protocol?: string;
  v3_priv_protocol?: string;
  pop?: string;
  rack?: string;
  cabinet?: string;
  description?: string;
  status: "active" | "maintenance" | "offline" | "disabled";
  sync_interval: number;
  last_sync_at?: string;
  deleted_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PONPort {
  id: string;
  olt_id: string;
  olt?: OLT;
  port_index: number;
  port_name: string;
  onu_count: number;
  max_onus: number;
  status: string;
  archived_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ONU {
  id: string;
  olt_id: string;
  olt?: OLT;
  pon_port_id: string;
  pon_port?: PONPort;
  port_index: number;
  onu_slot: number;
  onu_id: string;
  mac_address?: string;
  serial_number?: string;
  vendor?: string;
  model?: string;
  status: "online" | "offline";
  reg_status: "registered" | "deregistered";
  distance?: number;
  rx_power?: number;
  tx_power?: number;
  last_online_at?: string;
  archived_at?: string;
  internet_account_id?: string;
  internet_account?: InternetAccount;
  created_at: string;
  updated_at: string;
}

export interface OLTSyncLog {
  id: string;
  olt_id: string;
  olt?: OLT;
  status: "running" | "success" | "failed";
  started_at: string;
  completed_at?: string;
  duration_ms: number;
  ports_discovered: number;
  onus_discovered: number;
  new_onus: number;
  updated_onus: number;
  archived_onus: number;
  error_message?: string;
}

export interface OLTStats {
  total_olts: number;
  active_olts: number;
  total_pon_ports: number;
  total_onus: number;
  online_onus: number;
  offline_onus: number;
  unassigned_onus: number;
  port_utilization_pct: number;
}

export interface PaginatedResponse<T> {
  success: boolean;
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

// ─── Collection Report ────────────────────────────────────────────────────────

export interface CollectionRow {
  account_id: string;
  customer_name: string;
  username: string;
  router_name: string;
  router_id: string;
  package_name: string;
  package_id: string;
  monthly_charge: number;
  bill_id: string;
  bill_number: string;
  total_amount: number;
  paid_amount: number;
  due_amount: number;
  bill_status: string;
  payment_status: "paid" | "partial" | "unpaid" | "no_bill";
  last_payment_at: string | null;
  collector_name: string;
  collector_id: string;
  olt_name: string;
  olt_id: string;
  pon_port_label: string;
  pon_port_id: string;
  onu_id: string;
  onu_mac: string;
  onu_status: string;
}

export interface CollectionSummary {
  active_clients: number;
  collected_clients: number;
  uncollected_clients: number;
  total_bill: number;
  collection_amount: number;
  total_due: number;
  collection_rate: number;
}

export interface CollectorSummaryRow {
  collector_id: string;
  collector_name: string;
  client_count: number;
  collection: number;
}

export interface PackageSummaryRow {
  package_id: string;
  package_name: string;
  client_count: number;
  collection: number;
}

export interface DailyChartPoint {
  date: string;
  label: string;
  collection: number;
}

export interface CollectionReportData {
  summary: CollectionSummary;
  collector_summary: CollectorSummaryRow[];
  package_summary: PackageSummaryRow[];
  daily_chart: DailyChartPoint[];
  data: CollectionRow[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
  billing_month: number;
  billing_year: number;
}
