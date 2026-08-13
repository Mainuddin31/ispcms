"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Search,
  RefreshCw,
  TestTube2,
  Edit,
  Trash2,
  MoreHorizontal,
  CheckCircle2,
  XCircle,
  Loader2,
  Eye,
  ChevronDown,
} from "lucide-react";
import { oltsApi, snmpProfilesApi } from "@/lib/api";
import { OLT, SNMPProfile, OLTSyncLog, PaginatedResponse, ApiResponse } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/use-toast";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

function formatRelative(dateStr?: string) {
  if (!dateStr) return "—";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    active: "bg-green-100 text-green-700",
    maintenance: "bg-yellow-100 text-yellow-700",
    offline: "bg-red-100 text-red-700",
    disabled: "bg-slate-100 text-slate-600",
  };
  return (
    <span className={cn("px-2 py-0.5 rounded text-xs font-medium capitalize", map[status] ?? "bg-slate-100 text-slate-600")}>
      {status}
    </span>
  );
}

// ── OLT Form Dialog ─────────────────────────────────────────────────────────

interface OLTFormData {
  name: string;
  vendor: string;
  model: string;
  snmp_profile_id: string;
  management_ip: string;
  snmp_version: "v2c" | "v3";
  snmp_port: number;
  timeout: number;
  retries: number;
  community: string;
  v3_username: string;
  v3_auth_protocol: string;
  v3_auth_password: string;
  v3_priv_protocol: string;
  v3_priv_password: string;
  pop: string;
  rack: string;
  cabinet: string;
  description: string;
  status: string;
  sync_interval: number;
  cli_protocol: string;
  cli_port: number;
  cli_username: string;
  cli_password: string;
  cli_enable_password: string;
}

const defaultForm: OLTFormData = {
  name: "", vendor: "", model: "", snmp_profile_id: "",
  management_ip: "", snmp_version: "v2c", snmp_port: 161,
  timeout: 5, retries: 2, community: "public",
  v3_username: "", v3_auth_protocol: "SHA", v3_auth_password: "",
  v3_priv_protocol: "AES", v3_priv_password: "",
  pop: "", rack: "", cabinet: "", description: "",
  status: "active", sync_interval: 30,
  cli_protocol: "telnet", cli_port: 23, cli_username: "",
  cli_password: "", cli_enable_password: "",
};

