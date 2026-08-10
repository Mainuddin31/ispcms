"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Router,
  Wifi,
  WifiOff,
  UserCheck,
  UserX,
  Activity,
  Clock,
  RefreshCw,
  Globe,
  Archive,
  Package,
  CreditCard,
  FileText,
  AlertTriangle,
  Banknote,
  TrendingUp,
  CircleDollarSign,
  CheckCircle,
  CircleDot,
  Receipt,
  Wallet,
  ShieldCheck,
  ArrowDownCircle,
  ArrowUpCircle,
  BarChart2,
  RefreshCcw,
  Tag,
} from "lucide-react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
} from "recharts";
import { dashboardApi } from "@/lib/api";
import { DashboardStats, ActivityLog } from "@/types";
import { useAuth } from "@/contexts/AuthContext";
import { Topbar } from "@/components/layout/Topbar";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { formatRelative } from "@/lib/utils";

// ── Helpers ──────────────────────────────────────────────────────────────────

function fmt(n: number) {
  return `৳${n.toLocaleString("en-BD", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function StatCard({
  title,
  value,
  icon: Icon,
  color,
  sub,
}: {
  title: string;
  value: number | string;
  icon: React.ElementType;
  color: string;
  sub?: string;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div className="min-w-0">
            <p className="text-xs font-medium text-muted-foreground truncate">{title}</p>
            <p className="text-2xl font-bold mt-0.5 truncate">{value}</p>
            {sub && <p className="text-xs text-muted-foreground mt-0.5 truncate">{sub}</p>}
          </div>
          <div className={`w-11 h-11 rounded-xl flex-shrink-0 flex items-center justify-center ml-3 ${color}`}>
            <Icon className="w-5 h-5 text-white" />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function StatusBadge({ status }: { status: string }) {
  const cls: Record<string, string> = {
    success: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
    failed: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
    running: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
    completed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
    partial: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400",
  };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls[status] ?? "bg-gray-100 text-gray-800"}`}>
      {status}
    </span>
  );
}

const PIE_COLORS = ["#6366f1", "#22c55e", "#f59e0b", "#ef4444", "#06b6d4", "#8b5cf6", "#ec4899", "#14b8a6", "#f97316", "#84cc16"];

// ── Activity icon/colour per type ─────────────────────────────────────────────

function activityMeta(a: ActivityLog) {
  const type = a.activity_type || a.action || "";
  const mod = a.module || "";
  if (type === "payment_received") return { Icon: Banknote, color: "text-green-500", bg: "bg-green-100 dark:bg-green-900/30" };
  if (type === "bills_generated") return { Icon: FileText, color: "text-blue-500", bg: "bg-blue-100 dark:bg-blue-900/30" };
  if (type === "expense_created") return { Icon: Receipt, color: "text-orange-500", bg: "bg-orange-100 dark:bg-orange-900/30" };
  if (type === "expense_updated") return { Icon: Receipt, color: "text-amber-500", bg: "bg-amber-100 dark:bg-amber-900/30" };
  if (type === "expense_deleted") return { Icon: Receipt, color: "text-red-500", bg: "bg-red-100 dark:bg-red-900/30" };
  if (type === "sync_completed") return { Icon: RefreshCcw, color: "text-teal-500", bg: "bg-teal-100 dark:bg-teal-900/30" };
  if (mod === "billing") return { Icon: CreditCard, color: "text-blue-500", bg: "bg-blue-100 dark:bg-blue-900/30" };
  if (mod === "expenses") return { Icon: Receipt, color: "text-orange-500", bg: "bg-orange-100 dark:bg-orange-900/30" };
  if (mod === "sync") return { Icon: RefreshCcw, color: "text-teal-500", bg: "bg-teal-100 dark:bg-teal-900/30" };
  return { Icon: Activity, color: "text-slate-500", bg: "bg-slate-100 dark:bg-slate-800" };
}

