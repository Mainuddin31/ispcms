"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Search,
  RefreshCw,
  Loader2,
  Wifi,
  WifiOff,
  Circle,
  Archive,
  ChevronLeft,
  ChevronRight,
  X,
  User,
  CreditCard,
  FileText,
} from "lucide-react";
import { internetAccountsApi, routersApi, billingHistoryApi, subscriptionsApi } from "@/lib/api";
import { InternetAccount, Router, PaginatedResponse, SyncSummary, BillingHistoryEntry } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { useToast } from "@/components/ui/use-toast";
import { formatDate, formatRelative } from "@/lib/utils";
import { cn } from "@/lib/utils";

const MONTHS = [
  "January","February","March","April","May","June",
  "July","August","September","October","November","December",
];

const STATUS_COLOR: Record<string, string> = {
  pending: "bg-slate-100 text-slate-700",
  due: "bg-red-100 text-red-700",
  partial: "bg-amber-100 text-amber-700",
  paid: "bg-green-100 text-green-700",
  cancelled: "bg-zinc-100 text-zinc-500",
};

// ─── Sub-components ──────────────────────────────────────────────────────────

function StatPill({
  label,
  value,
  active,
  onClick,
  color = "default",
}: {
  label: string;
  value: number;
  active?: boolean;
  onClick?: () => void;
  color?: "green" | "red" | "orange" | "default";
}) {
  const colorMap = {
    green: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
    red: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
    orange: "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400",
    default: "bg-muted text-muted-foreground",
  };
  return (
    <button
      onClick={onClick}
      className={cn(
        "flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium transition-all",
        active
          ? "ring-2 ring-offset-1 ring-blue-500 " + colorMap[color]
          : colorMap[color],
        onClick && "hover:opacity-80 cursor-pointer"
      )}
    >
      {label}
      <span className="font-bold">{value}</span>
    </button>
  );
}

function OnlineBadge({ online }: { online: boolean }) {
  return online ? (
    <span className="inline-flex items-center gap-1 text-xs text-green-600 dark:text-green-400 font-medium">
      <Wifi className="w-3 h-3" /> Online
    </span>
  ) : (
    <span className="inline-flex items-center gap-1 text-xs text-slate-400 font-medium">
      <WifiOff className="w-3 h-3" /> Offline
    </span>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-medium mt-0.5 break-all">{value}</p>
    </div>
  );
}

// ─── Account Detail Dialog (PPPoE info) ────────────────────────────────────

