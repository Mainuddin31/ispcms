"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Search,
  Wifi,
  WifiOff,
  Link2,
  Link2Off,
  ChevronLeft,
  ChevronRight,
  Filter,
} from "lucide-react";
import { onusApi, oltsApi, internetAccountsApi } from "@/lib/api";
import { ONU, OLT, PONPort, InternetAccount, PaginatedResponse } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/use-toast";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Sheet, SheetContent, SheetHeader, SheetTitle,
} from "@/components/ui/sheet";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

function StatusDot({ status }: { status: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={cn("w-2 h-2 rounded-full", status === "online" ? "bg-green-500" : "bg-slate-400")} />
      <span className="text-sm capitalize">{status}</span>
    </span>
  );
}

function formatPower(val?: number) {
  if (val === undefined || val === null) return "—";
  return `${val.toFixed(1)} dBm`;
}

function powerColor(val?: number | null): string {
  if (val === undefined || val === null) return "";
  if (val >= -20) return "text-green-600";
  if (val >= -25) return "text-yellow-500";
  return "text-red-500";
}

function formatDistance(val?: number) {
  if (val === undefined || val === null) return "—";
  if (val >= 1000) return `${(val / 1000).toFixed(2)} km`;
  return `${val} m`;
}

function formatRelative(dateStr?: string) {
  if (!dateStr) return "—";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return new Date(dateStr).toLocaleDateString();
}

// ── Link Account Dialog ──────────────────────────────────────────────────────