// ── Main page ─────────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const [activityPeriod, setActivityPeriod] = useState("30days");
  const [activityModule, setActivityModule] = useState("");
  const { hasPermission } = useAuth();

  // Permission flags — drive section visibility
  const canBilling  = hasPermission("billing",  "view");
  const canExpenses = hasPermission("expenses", "view");
  const canRouters  = hasPermission("routers",  "view");
  const canAccounts = hasPermission("accounts", "view");
  const canReports  = hasPermission("reports",  "view");

  const { data, isLoading, refetch, isFetching } = useQuery<{ data: { data: DashboardStats } }>({
    queryKey: ["dashboard-stats"],
    queryFn: dashboardApi.stats,
    refetchInterval: 60_000,
  });

  const { data: activitiesData, isLoading: activitiesLoading } = useQuery({
    queryKey: ["dashboard-activities", activityPeriod, activityModule],
    queryFn: () => dashboardApi.activities({ period: activityPeriod, module: activityModule || undefined, limit: 50 }),
    refetchInterval: 30_000,
  });

  const stats = data?.data?.data;
  const activities: ActivityLog[] = activitiesData?.data?.data ?? [];

  return (
    <div>
      <Topbar
        title="Dashboard"
        subtitle="Business overview & financial summary"
        breadcrumbs={[{ label: "Home" }, { label: "Dashboard" }]}
      />

      <div className="p-6 space-y-6">
        {/* Toolbar */}
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            {stats?.last_sync_time
              ? `Last sync: ${formatRelative(stats.last_sync_time)}`
              : "No sync yet"}
          </p>
          <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
            <RefreshCw className={`w-3.5 h-3.5 mr-1.5 ${isFetching ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>

        {/* ── Collections ───────────────────────────────────────────────────── */}
        {canBilling && (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Collections</p>
              {canReports && (
                <a href="/collection-report" className="text-xs text-blue-500 hover:text-blue-400 font-medium">View Details →</a>
              )}
            </div>
            {isLoading ? (
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Card key={i}><CardContent className="p-5"><Skeleton className="h-4 w-24 mb-2" /><Skeleton className="h-7 w-20" /></CardContent></Card>
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard title="Today's Collection" value={fmt(stats?.today_collection ?? 0)} icon={ArrowDownCircle} color="bg-emerald-500" />
                <StatCard title="This Month" value={fmt(stats?.monthly_collection ?? 0)} icon={TrendingUp} color="bg-green-600" />
                <StatCard title="Last Month" value={fmt(stats?.last_month_collection ?? 0)} icon={Banknote} color="bg-teal-600" />
                <StatCard title="Outstanding Due" value={fmt(stats?.total_outstanding_due ?? 0)} icon={AlertTriangle} color={(stats?.total_outstanding_due ?? 0) > 0 ? "bg-red-500" : "bg-slate-400"} />
              </div>
            )}
          </div>
        )}

        {/* ── Expenses ──────────────────────────────────────────────────────── */}
        {canExpenses && (
          <div className="space-y-2">
            <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Expenses</p>
            {isLoading ? (
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                {Array.from({ length: 4 }).map((_, i) => (
                  <Card key={i}><CardContent className="p-5"><Skeleton className="h-4 w-24 mb-2" /><Skeleton className="h-7 w-20" /></CardContent></Card>
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                <StatCard title="Today's Expense" value={fmt(stats?.today_expense ?? 0)} icon={ArrowUpCircle} color="bg-orange-500" />
                <StatCard title="This Month" value={fmt(stats?.monthly_expense ?? 0)} icon={Receipt} color="bg-amber-500" />
                <StatCard title="Last Month" value={fmt(stats?.last_month_expense ?? 0)} icon={Wallet} color="bg-red-400" />
                <StatCard
                  title="Cash in Hand"
                  value={fmt((stats?.monthly_collection ?? 0) - (stats?.monthly_expense ?? 0))}
                  icon={ShieldCheck}
                  color={((stats?.monthly_collection ?? 0) - (stats?.monthly_expense ?? 0)) >= 0 ? "bg-indigo-600" : "bg-red-600"}
                  sub="This month collection − expense"
                />
              </div>
            )}
          </div>
        )}

        {/* Cash in Hand for billing officers who can't see Expenses section */}
        {canBilling && !canExpenses && !isLoading && (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              title="Cash in Hand"
              value={fmt((stats?.monthly_collection ?? 0) - (stats?.monthly_expense ?? 0))}
              icon={ShieldCheck}
              color={((stats?.monthly_collection ?? 0) - (stats?.monthly_expense ?? 0)) >= 0 ? "bg-indigo-600" : "bg-red-600"}
              sub="This month collection − expense"
            />
          </div>
        )}

        {/* ── Charts ─────────────────────────────────────────────────────────── */}
        {(canBilling || canExpenses) && !isLoading && (stats?.monthly_chart?.length ?? 0) > 0 && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Collection vs Expense line chart */}
            <Card className="lg:col-span-2">
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-semibold flex items-center gap-2">
                  <BarChart2 className="w-4 h-4" />
                  Collection vs Expense — Last 12 Months
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={260}>
                  <LineChart data={stats!.monthly_chart} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="currentColor" strokeOpacity={0.1} />
                    <XAxis dataKey="label" tick={{ fontSize: 11 }} interval={1} />
                    <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => `৳${(v / 1000).toFixed(0)}k`} />
                    <Tooltip formatter={(v: number) => fmt(v)} />
                    <Legend />
                    <Line type="monotone" dataKey="collection" name="Collection" stroke="#22c55e" strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="expense" name="Expense" stroke="#ef4444" strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="cash_in_hand" name="Cash in Hand" stroke="#6366f1" strokeWidth={1.5} dot={false} strokeDasharray="4 2" />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            {/* Expense category pie */}
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-semibold flex items-center gap-2">
                  <Tag className="w-4 h-4" />
                  Expense by Category
                  <span className="text-xs text-muted-foreground font-normal">(this month)</span>
                </CardTitle>
              </CardHeader>
              <CardContent>
                {(stats?.expense_category_pie?.length ?? 0) === 0 ? (
                  <p className="text-sm text-muted-foreground text-center py-16">No expenses this month</p>
                ) : (
                  <>
                    <ResponsiveContainer width="100%" height={180}>
                      <PieChart>
                        <Pie data={stats!.expense_category_pie} dataKey="total" nameKey="category_name" cx="50%" cy="50%" outerRadius={70} label={false}>
                          {stats!.expense_category_pie.map((_, i) => (
                            <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                          ))}
                        </Pie>
                        <Tooltip formatter={(v: number) => fmt(v)} />
                      </PieChart>
                    </ResponsiveContainer>
                    <div className="space-y-1.5 mt-2">
                      {stats!.expense_category_pie.slice(0, 6).map((c, i) => (
                        <div key={c.category_id} className="flex items-center justify-between text-xs">
                          <div className="flex items-center gap-1.5 min-w-0">
                            <span className="w-2.5 h-2.5 rounded-full flex-shrink-0" style={{ background: PIE_COLORS[i % PIE_COLORS.length] }} />
                            <span className="truncate text-muted-foreground">{c.category_name}</span>
                          </div>
                          <span className="font-medium ml-2 flex-shrink-0">{fmt(c.total)}</span>
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          </div>
        )}

        {/* ── Monthly Collection Bar Chart ───────────────────────────────────── */}
        {canBilling && !isLoading && (stats?.monthly_chart?.length ?? 0) > 0 && (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-semibold">Monthly Collection</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={180}>
                <BarChart data={stats!.monthly_chart} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="currentColor" strokeOpacity={0.1} />
                  <XAxis dataKey="label" tick={{ fontSize: 11 }} interval={1} />
                  <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => `৳${(v / 1000).toFixed(0)}k`} />
                  <Tooltip formatter={(v: number) => fmt(v)} />
                  <Bar dataKey="collection" name="Collection" fill="#22c55e" radius={[3, 3, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        )}

        {/* ── Billing Stats ──────────────────────────────────────────────────── */}
        {canBilling && ((stats?.total_packages ?? 0) > 0 || (stats?.active_subscriptions ?? 0) > 0) && (
          <div className="space-y-2">
            <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Billing</p>
            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-4">
              <StatCard title="Active Packages" value={stats?.active_packages ?? 0} icon={Package} color="bg-indigo-500" sub={`${stats?.total_packages ?? 0} total`} />
              <StatCard title="Subscriptions" value={stats?.active_subscriptions ?? 0} icon={CreditCard} color="bg-purple-500" />
              <StatCard title="Bills This Month" value={stats?.bills_this_month ?? 0} icon={FileText} color="bg-sky-500" sub={(stats?.bills_pending_generate ?? 0) > 0 ? `${stats?.bills_pending_generate} pending` : undefined} />
              <StatCard title="Bills Paid" value={stats?.bills_paid ?? 0} icon={CheckCircle} color="bg-green-600" />
              <StatCard title="Bills Unpaid" value={stats?.bills_billing_pending ?? 0} icon={CircleDot} color={(stats?.bills_billing_pending ?? 0) > 0 ? "bg-orange-500" : "bg-slate-400"} />
              <StatCard title="Unmapped Profiles" value={stats?.unmapped_profiles ?? 0} icon={AlertTriangle} color={(stats?.unmapped_profiles ?? 0) > 0 ? "bg-amber-500" : "bg-slate-400"} />
            </div>
          </div>
        )}

        {/* ── Network Stats & Internet Accounts ─────────────────────────────── */}
        {(canRouters || canAccounts) && (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {canRouters && (
              <div className="space-y-2">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Network</p>
                <div className="grid grid-cols-2 gap-3">
                  <StatCard title="Total Routers" value={stats?.total_routers ?? 0} icon={Router} color="bg-blue-500" />
                  <StatCard title="Online Routers" value={stats?.online_routers ?? 0} icon={Wifi} color="bg-emerald-500" />
                  <StatCard title="Offline Routers" value={stats?.offline_routers ?? 0} icon={WifiOff} color="bg-red-500" />
                  <StatCard title="Active Sessions" value={stats?.active_sessions ?? 0} icon={Activity} color="bg-cyan-500" />
                </div>
              </div>
            )}
            {canAccounts && (
              <div className="space-y-2">
                <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Internet Accounts</p>
                <div className="grid grid-cols-2 gap-3">
                  <StatCard title="Total Accounts" value={stats?.total_accounts ?? 0} icon={Globe} color="bg-violet-500" />
                  <StatCard title="Online" value={stats?.online_accounts ?? 0} icon={UserCheck} color="bg-teal-500" />
                  <StatCard title="Offline" value={stats?.offline_accounts ?? 0} icon={WifiOff} color="bg-slate-400" />
                  <StatCard title="Disabled" value={stats?.disabled_accounts ?? 0} icon={UserX} color="bg-orange-500" />
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Activity Timeline + Recent Syncs ──────────────────────────────── */}
        <div className={`grid grid-cols-1 ${canRouters ? "lg:grid-cols-2" : ""} gap-6`}>
          {/* Activity Timeline */}
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between flex-wrap gap-2">
                <CardTitle className="text-sm font-semibold">System Activity</CardTitle>
                <div className="flex items-center gap-2 flex-wrap">
                  {/* Period filter */}
                  <div className="flex rounded-md border overflow-hidden">
                    {[["today", "Today"], ["7days", "7d"], ["30days", "30d"]].map(([val, label]) => (
                      <button
                        key={val}
                        onClick={() => setActivityPeriod(val)}
                        className={`px-2 py-1 text-xs transition-colors ${activityPeriod === val ? "bg-primary text-primary-foreground" : "bg-background hover:bg-muted"}`}
                      >
                        {label}
                      </button>
                    ))}
                  </div>
                  {/* Module filter */}
                  <select
                    value={activityModule}
                    onChange={(e) => setActivityModule(e.target.value)}
                    className="text-xs border rounded-md px-2 py-1 bg-background"
                  >
                    <option value="">All modules</option>
                    <option value="billing">Billing</option>
                    <option value="expenses">Expenses</option>
                    <option value="sync">Sync</option>
                  </select>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              {activitiesLoading ? (
                <div className="space-y-3">
                  {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-14 w-full" />)}
                </div>
              ) : !activities.length ? (
                <p className="text-sm text-muted-foreground text-center py-12">No activity in this period</p>
              ) : (
                <div className="relative">
                  {/* Timeline line */}
                  <div className="absolute left-5 top-0 bottom-0 w-px bg-border" />
                  <div className="space-y-0">
                    {activities.map((a, idx) => {
                      const { Icon, color, bg } = activityMeta(a);
                      const title = a.title || `${a.action} ${a.module}`;
                      const desc = a.description || a.details || "";
                      return (
                        <div key={a.id} className={`flex gap-3 pb-4 ${idx === activities.length - 1 ? "" : ""}`}>
                          <div className={`w-10 h-10 rounded-full flex-shrink-0 flex items-center justify-center ${bg} z-10`}>
                            <Icon className={`w-4 h-4 ${color}`} />
                          </div>
                          <div className="flex-1 min-w-0 pt-1">
                            <div className="flex items-start justify-between gap-2">
                              <div className="min-w-0">
                                <p className="text-sm font-medium truncate">{title}</p>
                                {desc && <p className="text-xs text-muted-foreground truncate">{desc}</p>}
                                {a.user && <p className="text-xs text-muted-foreground">by {a.user.full_name}</p>}
                              </div>
                              <p className="text-xs text-muted-foreground flex-shrink-0 pt-0.5">{formatRelative(a.created_at)}</p>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Recent Syncs — only for users with router access */}
          {canRouters && (
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-semibold">Recent Synchronizations</CardTitle>
              </CardHeader>
              <CardContent>
                {isLoading ? (
                  <div className="space-y-3">
                    {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-14 w-full" />)}
                  </div>
                ) : !stats?.recent_sync_logs?.length ? (
                  <p className="text-sm text-muted-foreground text-center py-12">No sync records</p>
                ) : (
                  <div className="space-y-3">
                    {stats.recent_sync_logs.map((log) => (
                      <div key={log.id} className="flex items-center justify-between p-3 rounded-lg border bg-muted/30">
                        <div className="min-w-0">
                          <p className="text-sm font-medium truncate">{log.router?.name ?? "Unknown router"}</p>
                          <p className="text-xs text-muted-foreground">
                            {log.secrets_total} secrets · {log.new_accounts ?? 0} new · {log.duration_ms}ms
                          </p>
                        </div>
                        <div className="text-right flex-shrink-0 ml-3">
                          <StatusBadge status={log.status} />
                          <p className="text-xs text-muted-foreground mt-1">{formatRelative(log.started_at)}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>

        {/* Last sync footer */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Clock className="w-3.5 h-3.5" />
          {stats?.last_sync_time ? `Last sync ${formatRelative(stats.last_sync_time)}` : "No sync performed yet"}
        </div>
      </div>
    </div>
  );
}