function AccountDetailDialog({
  account,
  onClose,
}: {
  account: InternetAccount | null;
  onClose: () => void;
}) {
  if (!account) return null;
  return (
    <Dialog open={!!account} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {account.is_online ? (
              <Wifi className="w-4 h-4 text-green-500" />
            ) : (
              <WifiOff className="w-4 h-4 text-slate-400" />
            )}
            {account.username}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-4 text-sm">
          <div className="grid grid-cols-2 gap-3">
            <Field label="Router" value={account.router?.name ?? "—"} />
            <Field label="Profile" value={account.profile || "—"} />
            <Field label="Service" value={account.service || "—"} />
            <Field label="Status" value={account.disabled ? "Disabled" : "Enabled"} />
            <Field label="Caller ID" value={account.caller_id || "—"} />
            <Field label="Local Address" value={account.local_address || "—"} />
            <Field label="Remote Address" value={account.remote_address || "—"} />
            <Field label="Sync Status" value={account.sync_status} />
          </div>

          {account.comment && (
            <div>
              <p className="text-xs text-muted-foreground mb-1">Comment</p>
              <p className="text-sm">{account.comment}</p>
            </div>
          )}

          {account.is_online && (
            <div className="rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-3 space-y-2">
              <p className="text-xs font-semibold text-green-700 dark:text-green-400 uppercase tracking-wide">
                Live Session
              </p>
              <div className="grid grid-cols-2 gap-2">
                <Field label="IP Address" value={account.current_ip || "—"} />
                <Field label="Uptime" value={account.uptime || "—"} />
                <Field
                  label="Connected Since"
                  value={account.connected_since ? formatDate(account.connected_since) : "—"}
                />
                <Field label="Session ID" value={account.session_id || "—"} />
              </div>
            </div>
          )}

          {account.onu && (
            <div className="rounded-lg bg-muted/40 border p-3 space-y-2">
              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Linked ONU</p>
              <div className="grid grid-cols-2 gap-2">
                <Field label="OLT" value={account.onu.olt?.name || "—"} />
                <Field label="ONU ID" value={account.onu.onu_id || "—"} />
                <Field label="MAC Address" value={account.onu.mac_address || "—"} />
                <Field label="Model" value={account.onu.model || "—"} />
                <Field label="Status" value={account.onu.status} />
                <Field
                  label="Rx Power"
                  value={account.onu.rx_power != null ? `${account.onu.rx_power.toFixed(2)} dBm` : "—"}
                />
                <Field
                  label="Tx Power"
                  value={account.onu.tx_power != null ? `${account.onu.tx_power.toFixed(2)} dBm` : "—"}
                />
                <Field
                  label="Distance"
                  value={account.onu.distance != null ? `${account.onu.distance} m` : "—"}
                />
                <Field label="Serial" value={account.onu.serial_number || "—"} />
              </div>
            </div>
          )}

          <div className="text-xs text-muted-foreground pt-1 border-t flex justify-between">
            <span>Created {formatDate(account.created_at)}</span>
            <span>Last sync {account.last_sync_at ? formatRelative(account.last_sync_at) : "—"}</span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// ─── Customer Profile Drawer ──────────────────────────────────────────────

function CustomerProfileDialog({
  account,
  onClose,
}: {
  account: InternetAccount | null;
  onClose: () => void;
}) {
  const { data: historyData, isLoading: historyLoading } = useQuery({
    queryKey: ["billing-history", account?.id],
    queryFn: () =>
      billingHistoryApi
        .listByAccount(account!.id)
        .then((r) => r.data.data as BillingHistoryEntry[]),
    enabled: !!account,
  });

  const { data: subData } = useQuery({
    queryKey: ["subscription-active", account?.id],
    queryFn: () =>
      subscriptionsApi
        .getActiveForAccount(account!.id)
        .then((r) => r.data.data),
    enabled: !!account,
  });

  if (!account) return null;

  const history = historyData ?? [];
  const sub = subData ?? null;

  const totalPaid = history.reduce((sum, e) => sum + e.paid_amount, 0);
  const totalDue = history.reduce((sum, e) => sum + e.due_amount, 0);

  return (
    <Dialog open={!!account} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-4xl max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <User className="w-5 h-5 text-primary" />
            Customer Profile — {account.username}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 pr-1">
          {/* Customer Info */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 p-3 rounded-lg bg-muted/50 border">
            <div>
              <p className="text-xs text-muted-foreground">Username</p>
              <p className="text-sm font-semibold">{account.username}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Router</p>
              <p className="text-sm font-medium">{account.router?.name ?? "—"}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Package</p>
              <p className="text-sm font-medium">
                {sub?.package?.display_name ?? account.profile ?? "—"}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Monthly Charge</p>
              <p className="text-sm font-semibold">
                {sub ? `৳${sub.monthly_price.toFixed(2)}` : "—"}
              </p>
            </div>
          </div>

          {/* Summary pills */}
          {history.length > 0 && (
            <div className="flex gap-3 flex-wrap">
              <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400 text-xs font-medium">
                <FileText className="w-3.5 h-3.5" />
                {history.length} bills
              </div>
              <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400 text-xs font-medium">
                <CreditCard className="w-3.5 h-3.5" />
                Paid ৳{totalPaid.toFixed(2)}
              </div>
              {totalDue > 0 && (
                <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 text-xs font-medium">
                  Due ৳{totalDue.toFixed(2)}
                </div>
              )}
            </div>
          )}

          {/* Billing History Table */}
          <div>
            <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">
              Billing History
            </p>
            {historyLoading ? (
              <div className="space-y-2">
                {Array.from({ length: 5 }).map((_, i) => (
                  <Skeleton key={i} className="h-8 w-full" />
                ))}
              </div>
            ) : history.length === 0 ? (
              <div className="text-center py-10 text-muted-foreground text-sm border rounded-lg">
                No billing records found
              </div>
            ) : (
              <div className="border rounded-lg overflow-x-auto">
                <Table className="min-w-[820px]">
                  <TableHeader>
                    <TableRow>
                      <TableHead>Month</TableHead>
                      <TableHead>Bill #</TableHead>
                      <TableHead className="text-right">Amount</TableHead>
                      <TableHead className="text-right">Paid</TableHead>
                      <TableHead className="text-right">Due</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Method</TableHead>
                      <TableHead>Receipt #</TableHead>
                      <TableHead>Collected By</TableHead>
                      <TableHead>Paid At</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {history.map((entry) => (
                      <TableRow key={entry.id}>
                        <TableCell className="text-sm whitespace-nowrap font-medium">
                          {MONTHS[entry.billing_month - 1]} {entry.billing_year}
                        </TableCell>
                        <TableCell className="font-mono text-xs text-muted-foreground">
                          {entry.bill_number}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm">
                          ৳{entry.total_amount.toFixed(2)}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-green-600">
                          {entry.paid_amount > 0 ? `৳${entry.paid_amount.toFixed(2)}` : "—"}
                        </TableCell>
                        <TableCell className="text-right font-mono text-sm text-red-600">
                          {entry.due_amount > 0 ? `৳${entry.due_amount.toFixed(2)}` : "—"}
                        </TableCell>
                        <TableCell>
                          <span className={cn("px-2 py-0.5 rounded text-xs font-medium", STATUS_COLOR[entry.status] ?? "bg-slate-100")}>
                            {entry.status}
                          </span>
                        </TableCell>
                        <TableCell className="text-xs capitalize">
                          {entry.last_payment_method || "—"}
                        </TableCell>
                        <TableCell className="text-xs font-mono">
                          {entry.last_receipt_number || "—"}
                        </TableCell>
                        <TableCell className="text-xs whitespace-nowrap">
                          {entry.last_collected_by ?? "—"}
                        </TableCell>
                        <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                          {entry.last_paid_at ? formatDate(entry.last_paid_at) : "—"}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── Main Page ───────────────────────────────────────────────────────────────

export default function InternetAccountsPage() {
  const [search, setSearch] = useState("");
  const [routerFilter, setRouterFilter] = useState<string>("all");
  const [profileFilter, setProfileFilter] = useState<string>("all");
  const [onlineFilter, setOnlineFilter] = useState<boolean | undefined>(undefined);
  const [disabledFilter, setDisabledFilter] = useState<boolean | undefined>(undefined);
  const [archivedFilter, setArchivedFilter] = useState<boolean | undefined>(undefined);
  const [page, setPage] = useState(1);
  const [selectedAccount, setSelectedAccount] = useState<InternetAccount | null>(null);
  const [profileAccount, setProfileAccount] = useState<InternetAccount | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [syncSummary, setSyncSummary] = useState<SyncSummary | null>(null);

  const queryClient = useQueryClient();
  const { toast } = useToast();
  const PAGE_SIZE = 50;

  const params = {
    page,
    page_size: PAGE_SIZE,
    search: search || undefined,
    router_id: routerFilter !== "all" ? routerFilter : undefined,
    profile: profileFilter !== "all" ? profileFilter : undefined,
    is_online: onlineFilter,
    disabled: disabledFilter,
    archived: archivedFilter,
  };

  const { data, isLoading } = useQuery<{ data: PaginatedResponse<InternetAccount> }>({
    queryKey: ["internet-accounts", params],
    queryFn: () => internetAccountsApi.list(params),
    refetchInterval: 30_000,
  });

  const { data: statsData } = useQuery<{ data: { data: { total: number; enabled: number; disabled: number; online: number; offline: number; archived: number } } }>({
    queryKey: ["internet-accounts-stats"],
    queryFn: internetAccountsApi.stats,
    refetchInterval: 30_000,
  });

  const { data: profilesData } = useQuery<{ data: { data: string[] } }>({
    queryKey: ["internet-accounts-profiles"],
    queryFn: internetAccountsApi.profiles,
  });

  const { data: routersData } = useQuery<{ data: PaginatedResponse<Router> }>({
    queryKey: ["routers-all"],
    queryFn: () => routersApi.list({ page_size: 200 }),
  });

  const accounts = data?.data?.data ?? [];
  const total = data?.data?.total ?? 0;
  const totalPages = data?.data?.total_pages ?? 1;
  const stats = statsData?.data?.data;
  const profiles = profilesData?.data?.data ?? [];
  const routers = routersData?.data?.data ?? [];

  const handleSyncAll = async () => {
    setSyncing(true);
    try {
      const { data: res } = await internetAccountsApi.syncAll();
      queryClient.invalidateQueries({ queryKey: ["internet-accounts"] });
      queryClient.invalidateQueries({ queryKey: ["internet-accounts-stats"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard-stats"] });
      setSyncSummary(res.data as SyncSummary);
    } catch {
      toast({ title: "Sync failed", variant: "destructive" });
    } finally {
      setSyncing(false);
    }
  };

  const clearFilters = () => {
    setSearch("");
    setRouterFilter("all");
    setProfileFilter("all");
    setOnlineFilter(undefined);
    setDisabledFilter(undefined);
    setArchivedFilter(undefined);
    setPage(1);
  };

  const hasFilters =
    search ||
    routerFilter !== "all" ||
    profileFilter !== "all" ||
    onlineFilter !== undefined ||
    disabledFilter !== undefined ||
    archivedFilter !== undefined;

  return (
    <div>
      <Topbar
        title="Internet Accounts"
        breadcrumbs={[{ label: "Home" }, { label: "Internet Accounts" }]}
      />

      <div className="p-6 space-y-4">
        {/* Stats row */}
        {stats && (
          <div className="flex flex-wrap items-center gap-2">
            <StatPill label="Total" value={stats.total} />
            <StatPill
              label="Online"
              value={stats.online}
              color="green"
              active={onlineFilter === true}
              onClick={() => {
                setOnlineFilter(onlineFilter === true ? undefined : true);
                setPage(1);
              }}
            />
            <StatPill
              label="Offline"
              value={stats.offline}
              color="default"
              active={onlineFilter === false}
              onClick={() => {
                setOnlineFilter(onlineFilter === false ? undefined : false);
                setPage(1);
              }}
            />
            <StatPill
              label="Disabled"
              value={stats.disabled}
              color="orange"
              active={disabledFilter === true}
              onClick={() => {
                setDisabledFilter(disabledFilter === true ? undefined : true);
                setPage(1);
              }}
            />
            <StatPill
              label="Archived"
              value={stats.archived}
              color="red"
              active={archivedFilter === true}
              onClick={() => {
                setArchivedFilter(archivedFilter === true ? undefined : true);
                setPage(1);
              }}
            />
          </div>
        )}

        {/* Toolbar */}
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search username, comment…"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
              className="pl-9"
            />
          </div>

          <Select
            value={routerFilter}
            onValueChange={(v) => {
              setRouterFilter(v);
              setPage(1);
            }}
          >
            <SelectTrigger className="w-40">
              <SelectValue placeholder="All Routers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Routers</SelectItem>
              {routers.map((r) => (
                <SelectItem key={r.id} value={r.id}>
                  {r.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {profiles.length > 0 && (
            <Select
              value={profileFilter}
              onValueChange={(v) => {
                setProfileFilter(v);
                setPage(1);
              }}
            >
              <SelectTrigger className="w-40">
                <SelectValue placeholder="All Profiles" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Profiles</SelectItem>
                {profiles.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}

          {hasFilters && (
            <Button variant="ghost" size="sm" onClick={clearFilters}>
              <X className="w-3.5 h-3.5 mr-1" /> Clear
            </Button>
          )}

          <div className="ml-auto">
            <Button onClick={handleSyncAll} disabled={syncing}>
              {syncing ? (
                <Loader2 className="w-4 h-4 mr-1.5 animate-spin" />
              ) : (
                <RefreshCw className="w-4 h-4 mr-1.5" />
              )}
              Sync All
            </Button>
          </div>
        </div>

        {/* Table */}
        <div className="rounded-lg border bg-card overflow-hidden overflow-x-auto">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Router</TableHead>
                <TableHead>Profile</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Online</TableHead>
                <TableHead>ONU</TableHead>
                <TableHead>ONU Status</TableHead>
                <TableHead>Caller ID</TableHead>
                <TableHead>Local Address</TableHead>
                <TableHead>Remote Address</TableHead>
                <TableHead>IP Address</TableHead>
                <TableHead>Uptime</TableHead>
                <TableHead>Last Seen</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 13 }).map((_, j) => (
                      <TableCell key={j}>
                        <Skeleton className="h-4 w-full" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : accounts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={13} className="text-center py-16 text-muted-foreground">
                    No accounts found
                  </TableCell>
                </TableRow>
              ) : (
                accounts.map((acc) => (
                  <TableRow
                    key={acc.id}
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => setSelectedAccount(acc)}
                  >
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {acc.archived_at ? (
                          <Archive className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                        ) : acc.is_online ? (
                          <Circle className="w-2 h-2 fill-green-500 text-green-500 shrink-0" />
                        ) : (
                          <Circle className="w-2 h-2 fill-slate-300 text-slate-300 dark:fill-slate-600 dark:text-slate-600 shrink-0" />
                        )}
                        {/* Username click → Customer Profile (billing history) */}
                        <button
                          className="font-medium text-primary hover:underline text-left"
                          onClick={(e) => {
                            e.stopPropagation();
                            setProfileAccount(acc);
                          }}
                          title="View customer profile & billing history"
                        >
                          {acc.username}
                        </button>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {acc.router?.name ?? "—"}
                    </TableCell>
                    <TableCell>
                      {acc.profile ? (
                        <Badge variant="outline" className="text-xs font-normal">
                          {acc.profile}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {acc.archived_at ? (
                        <Badge variant="secondary">Archived</Badge>
                      ) : acc.disabled ? (
                        <Badge variant="secondary">Disabled</Badge>
                      ) : (
                        <Badge variant="default">Enabled</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <OnlineBadge online={acc.is_online} />
                    </TableCell>
                    <TableCell>
                      {acc.onu ? (
                        <div className="flex flex-col gap-0.5">
                          <span className="font-mono text-xs">{acc.onu.mac_address || "—"}</span>
                          {acc.onu.rx_power != null && (
                            <span className={`text-xs ${acc.onu.rx_power < -25 ? "text-red-500" : acc.onu.rx_power < -20 ? "text-yellow-500" : "text-green-600"}`}>
                              {acc.onu.rx_power.toFixed(1)} dBm
                            </span>
                          )}
                        </div>
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {acc.onu ? (
                        <span className={`inline-flex items-center gap-1 text-xs font-medium px-2 py-0.5 rounded-full ${
                          acc.onu.status === "online"
                            ? "bg-green-100 text-green-700"
                            : "bg-slate-100 text-slate-500"
                        }`}>
                          <Circle className={`w-1.5 h-1.5 ${acc.onu.status === "online" ? "fill-green-500 text-green-500" : "fill-slate-400 text-slate-400"}`} />
                          {acc.onu.status === "online" ? "Online" : "Offline"}
                        </span>
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {acc.caller_id || "—"}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {acc.local_address || "—"}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {acc.remote_address || "—"}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {acc.current_ip || "—"}
                    </TableCell>
                    <TableCell className="text-sm">
                      {acc.is_online ? acc.uptime || "—" : "—"}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {acc.last_seen ? formatRelative(acc.last_seen) : "—"}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span>{total} accounts</span>
          {totalPages > 1 && (
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                disabled={page === 1}
                onClick={() => setPage((p) => p - 1)}
              >
                <ChevronLeft className="w-4 h-4" />
              </Button>
              <span className="px-2">
                {page} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="icon"
                className="h-8 w-8"
                disabled={page === totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Account Detail Dialog (row click → PPPoE info) */}
      <AccountDetailDialog
        account={selectedAccount}
        onClose={() => setSelectedAccount(null)}
      />

      {/* Customer Profile Dialog (username click → billing history) */}
      <CustomerProfileDialog
        account={profileAccount}
        onClose={() => setProfileAccount(null)}
      />

      {/* ── Sync Summary Dialog ─────────────────────────────────────────── */}
      <Dialog open={!!syncSummary} onOpenChange={(v) => !v && setSyncSummary(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <RefreshCw className="w-5 h-5 text-green-500" />
              Synchronization Completed
            </DialogTitle>
          </DialogHeader>

          {syncSummary && (
            <div className="space-y-1 text-sm">
              <SummaryRow label="Routers Processed" value={syncSummary.routers_processed} />
              <SummaryRow label="Total Secrets Read" value={syncSummary.total_secrets} />
              <div className="border-t my-2" />
              <SummaryRow label="New Accounts" value={syncSummary.new_accounts} color="green" />
              <SummaryRow label="Updated Accounts" value={syncSummary.updated_accounts} color="blue" />
              <SummaryRow label="Archived Accounts" value={syncSummary.archived_accounts} color="orange" />
              <div className="border-t my-2" />
              <SummaryRow label="Online Users" value={syncSummary.online_users} color="green" />
              <SummaryRow label="Offline Users" value={syncSummary.offline_users} />
              <div className="border-t my-2" />
              <SummaryRow label="Errors" value={syncSummary.routers_failed} color={syncSummary.routers_failed > 0 ? "red" : undefined} />
              <SummaryRow label="Duration" value={`${(syncSummary.duration_ms / 1000).toFixed(1)}s`} />

              {syncSummary.errors && syncSummary.errors.length > 0 && (
                <div className="mt-3 rounded bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-2 space-y-1">
                  {syncSummary.errors.map((e, i) => (
                    <p key={i} className="text-xs text-red-700 dark:text-red-400">{e}</p>
                  ))}
                </div>
              )}
            </div>
          )}

          <DialogFooter>
            <Button onClick={() => setSyncSummary(null)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function SummaryRow({
  label,
  value,
  color,
}: {
  label: string;
  value: number | string;
  color?: "green" | "blue" | "orange" | "red";
}) {
  const colorMap: Record<string, string> = {
    green: "text-green-600 dark:text-green-400 font-semibold",
    blue: "text-blue-600 dark:text-blue-400 font-semibold",
    orange: "text-orange-500 dark:text-orange-400 font-semibold",
    red: "text-red-600 dark:text-red-400 font-semibold",
  };
  const colorClass = color ? (colorMap[color] ?? "font-medium") : "font-medium";

  return (
    <div className="flex items-center justify-between py-0.5">
      <span className="text-muted-foreground">{label}</span>
      <span className={colorClass}>{value}</span>
    </div>
  );
}