function OLTFormDialog({
  open,
  onClose,
  olt,
  profiles,
  onSaved,
}: {
  open: boolean;
  onClose: () => void;
  olt?: OLT | null;
  profiles: SNMPProfile[];
  onSaved: () => void;
}) {
  const { toast } = useToast();
  const [form, setForm] = useState<OLTFormData>(defaultForm);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setForm(olt
        ? {
            name: olt.name,
            vendor: olt.vendor ?? "",
            model: olt.model ?? "",
            snmp_profile_id: olt.snmp_profile_id,
            management_ip: olt.management_ip,
            snmp_version: olt.snmp_version,
            snmp_port: olt.snmp_port,
            timeout: olt.timeout,
            retries: olt.retries,
            community: olt.community ?? "public",
            v3_username: olt.v3_username ?? "",
            v3_auth_protocol: olt.v3_auth_protocol ?? "SHA",
            v3_auth_password: "",
            v3_priv_protocol: olt.v3_priv_protocol ?? "AES",
            v3_priv_password: "",
            pop: olt.pop ?? "",
            rack: olt.rack ?? "",
            cabinet: olt.cabinet ?? "",
            description: olt.description ?? "",
            status: olt.status,
            sync_interval: olt.sync_interval,
            cli_protocol: olt.cli_protocol ?? "telnet",
            cli_port: olt.cli_port ?? 23,
            cli_username: olt.cli_username ?? "",
            cli_password: "",
            cli_enable_password: "",
          }
        : defaultForm
      );
    }
  }, [open, olt]);

  const set = (k: keyof OLTFormData, v: string | number) =>
    setForm((f) => ({ ...f, [k]: v }));

  const handleSave = async () => {
    if (!form.name || !form.management_ip || !form.snmp_profile_id) {
      toast({ title: "Required fields missing", variant: "destructive" });
      return;
    }
    setSaving(true);
    try {
      const payload = { ...form };
      if (olt) {
        await oltsApi.update(olt.id, payload);
        toast({ title: "OLT updated" });
      } else {
        await oltsApi.create(payload);
        toast({ title: "OLT created" });
      }
      onSaved();
      onClose();
    } catch {
      toast({ title: "Error saving OLT", variant: "destructive" });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{olt ? "Edit OLT" : "Add OLT"}</DialogTitle>
        </DialogHeader>
        <div className="grid grid-cols-2 gap-4 py-2">
          {/* Basic info */}
          <div className="space-y-1 col-span-2 md:col-span-1">
            <Label>Name *</Label>
            <Input value={form.name} onChange={(e) => set("name", e.target.value)} placeholder="Main OLT" />
          </div>
          <div className="space-y-1 col-span-2 md:col-span-1">
            <Label>Management IP *</Label>
            <Input value={form.management_ip} onChange={(e) => set("management_ip", e.target.value)} placeholder="192.168.1.1" />
          </div>
          <div className="space-y-1">
            <Label>Vendor</Label>
            <Input value={form.vendor} onChange={(e) => set("vendor", e.target.value)} placeholder="BDCOM" />
          </div>
          <div className="space-y-1">
            <Label>Model</Label>
            <Input value={form.model} onChange={(e) => set("model", e.target.value)} placeholder="P3608" />
          </div>

          {/* SNMP Profile */}
          <div className="space-y-1 col-span-2">
            <Label>SNMP Profile *</Label>
            <Select value={form.snmp_profile_id} onValueChange={(v) => set("snmp_profile_id", v)}>
              <SelectTrigger>
                <SelectValue placeholder="Select profile…" />
              </SelectTrigger>
              <SelectContent>
                {profiles.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {p.name} ({p.vendor} · {p.technology})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* SNMP version */}
          <div className="space-y-1">
            <Label>SNMP Version</Label>
            <Select value={form.snmp_version} onValueChange={(v) => set("snmp_version", v as "v2c" | "v3")}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="v2c">v2c</SelectItem>
                <SelectItem value="v3">v3</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <Label>SNMP Port</Label>
            <Input type="number" value={form.snmp_port} onChange={(e) => set("snmp_port", Number(e.target.value))} />
          </div>

          {/* v2c community */}
          {form.snmp_version === "v2c" && (
            <div className="space-y-1 col-span-2">
              <Label>Community String</Label>
              <Input value={form.community} onChange={(e) => set("community", e.target.value)} placeholder="public" />
            </div>
          )}

          {/* v3 credentials */}
          {form.snmp_version === "v3" && (
            <>
              <div className="space-y-1 col-span-2">
                <Label>Username</Label>
                <Input value={form.v3_username} onChange={(e) => set("v3_username", e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label>Auth Protocol</Label>
                <Select value={form.v3_auth_protocol} onValueChange={(v) => set("v3_auth_protocol", v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="SHA">SHA</SelectItem>
                    <SelectItem value="MD5">MD5</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label>Auth Password</Label>
                <Input type="password" value={form.v3_auth_password} onChange={(e) => set("v3_auth_password", e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label>Priv Protocol</Label>
                <Select value={form.v3_priv_protocol} onValueChange={(v) => set("v3_priv_protocol", v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="AES">AES</SelectItem>
                    <SelectItem value="DES">DES</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label>Priv Password</Label>
                <Input type="password" value={form.v3_priv_password} onChange={(e) => set("v3_priv_password", e.target.value)} />
              </div>
            </>
          )}

          {/* Timing */}
          <div className="space-y-1">
            <Label>Timeout (sec)</Label>
            <Input type="number" value={form.timeout} onChange={(e) => set("timeout", Number(e.target.value))} />
          </div>
          <div className="space-y-1">
            <Label>Retries</Label>
            <Input type="number" value={form.retries} onChange={(e) => set("retries", Number(e.target.value))} />
          </div>
          <div className="space-y-1">
            <Label>Sync Interval (min, 0=off)</Label>
            <Input type="number" value={form.sync_interval} onChange={(e) => set("sync_interval", Number(e.target.value))} />
          </div>
          <div className="space-y-1">
            <Label>Status</Label>
            <Select value={form.status} onValueChange={(v) => set("status", v)}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="maintenance">Maintenance</SelectItem>
                <SelectItem value="offline">Offline</SelectItem>
                <SelectItem value="disabled">Disabled</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* CLI access — Richerlink only */}
          {form.vendor.toLowerCase().includes("richerlink") && (
            <>
              <div className="col-span-2 border-t pt-3 mt-1">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3">
                  CLI Access (Telnet) — for ONU auto-linking
                </p>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-1">
                    <Label>CLI Username</Label>
                    <Input
                      value={form.cli_username}
                      onChange={(e) => set("cli_username", e.target.value)}
                      placeholder="admin"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label>CLI Port</Label>
                    <Input
                      type="number"
                      value={form.cli_port}
                      onChange={(e) => set("cli_port", Number(e.target.value))}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label>CLI Password</Label>
                    <Input
                      type="password"
                      value={form.cli_password}
                      onChange={(e) => set("cli_password", e.target.value)}
                      placeholder={form.cli_username ? "leave blank to keep existing" : ""}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label>Enable Password</Label>
                    <Input
                      type="password"
                      value={form.cli_enable_password}
                      onChange={(e) => set("cli_enable_password", e.target.value)}
                      placeholder="leave blank = same as CLI password"
                    />
                  </div>
                </div>
              </div>
            </>
          )}

          {/* Location */}
          <div className="space-y-1">
            <Label>POP</Label>
            <Input value={form.pop} onChange={(e) => set("pop", e.target.value)} placeholder="Main POP" />
          </div>
          <div className="space-y-1">
            <Label>Rack</Label>
            <Input value={form.rack} onChange={(e) => set("rack", e.target.value)} placeholder="Rack A" />
          </div>
          <div className="space-y-1 col-span-2">
            <Label>Description</Label>
            <Input value={form.description} onChange={(e) => set("description", e.target.value)} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            {olt ? "Update" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function OLTsPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingOLT, setEditingOLT] = useState<OLT | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<OLT | null>(null);
  const [syncingId, setSyncingId] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [logsOlt, setLogsOlt] = useState<OLT | null>(null);

  const { toast } = useToast();
  const queryClient = useQueryClient();

  const { data: oltsData, isLoading } = useQuery<{ data: PaginatedResponse<OLT> }>({
    queryKey: ["olts", search, statusFilter],
    queryFn: () =>
      oltsApi.list({ search: search || undefined, status: statusFilter === "all" ? undefined : statusFilter }),
  });

  const { data: profilesData } = useQuery<{ data: { data: SNMPProfile[] } }>({
    queryKey: ["snmp-profiles"],
    queryFn: () => snmpProfilesApi.list(),
  });

  const { data: syncLogsData } = useQuery<{ data: { data: OLTSyncLog[] } }>({
    queryKey: ["olt-logs", logsOlt?.id],
    queryFn: () => oltsApi.syncLogs(logsOlt!.id, 20),
    enabled: !!logsOlt,
  });

  const olts = oltsData?.data?.data ?? [];
  const profiles = profilesData?.data?.data ?? [];

  const deleteMutation = useMutation({
    mutationFn: (id: string) => oltsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["olts"] });
      toast({ title: "OLT deleted" });
      setDeleteTarget(null);
    },
    onError: () => toast({ title: "Delete failed", variant: "destructive" }),
  });

  const handleSync = async (olt: OLT) => {
    setSyncingId(olt.id);
    try {
      await oltsApi.sync(olt.id);
      toast({ title: `Sync started for ${olt.name}` });
      queryClient.invalidateQueries({ queryKey: ["olts"] });
      queryClient.invalidateQueries({ queryKey: ["olt-stats"] });
    } catch {
      toast({ title: "Sync failed", variant: "destructive" });
    } finally {
      setSyncingId(null);
    }
  };

  const handleTest = async (olt: OLT) => {
    setTestingId(olt.id);
    try {
      await oltsApi.testConnection(olt.id);
      toast({ title: `Connection OK for ${olt.name}` });
    } catch {
      toast({ title: "Connection failed", variant: "destructive" });
    } finally {
      setTestingId(null);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Topbar title="OLT Management" />
      <div className="flex-1 overflow-auto p-6">
        {/* Header row */}
        <div className="flex items-center justify-between mb-4 gap-3 flex-wrap">
          <div className="flex gap-2 flex-1">
            <div className="relative max-w-xs flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
              <Input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search OLTs…"
                className="pl-9"
              />
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All statuses</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="maintenance">Maintenance</SelectItem>
                <SelectItem value="offline">Offline</SelectItem>
                <SelectItem value="disabled">Disabled</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button onClick={() => { setEditingOLT(null); setDialogOpen(true); }} className="gap-2">
            <Plus className="w-4 h-4" /> Add OLT
          </Button>
        </div>

        {/* Table */}
        <div className="border rounded-lg overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name / IP</TableHead>
                <TableHead>Profile</TableHead>
                <TableHead>Version</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Last Sync</TableHead>
                <TableHead>Interval</TableHead>
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 7 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : olts.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-slate-400 py-8">
                    No OLTs found. Add your first OLT.
                  </TableCell>
                </TableRow>
              ) : (
                olts.map((olt) => (
                  <TableRow key={olt.id}>
                    <TableCell>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium text-sm">{olt.name}</p>
                          {olt.cli_protocol && (
                            <span className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-[10px] font-medium uppercase">
                              {olt.cli_protocol}
                            </span>
                          )}
                        </div>
                        <p className="text-xs text-slate-400">{olt.management_ip}</p>
                        {olt.vendor && <p className="text-xs text-slate-400">{olt.vendor} {olt.model}</p>}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm">{olt.snmp_profile?.name ?? "—"}</TableCell>
                    <TableCell>
                      <span className="px-2 py-0.5 bg-slate-100 rounded text-xs font-mono">{olt.snmp_version}</span>
                    </TableCell>
                    <TableCell><StatusBadge status={olt.status} /></TableCell>
                    <TableCell className="text-sm text-slate-500">{formatRelative(olt.last_sync_at)}</TableCell>
                    <TableCell className="text-sm text-slate-500">
                      {olt.sync_interval > 0 ? `${olt.sync_interval}m` : "Manual"}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="w-8 h-8">
                            <MoreHorizontal className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => handleTest(olt)}
                            disabled={testingId === olt.id}
                          >
                            {testingId === olt.id ? (
                              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                            ) : (
                              <TestTube2 className="w-4 h-4 mr-2" />
                            )}
                            Test Connection
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => handleSync(olt)}
                            disabled={syncingId === olt.id}
                          >
                            {syncingId === olt.id ? (
                              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                            ) : (
                              <RefreshCw className="w-4 h-4 mr-2" />
                            )}
                            Sync Now
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => setLogsOlt(olt)}>
                            <Eye className="w-4 h-4 mr-2" /> View Sync Logs
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem onClick={() => { setEditingOLT(olt); setDialogOpen(true); }}>
                            <Edit className="w-4 h-4 mr-2" /> Edit
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            className="text-red-600"
                            onClick={() => setDeleteTarget(olt)}
                          >
                            <Trash2 className="w-4 h-4 mr-2" /> Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* OLT Form Dialog */}
      <OLTFormDialog
        open={dialogOpen}
        onClose={() => { setDialogOpen(false); setEditingOLT(null); }}
        olt={editingOLT}
        profiles={profiles}
        onSaved={() => queryClient.invalidateQueries({ queryKey: ["olts"] })}
      />

      {/* Delete Confirm */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(v) => !v && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete OLT?</AlertDialogTitle>
            <AlertDialogDescription>
              This will soft-delete <strong>{deleteTarget?.name}</strong> and all associated PON ports and ONUs. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-red-600 hover:bg-red-700"
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Sync Logs Dialog */}
      <Dialog open={!!logsOlt} onOpenChange={(v) => !v && setLogsOlt(null)}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Sync Logs — {logsOlt?.name}</DialogTitle>
          </DialogHeader>
          <div className="space-y-2 mt-2">
            {(syncLogsData?.data?.data ?? []).length === 0 ? (
              <p className="text-sm text-slate-500 text-center py-4">No sync logs yet.</p>
            ) : (
              (syncLogsData?.data?.data ?? []).map((log) => (
                <div key={log.id} className="border rounded-lg p-3 text-sm">
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center gap-2">
                      {log.status === "success" ? (
                        <CheckCircle2 className="w-4 h-4 text-green-500" />
                      ) : log.status === "failed" ? (
                        <XCircle className="w-4 h-4 text-red-500" />
                      ) : (
                        <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />
                      )}
                      <span className="font-medium capitalize">{log.status}</span>
                    </div>
                    <span className="text-slate-400 text-xs">
                      {new Date(log.started_at).toLocaleString()}
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-2 text-xs text-slate-500 mt-2">
                    <span>Ports: {log.ports_discovered}</span>
                    <span>ONUs: {log.onus_discovered}</span>
                    <span>New: {log.new_onus}</span>
                    <span>Updated: {log.updated_onus}</span>
                    <span>Archived: {log.archived_onus}</span>
                    <span>Duration: {log.duration_ms}ms</span>
                  </div>
                  {log.error_message && (
                    <p className="text-red-600 text-xs mt-2">{log.error_message}</p>
                  )}
                </div>
              ))
            )}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
