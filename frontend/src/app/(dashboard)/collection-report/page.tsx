"use client";

import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Users, Wallet, AlertTriangle, TrendingUp, CheckCircle,
  XCircle, BarChart2, Search, X, ChevronRight, ChevronDown,
  Download, Filter,
} from "lucide-react";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer,
} from "recharts";
import { reportsApi, routersApi } from "@/lib/api";
import { useAuth } from "@/contexts/AuthContext";
import { CollectionRow, CollectionReportData } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet, SheetContent, SheetHeader, SheetTitle,
} from "@/components/ui/sheet";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";

// ── helpers ──────────────────────────────────────────────────────────────────

function fmt(n: number) {
  return `৳${n.toLocaleString("en-BD", { minimumFractionDigits: 0, maximumFractionDigits: 0 })}`;
}

function fmtDate(d: string | null) {
  if (!d) return "—";
  return new Date(d).toLocaleDateString("en-BD", { day: "numeric", month: "short" });
}

const MONTHS = [
  "January","February","March","April","May","June",
  "July","August","September","October","November","December",
];

function statusBadge(status: string) {
  const map: Record<string, string> = {
    paid:    "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
    partial: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
    unpaid:  "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
    no_bill: "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",
  };
  const label: Record<string, string> = {
    paid: "Paid", partial: "Partial", unpaid: "Unpaid", no_bill: "No Bill",
  };
  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${map[status] ?? ""}`}>
      {label[status] ?? status}
    </span>
  );
}

function SummaryCard({
  title, value, sub, icon: Icon, color,
}: { title: string; value: string | number; sub?: string; icon: React.ElementType; color: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs text-muted-foreground font-medium">{title}</p>
            <p className="text-xl font-bold mt-0.5">{value}</p>
            {sub && <p className="text-xs text-muted-foreground mt-0.5">{sub}</p>}
          </div>
          <div className={`w-10 h-10 rounded-xl flex items-center justify-center ${color}`}>
            <Icon className="w-5 h-5 text-white" />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ── Customer Detail Drawer ───────────────────────────────────────────────────

function CustomerDrawer({ row, open, onClose }: { row: CollectionRow | null; open: boolean; onClose: () => void }) {
  if (!row) return null;
  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="w-[420px] sm:max-w-[480px] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{row.customer_name}</SheetTitle>
        </SheetHeader>
        <div className="mt-4 space-y-5 text-sm">
          {/* Internet Account */}
          <Section title="Internet Account">
            <Row label="PPPoE Username" value={row.username} mono />
            <Row label="Package" value={row.package_name || "—"} />
            <Row label="Monthly Charge" value={row.monthly_charge ? fmt(row.monthly_charge) : "—"} />
            <Row label="MikroTik Router" value={row.router_name || "—"} />
          </Section>

          {/* Network */}
          {row.olt_name && (
            <Section title="Network (ONU)">
              <Row label="OLT" value={row.olt_name} />
              <Row label="PON Port" value={row.pon_port_label || "—"} />
              <Row label="ONU ID" value={row.onu_id || "—"} mono />
              <Row label="ONU MAC" value={row.onu_mac || "—"} mono />
              <Row label="ONU Status" value={
                <span className={row.onu_status === "online" ? "text-green-600" : "text-red-500"}>
                  {row.onu_status || "—"}
                </span>
              } />
            </Section>
          )}

          {/* Current Bill */}
          <Section title="Current Bill">
            {row.bill_id ? (
              <>
                <Row label="Bill Number" value={row.bill_number} mono />
                <Row label="Bill Amount" value={fmt(row.total_amount)} />
                <Row label="Paid Amount" value={fmt(row.paid_amount)} />
                <Row label="Due Amount" value={fmt(row.due_amount)} />
                <Row label="Status" value={statusBadge(row.payment_status)} />
              </>
            ) : (
              <p className="text-muted-foreground text-xs">No bill generated for this period.</p>
            )}
          </Section>

          {/* Last Payment */}
          {row.last_payment_at && (
            <Section title="Last Payment">
              <Row label="Date" value={fmtDate(row.last_payment_at)} />
              <Row label="Collected By" value={row.collector_name || "—"} />
            </Section>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">{title}</p>
      <div className="space-y-1.5 bg-muted/30 rounded-lg p-3">{children}</div>
    </div>
  );
}

function Row({ label, value, mono }: { label: string; value: React.ReactNode; mono?: boolean }) {
  return (
    <div className="flex justify-between items-center gap-2">
      <span className="text-muted-foreground shrink-0">{label}</span>
      <span className={`font-medium text-right ${mono ? "font-mono text-xs" : ""}`}>{value}</span>
    </div>
  );
}

// ── Main Page ─────────────────────────────────────────────────────────────────

export default function CollectionReportPage() {
  const { user, hasRole } = useAuth();
  // Billing officers only see their own collections; admins see everyone's
  const isBillingOfficer = !hasRole("super_admin") && !hasRole("admin");
  const ownCollectorId = isBillingOfficer ? user?.id : undefined;

  const now = new Date();
  // Default to current (running) month
  const [billingMonth, setBillingMonth] = useState(now.getMonth() + 1); // getMonth() is 0-indexed
  const [billingYear, setBillingYear] = useState(now.getFullYear());
  const [paymentStatus, setPaymentStatus] = useState("all");
  const [search, setSearch] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [page, setPage] = useState(1);
  const [selectedRow, setSelectedRow] = useState<CollectionRow | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [routerId, setRouterId] = useState("");
  // Admin: click a staff card to filter table to that collector
  const [selectedCollectorId, setSelectedCollectorId] = useState<string | null>(null);
  const [selectedCollectorName, setSelectedCollectorName] = useState<string | null>(null);

  const params = {
    billing_month: billingMonth,
    billing_year: billingYear,
    payment_status: paymentStatus === "all" ? undefined : paymentStatus,
    search: search || undefined,
    router_id: routerId || undefined,
    // Billing officers see only their own; admins filter by selected card (or all)
    collector_id: ownCollectorId ?? selectedCollectorId ?? undefined,
    page,
    page_size: 50,
  };

  const { data: raw, isLoading, error } = useQuery<{ data: { data: CollectionReportData } }>({
    queryKey: ["collection-report", params],
    queryFn: () => reportsApi.activeUserCollection(params),
    retry: false,
  });

  const { data: routersRaw } = useQuery<{ data: { data: any[] } }>({
    queryKey: ["routers-list"],
    queryFn: () => routersApi.list(),
    enabled: showFilters,
  });

  const report = raw?.data?.data;
  const summary = report?.summary;
  const rows = report?.data ?? [];

  const years = useMemo(() => {
    const y = [];
    for (let i = now.getFullYear(); i >= now.getFullYear() - 3; i--) y.push(i);
    return y;
  }, []);

  function handleSearch() {
    setSearch(searchInput);
    setPage(1);
  }

  function openRow(row: CollectionRow) {
    setSelectedRow(row);
    setDrawerOpen(true);
  }

  // Export as CSV
  function exportCSV() {
    if (!rows.length) return;
    const headers = ["Username","Package","Bill Amount","Paid","Due","Status","Last Payment","Collected By"];
    const csvRows = rows.map(r => [
      r.username, r.package_name,
      r.total_amount, r.paid_amount, r.due_amount,
      r.payment_status, r.last_payment_at ? fmtDate(r.last_payment_at) : "",
      r.collector_name,
    ].map(v => `"${String(v).replace(/"/g,'""')}"`).join(","));
    const blob = new Blob([[headers.join(","), ...csvRows].join("\n")], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `collection-${billingYear}-${billingMonth}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  const periodLabel = `${MONTHS[billingMonth - 1]} ${billingYear}`;

  return (
    <div>
      <Topbar
        title="Active User Collection"
        breadcrumbs={[{ label: "Home" }, { label: "Dashboard", href: "/dashboard" }, { label: "Collection Report" }]}
      />

      <div className="p-6 space-y-5">

        {/* Period selector */}
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Select value={String(billingMonth)} onValueChange={v => { setBillingMonth(Number(v)); setPage(1); }}>
              <SelectTrigger className="w-36 h-8 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {MONTHS.map((m, i) => (
                  <SelectItem key={i} value={String(i + 1)}>{m}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={String(billingYear)} onValueChange={v => { setBillingYear(Number(v)); setPage(1); }}>
              <SelectTrigger className="w-24 h-8 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {years.map(y => <SelectItem key={y} value={String(y)}>{y}</SelectItem>)}
              </SelectContent>
            </Select>
            <p className="text-sm text-muted-foreground">Period: <strong>{periodLabel}</strong></p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => setShowFilters(!showFilters)}>
              <Filter className="w-3.5 h-3.5 mr-1.5" />Filters
            </Button>
            <Button variant="outline" size="sm" onClick={exportCSV}>
              <Download className="w-3.5 h-3.5 mr-1.5" />Export CSV
            </Button>
          </div>
        </div>

        {/* Filters row */}
        {showFilters && (
          <Card>
            <CardContent className="p-4">
              <div className="flex flex-wrap gap-3">
                <Select value={paymentStatus} onValueChange={v => { setPaymentStatus(v); setPage(1); }}>
                  <SelectTrigger className="w-36 h-8 text-sm">
                    <SelectValue placeholder="Payment Status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Status</SelectItem>
                    <SelectItem value="paid">Paid</SelectItem>
                    <SelectItem value="partial">Partial</SelectItem>
                    <SelectItem value="unpaid">Unpaid</SelectItem>
                    <SelectItem value="no_bill">No Bill</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>
        )}

        {/* API error */}
        {error && (
          <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-900/20 dark:border-red-800 p-4 text-sm text-red-700 dark:text-red-400">
            <strong>Error loading report:</strong>{" "}
            {(error as any)?.response?.data?.error ?? (error as any)?.message ?? "Unknown error"}
            {(error as any)?.response?.status === 403 && (
              <span className="block mt-1 text-xs">Your role does not have <code>reports.view</code> permission. Ask an admin to grant it.</span>
            )}
          </div>
        )}

        {/* Summary cards */}
        {isLoading ? (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {Array.from({ length: 7 }).map((_, i) => (
              <Card key={i}><CardContent className="p-4"><Skeleton className="h-4 w-24 mb-2" /><Skeleton className="h-6 w-20" /></CardContent></Card>
            ))}
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-3">
            <SummaryCard title="Total Active" value={summary?.active_clients ?? 0} icon={Users} color="bg-blue-500" />
            <SummaryCard title="Collected" value={summary?.collected_clients ?? 0} icon={CheckCircle} color="bg-green-500" />
            <SummaryCard title="Uncollected" value={summary?.uncollected_clients ?? 0} icon={XCircle} color="bg-red-500" />
            <SummaryCard title="Collection" value={fmt(summary?.collection_amount ?? 0)} icon={Wallet} color="bg-emerald-600" />
            <SummaryCard title="Total Bill" value={fmt(summary?.total_bill ?? 0)} icon={BarChart2} color="bg-slate-500" />
            <SummaryCard title="Total Due" value={fmt(summary?.total_due ?? 0)} icon={AlertTriangle} color={(summary?.total_due ?? 0) > 0 ? "bg-orange-500" : "bg-slate-400"} />
            <SummaryCard title="Rate" value={`${(summary?.collection_rate ?? 0).toFixed(1)}%`} icon={TrendingUp} color="bg-indigo-500" />
          </div>
        )}

        {/* ── Staff Collection Cards ─────────────────────────────────────── */}
        {!isLoading && report && (
          <>
            <div>
              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-3">
                {isBillingOfficer ? `My Collection — ${periodLabel}` : `Staff Collection — ${periodLabel}`}
              </p>
              {!isBillingOfficer && (
                <p className="text-xs text-muted-foreground mb-3">Click a card to see that staff member's collected bill list below.</p>
              )}
              {report.collector_summary.length === 0 ? (
                <Card><CardContent className="p-6 text-center text-sm text-muted-foreground">No collection recorded for this period.</CardContent></Card>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                  {report.collector_summary.map((c, idx) => {
                    const totalCollection = report.summary?.collection_amount ?? 0;
                    const share = totalCollection > 0 ? (c.collection / totalCollection) * 100 : 0;
                    const colors = [
                      "border-l-blue-500", "border-l-emerald-500", "border-l-violet-500",
                      "border-l-amber-500", "border-l-rose-500", "border-l-cyan-500",
                    ];
                    const bar = [
                      "bg-blue-500", "bg-emerald-500", "bg-violet-500",
                      "bg-amber-500", "bg-rose-500", "bg-cyan-500",
                    ];
                    const color = colors[idx % colors.length];
                    const barColor = bar[idx % bar.length];
                    const isSelected = !isBillingOfficer && selectedCollectorId === c.collector_id;
                    return (
                      <Card
                        key={c.collector_id}
                        className={`border-l-4 ${color} transition-all ${
                          !isBillingOfficer
                            ? "cursor-pointer hover:shadow-md " + (isSelected ? "ring-2 ring-blue-500 shadow-md" : "")
                            : ""
                        }`}
                        onClick={() => {
                          if (isBillingOfficer) return;
                          if (selectedCollectorId === c.collector_id) {
                            setSelectedCollectorId(null);
                            setSelectedCollectorName(null);
                          } else {
                            setSelectedCollectorId(c.collector_id);
                            setSelectedCollectorName(c.collector_name);
                            setPage(1);
                          }
                        }}
                      >
                        <CardContent className="p-5">
                          {/* Avatar + name */}
                          <div className="flex items-center gap-3 mb-4">
                            <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center shrink-0">
                              <span className="text-base font-bold text-muted-foreground">
                                {c.collector_name.charAt(0).toUpperCase()}
                              </span>
                            </div>
                            <div className="min-w-0">
                              <p className="font-semibold text-sm truncate">{c.collector_name}</p>
                              <p className="text-xs text-muted-foreground">Billing Officer</p>
                            </div>
                          </div>

                          {/* Stats */}
                          <div className="grid grid-cols-2 gap-3 mb-4">
                            <div className="bg-muted/40 rounded-lg p-3">
                              <p className="text-xs text-muted-foreground mb-0.5">Total Collection</p>
                              <p className="text-lg font-bold text-green-600">{fmt(c.collection)}</p>
                            </div>
                            <div className="bg-muted/40 rounded-lg p-3">
                              <p className="text-xs text-muted-foreground mb-0.5">Paid Clients</p>
                              <p className="text-lg font-bold">{c.client_count}</p>
                            </div>
                          </div>

                          {/* Share bar */}
                          <div>
                            <div className="flex justify-between text-xs text-muted-foreground mb-1">
                              <span>Share of total</span>
                              <span className="font-medium">{share.toFixed(1)}%</span>
                            </div>
                            <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                              <div
                                className={`h-full ${barColor} rounded-full transition-all duration-500`}
                                style={{ width: `${Math.min(share, 100)}%` }}
                              />
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    );
                  })}
                </div>
              )}
            </div>

          {/* ── Package + Daily chart ──────────────────────────────────────── */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {/* By Package */}
            <Card>
              <CardHeader className="py-3 px-4">
                <CardTitle className="text-sm">By Package</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <table className="w-full text-xs">
                  <thead><tr className="border-b">
                    <th className="text-left px-4 py-2 text-muted-foreground font-medium">Package</th>
                    <th className="text-right px-4 py-2 text-muted-foreground font-medium">Clients</th>
                    <th className="text-right px-4 py-2 text-muted-foreground font-medium">Amount</th>
                  </tr></thead>
                  <tbody>
                    {report.package_summary.length === 0 ? (
                      <tr><td colSpan={3} className="text-center py-4 text-muted-foreground">No data</td></tr>
                    ) : report.package_summary.map((p) => (
                      <tr key={p.package_id} className="border-b last:border-0">
                        <td className="px-4 py-2 font-medium">{p.package_name}</td>
                        <td className="px-4 py-2 text-right">{p.client_count}</td>
                        <td className="px-4 py-2 text-right text-green-600 font-medium">{fmt(p.collection)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </CardContent>
            </Card>

            {/* Daily chart */}
            <Card>
              <CardHeader className="py-3 px-4">
                <CardTitle className="text-sm">Daily Collection</CardTitle>
              </CardHeader>
              <CardContent className="p-2 h-[180px]">
                {report.daily_chart.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-muted-foreground text-xs">No payments this period</div>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={report.daily_chart} margin={{ top: 4, right: 4, left: -20, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="rgba(100,100,100,0.15)" />
                      <XAxis dataKey="label" tick={{ fontSize: 9 }} />
                      <YAxis tick={{ fontSize: 9 }} />
                      <Tooltip formatter={(v: number) => fmt(v)} />
                      <Bar dataKey="collection" fill="#22c55e" radius={[2, 2, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>
          </div>
          </>
        )}

        {/* Active collector filter banner */}
        {!isBillingOfficer && selectedCollectorName && (
          <div className="flex items-center gap-2 text-sm bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg px-4 py-2.5">
            <Users className="w-4 h-4 text-blue-500 shrink-0" />
            <span className="text-blue-700 dark:text-blue-300 font-medium">
              Showing bills collected by <strong>{selectedCollectorName}</strong>
            </span>
            <button
              onClick={() => { setSelectedCollectorId(null); setSelectedCollectorName(null); setPage(1); }}
              className="ml-auto text-blue-500 hover:text-blue-700 text-xs font-medium"
            >
              ✕ Clear filter
            </button>
          </div>
        )}

        {/* Search bar */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
            <Input
              className="pl-8 h-8 text-sm"
              placeholder="Search username, name, bill number…"
              value={searchInput}
              onChange={e => setSearchInput(e.target.value)}
              onKeyDown={e => e.key === "Enter" && handleSearch()}
            />
            {searchInput && (
              <button
                onClick={() => { setSearchInput(""); setSearch(""); setPage(1); }}
                className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
          <Button size="sm" variant="secondary" onClick={handleSearch}>Search</Button>
        </div>

        {/* Table */}
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="border-b">
                <tr>
                  {["Username","Package","Bill","Paid","Due","Status","Last Payment","Collected By"].map(h => (
                    <th key={h} className="text-left px-4 py-2.5 text-xs font-medium text-muted-foreground whitespace-nowrap">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 8 }).map((_, i) => (
                    <tr key={i} className="border-b">
                      {Array.from({ length: 8 }).map((_, j) => (
                        <td key={j} className="px-4 py-2.5"><Skeleton className="h-4 w-16" /></td>
                      ))}
                    </tr>
                  ))
                ) : rows.length === 0 ? (
                  <tr><td colSpan={8} className="text-center py-10 text-muted-foreground">No records found</td></tr>
                ) : rows.map((row) => (
                  <tr
                    key={row.account_id}
                    className="border-b hover:bg-muted/50 cursor-pointer transition-colors"
                    onClick={() => openRow(row)}
                  >
                    <td className="px-4 py-2.5 font-mono text-xs">{row.username}</td>
                    <td className="px-4 py-2.5 text-muted-foreground">{row.package_name || "—"}</td>
                    <td className="px-4 py-2.5 tabular-nums">{row.total_amount ? fmt(row.total_amount) : "—"}</td>
                    <td className="px-4 py-2.5 tabular-nums text-green-600">{row.paid_amount ? fmt(row.paid_amount) : "—"}</td>
                    <td className="px-4 py-2.5 tabular-nums text-red-500">{row.due_amount > 0 ? fmt(row.due_amount) : "—"}</td>
                    <td className="px-4 py-2.5">{statusBadge(row.payment_status)}</td>
                    <td className="px-4 py-2.5 text-muted-foreground whitespace-nowrap">{fmtDate(row.last_payment_at)}</td>
                    <td className="px-4 py-2.5 font-medium">{row.collector_name || <span className="text-muted-foreground">—</span>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {report && report.total_pages > 1 && (
            <div className="flex items-center justify-between px-4 py-3 border-t text-sm">
              <span className="text-muted-foreground text-xs">
                {(page - 1) * 50 + 1}–{Math.min(page * 50, report.total)} of {report.total}
              </span>
              <div className="flex gap-2">
                <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>
                  Previous
                </Button>
                <Button size="sm" variant="outline" disabled={page >= report.total_pages} onClick={() => setPage(p => p + 1)}>
                  Next
                </Button>
              </div>
            </div>
          )}
        </Card>
      </div>

      <CustomerDrawer row={selectedRow} open={drawerOpen} onClose={() => setDrawerOpen(false)} />
    </div>
  );
}