function LinkAccountDialog({
  open,
  onu,
  onClose,
  onLinked,
}: {
  open: boolean;
  onu: ONU | null;
  onClose: () => void;
  onLinked: () => void;
}) {
  const { toast } = useToast();
  const [accountSearch, setAccountSearch] = useState("");
  const [selectedAccountId, setSelectedAccountId] = useState<string>("");
  const [saving, setSaving] = useState(false);

  const { data: accountsData } = useQuery<{ data: { data: InternetAccount[] } }>({
    queryKey: ["ia-link-search", accountSearch],
    queryFn: () => internetAccountsApi.list({ search: accountSearch || undefined, page_size: 20 }),
    enabled: open,
  });

  const accounts = accountsData?.data?.data ?? [];

  const handleLink = async () => {
    if (!onu) return;
    setSaving(true);
    try {
      await onusApi.link(onu.id, selectedAccountId || null);
      toast({ title: selectedAccountId ? "ONU linked to account" : "ONU unlinked" });
      onLinked();
      onClose();
    } catch {
      toast({ title: "Failed to link ONU", variant: "destructive" });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Link ONU to Account</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <p className="text-sm text-slate-500">ONU: <span className="font-mono">{onu?.onu_id}</span></p>
          {onu?.internet_account_id && (
            <div className="flex items-center justify-between p-2 bg-blue-50 rounded text-sm">
              <span>Currently linked: <strong>{onu.internet_account?.username}</strong></span>
              <Button
                variant="ghost"
                size="sm"
                className="text-red-600 h-7"
                onClick={async () => {
                  if (!onu) return;
                  setSaving(true);
                  try {
                    await onusApi.link(onu.id, null);
                    toast({ title: "ONU unlinked" });
                    onLinked();
                    onClose();
                  } catch {
                    toast({ title: "Failed", variant: "destructive" });
                  } finally {
                    setSaving(false);
                  }
                }}
              >
                <Link2Off className="w-3 h-3 mr-1" /> Unlink
              </Button>
            </div>
          )}
          <div className="space-y-1">
            <Label>Search Internet Account</Label>
            <Input
              value={accountSearch}
              onChange={(e) => setAccountSearch(e.target.value)}
              placeholder="Username or IP…"
            />
          </div>
          {accounts.length > 0 && (
            <div className="border rounded-lg divide-y max-h-48 overflow-y-auto">
              {accounts.map((acc) => (
                <button
                  key={acc.id}
                  onClick={() => setSelectedAccountId(acc.id)}
                  className={cn(
                    "w-full text-left px-3 py-2 text-sm hover:bg-slate-50 transition-colors",
                    selectedAccountId === acc.id && "bg-blue-50"
                  )}
                >
                  <span className="font-medium">{acc.username}</span>
                  {acc.remote_address && (
                    <span className="text-slate-400 text-xs ml-2">{acc.remote_address}</span>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleLink} disabled={saving || !selectedAccountId}>
            <Link2 className="w-4 h-4 mr-2" /> Link Account
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── ONU Detail Sheet ─────────────────────────────────────────────────────────

function ONUDetailSheet({
  onu,
  onClose,
  onLinkClick,
}: {
  onu: ONU | null;
  onClose: () => void;
  onLinkClick: () => void;
}) {
  return (
    <Sheet open={!!onu} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="w-96 overflow-y-auto">
        <SheetHeader>
          <SheetTitle>ONU Detail</SheetTitle>
        </SheetHeader>
        {onu && (
          <div className="mt-4 space-y-4">
            <div className="flex items-center justify-between">
              <StatusDot status={onu.status} />
              <span className="text-xs bg-slate-100 px-2 py-1 rounded font-mono">
                {onu.reg_status}
              </span>
            </div>

            <div className="space-y-3">
              <DetailRow label="ONU ID" value={onu.onu_id} mono />
              <DetailRow label="MAC Address" value={onu.mac_address} mono />
              <DetailRow label="Serial Number" value={onu.serial_number} mono />
              <DetailRow label="Vendor" value={onu.vendor} />
              <DetailRow label="Model" value={onu.model} />
              <DetailRow label="OLT" value={onu.olt?.name} />
              <DetailRow label="Port / Slot" value={`Port ${onu.port_index} · Slot ${onu.onu_slot}`} />
              <DetailRow label="Rx Power" value={formatPower(onu.rx_power)} valueClass={powerColor(onu.rx_power)} />
              <DetailRow label="Tx Power" value={formatPower(onu.tx_power)} />
              <DetailRow label="Distance" value={formatDistance(onu.distance)} />
              <DetailRow label="Last Online" value={formatRelative(onu.last_online_at)} />
            </div>

            <div className="border-t pt-4">
              <p className="text-sm font-medium mb-2">Internet Account</p>
              {onu.internet_account ? (
                <div className="p-3 bg-green-50 rounded-lg text-sm">
                  <p className="font-medium">{onu.internet_account.username}</p>
                  {onu.internet_account.remote_address && (
                    <p className="text-slate-500 text-xs mt-0.5">{onu.internet_account.remote_address}</p>
                  )}
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2 w-full"
                    onClick={onLinkClick}
                  >
                    Change Account
                  </Button>
                </div>
              ) : (
                <div className="p-3 bg-amber-50 rounded-lg text-sm text-amber-700">
                  Not linked to any account.
                  <Button
                    size="sm"
                    className="mt-2 w-full"
                    onClick={onLinkClick}
                  >
                    <Link2 className="w-3 h-3 mr-1" /> Link Account
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}

function DetailRow({ label, value, mono, valueClass }: { label: string; value?: string | null; mono?: boolean; valueClass?: string }) {
  return (
    <div className="flex justify-between text-sm">
      <span className="text-slate-500">{label}</span>
      <span className={cn("font-medium text-right max-w-[60%] truncate", mono && "font-mono text-xs", valueClass)}>
        {value ?? "—"}
      </span>
    </div>
  );
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function ONUsPage() {
  const [search, setSearch] = useState("");
  const [oltFilter, setOltFilter] = useState("all");
  const [portFilter, setPortFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [unlinkedOnly, setUnlinkedOnly] = useState(false);
  const [page, setPage] = useState(1);
  const [selectedONU, setSelectedONU] = useState<ONU | null>(null);
  const [linkTarget, setLinkTarget] = useState<ONU | null>(null);

  const queryClient = useQueryClient();

  const { data: onusData, isLoading } = useQuery<{ data: PaginatedResponse<ONU> }>({
    queryKey: ["onus", search, oltFilter, portFilter, statusFilter, unlinkedOnly, page],
    queryFn: () =>
      onusApi.list({
        page,
        page_size: 25,
        search: search || undefined,
        olt_id: oltFilter === "all" ? undefined : oltFilter,
        pon_port_id: portFilter === "all" ? undefined : portFilter,
        status: statusFilter === "all" ? undefined : statusFilter,
        unlinked: unlinkedOnly || undefined,
      }),
  });

  const { data: oltsData } = useQuery({
    queryKey: ["olts-select"],
    queryFn: () => oltsApi.list({}),
  });

  // Fetch PON ports when an OLT is selected
  const { data: portsData } = useQuery({
    queryKey: ["pon-ports", oltFilter],
    queryFn: () => oltsApi.ponPorts(oltFilter),
    enabled: oltFilter !== "all",
  });
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const ponPorts: PONPort[] = (portsData as any)?.data?.data ?? [];

  const onus = onusData?.data?.data ?? [];
  const total = onusData?.data?.total ?? 0;
  const totalPages = onusData?.data?.total_pages ?? 1;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const olts: OLT[] = (oltsData as any)?.data?.data ?? [];

  return (
    <div className="flex flex-col h-full">
      <Topbar title="ONU Inventory" />
      <div className="flex-1 overflow-auto p-6">
        {/* Filters */}
        <div className="flex flex-wrap gap-2 mb-4">
          <div className="relative flex-1 min-w-40 max-w-xs">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <Input
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              placeholder="Search ONU ID, MAC, serial…"
              className="pl-9"
            />
          </div>
          <Select value={oltFilter} onValueChange={(v) => { setOltFilter(v); setPortFilter("all"); setPage(1); }}>
            <SelectTrigger className="w-44">
              <SelectValue placeholder="All OLTs" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All OLTs</SelectItem>
              {olts.map((o) => (
                <SelectItem key={o.id} value={o.id}>{o.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          {oltFilter !== "all" && (
            <Select value={portFilter} onValueChange={(v) => { setPortFilter(v); setPage(1); }}>
              <SelectTrigger className="w-36">
                <SelectValue placeholder="All Ports" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Ports</SelectItem>
                {ponPorts.map((p) => (
                  <SelectItem key={p.id} value={p.id}>{p.port_name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1); }}>
            <SelectTrigger className="w-36">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="online">Online</SelectItem>
              <SelectItem value="offline">Offline</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant={unlinkedOnly ? "default" : "outline"}
            size="sm"
            onClick={() => { setUnlinkedOnly((v) => !v); setPage(1); }}
            className="gap-1.5"
          >
            <Filter className="w-3.5 h-3.5" />
            Unlinked only
          </Button>
          <span className="self-center text-sm text-slate-400 ml-auto">{total} ONUs</span>
        </div>

        {/* Table */}
        <div className="border rounded-lg overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ONU ID</TableHead>
                <TableHead>MAC / Serial</TableHead>
                <TableHead>OLT / Port</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Rx Power</TableHead>
                <TableHead>Last Online</TableHead>
                <TableHead>Account</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 7 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : onus.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-slate-400 py-8">
                    No ONUs found.
                  </TableCell>
                </TableRow>
              ) : (
                onus.map((onu) => (
                  <TableRow
                    key={onu.id}
                    className="cursor-pointer hover:bg-slate-50"
                    onClick={() => setSelectedONU(onu)}
                  >
                    <TableCell>
                      <span className="font-mono text-xs">{onu.onu_id}</span>
                    </TableCell>
                    <TableCell>
                      <div>
                        {onu.mac_address && <p className="font-mono text-xs">{onu.mac_address}</p>}
                        {onu.serial_number && <p className="text-xs text-slate-400">{onu.serial_number}</p>}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div>
                        <p className="text-sm">{onu.olt?.name ?? "—"}</p>
                        <p className="text-xs text-slate-400">P{onu.port_index}/{onu.onu_slot}</p>
                      </div>
                    </TableCell>
                    <TableCell><StatusDot status={onu.status} /></TableCell>
                    <TableCell className={`text-sm font-medium ${powerColor(onu.rx_power)}`}>{formatPower(onu.rx_power)}</TableCell>
                    <TableCell className="text-sm text-slate-500">{formatRelative(onu.last_online_at)}</TableCell>
                    <TableCell>
                      {onu.internet_account ? (
                        <span className="text-xs bg-green-50 text-green-700 px-2 py-0.5 rounded">
                          {onu.internet_account.username}
                        </span>
                      ) : (
                        <span className="text-xs text-slate-400">—</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-end gap-2 mt-3">
            <Button variant="outline" size="sm" onClick={() => setPage((p) => p - 1)} disabled={page <= 1}>
              <ChevronLeft className="w-4 h-4" />
            </Button>
            <span className="text-sm text-slate-500">Page {page} of {totalPages}</span>
            <Button variant="outline" size="sm" onClick={() => setPage((p) => p + 1)} disabled={page >= totalPages}>
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        )}
      </div>

      {/* Detail Sheet */}
      <ONUDetailSheet
        onu={selectedONU}
        onClose={() => setSelectedONU(null)}
        onLinkClick={() => {
          setLinkTarget(selectedONU);
          setSelectedONU(null);
        }}
      />

      {/* Link Account Dialog */}
      <LinkAccountDialog
        open={!!linkTarget}
        onu={linkTarget}
        onClose={() => setLinkTarget(null)}
        onLinked={() => {
          queryClient.invalidateQueries({ queryKey: ["onus"] });
          setLinkTarget(null);
        }}
      />
    </div>
  );
}
