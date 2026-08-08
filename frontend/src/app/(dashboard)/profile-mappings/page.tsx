"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Pencil, Trash2, Loader2, Search, ArrowLeftRight, AlertTriangle } from "lucide-react";
import { profileMappingsApi, packagesApi } from "@/lib/api";
import { ProfileMapping, Package, PaginatedResponse } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { useToast } from "@/components/ui/use-toast";
import { formatDate } from "@/lib/utils";

const EMPTY_FORM = { mikrotik_profile: "", package_id: "", notes: "" };

export default function ProfileMappingsPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [form, setForm] = useState(EMPTY_FORM);
  const [editId, setEditId] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["profile-mappings", page, search],
    queryFn: () =>
      profileMappingsApi
        .list({ page, page_size: 20, search: search || undefined })
        .then((r) => r.data as PaginatedResponse<ProfileMapping> & { data: ProfileMapping[] }),
  });

  const { data: unmappedData } = useQuery({
    queryKey: ["unmapped-profiles"],
    queryFn: () => profileMappingsApi.unmapped().then((r) => r.data.data as string[]),
  });

  const { data: packagesData } = useQuery({
    queryKey: ["packages-active"],
    queryFn: () => packagesApi.listActive().then((r) => r.data.data as Package[]),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => profileMappingsApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["profile-mappings"] });
      qc.invalidateQueries({ queryKey: ["unmapped-profiles"] });
      toast({ title: "Mapping deleted" });
      setDeleteId(null);
    },
    onError: (e: unknown) => {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Delete failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    },
  });

  function openCreate(prefillProfile?: string) {
    setForm({ ...EMPTY_FORM, mikrotik_profile: prefillProfile ?? "" });
    setEditId(null);
    setDialogOpen(true);
  }

  function openEdit(m: ProfileMapping) {
    setForm({ mikrotik_profile: m.mikrotik_profile, package_id: m.package_id, notes: m.notes ?? "" });
    setEditId(m.id);
    setDialogOpen(true);
  }

  async function handleSave() {
    if (!form.mikrotik_profile || !form.package_id) {
      toast({ title: "Validation", description: "Profile and package are required", variant: "destructive" });
      return;
    }
    setSaving(true);
    try {
      if (editId) {
        await profileMappingsApi.update(editId, { package_id: form.package_id, notes: form.notes });
        toast({ title: "Mapping updated" });
      } else {
        await profileMappingsApi.create(form);
        toast({ title: "Mapping created" });
      }
      qc.invalidateQueries({ queryKey: ["profile-mappings"] });
      qc.invalidateQueries({ queryKey: ["unmapped-profiles"] });
      setDialogOpen(false);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Save failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    } finally {
      setSaving(false);
    }
  }

  const mappings = data?.data ?? [];
  const totalPages = data?.total_pages ?? 1;
  const unmapped = unmappedData ?? [];
  const activePackages = packagesData ?? [];

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Profile Mappings" subtitle="Link MikroTik PPPoE profiles to billing packages" />

      <div className="flex-1 overflow-auto p-6 space-y-4">
        {/* Unmapped warning */}
        {unmapped.length > 0 && (
          <div className="rounded-lg border border-amber-200 bg-amber-50 dark:bg-amber-950/20 dark:border-amber-800 p-4">
            <div className="flex items-start gap-3">
              <AlertTriangle className="h-5 w-5 text-amber-600 shrink-0 mt-0.5" />
              <div className="flex-1">
                <p className="font-medium text-amber-800 dark:text-amber-400">
                  {unmapped.length} unmapped profile{unmapped.length > 1 ? "s" : ""}
                </p>
                <p className="text-sm text-amber-700 dark:text-amber-500 mt-1">
                  Billing is paused for accounts using these profiles:
                </p>
                <div className="flex flex-wrap gap-2 mt-2">
                  {unmapped.map((p) => (
                    <button
                      key={p}
                      className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs bg-amber-100 dark:bg-amber-900 text-amber-800 dark:text-amber-300 hover:bg-amber-200 transition-colors"
                      onClick={() => openCreate(p)}
                    >
                      <Plus className="h-3 w-3" /> {p}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Toolbar */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search profiles…"
              className="pl-9"
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            />
          </div>
          <Button onClick={() => openCreate()}>
            <Plus className="h-4 w-4 mr-2" /> Add Mapping
          </Button>
        </div>

        {/* Table */}
        <div className="rounded-lg border overflow-x-auto">
          <Table className="min-w-[600px]">
            <TableHeader>
              <TableRow>
                <TableHead>MikroTik Profile</TableHead>
                <TableHead>Package</TableHead>
                <TableHead>Notes</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="w-24">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 5 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 5 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : mappings.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center py-12 text-muted-foreground">
                    <ArrowLeftRight className="mx-auto h-8 w-8 mb-2 opacity-40" />
                    No mappings yet
                  </TableCell>
                </TableRow>
              ) : (
                mappings.map((m) => (
                  <TableRow key={m.id}>
                    <TableCell className="font-mono text-xs">{m.mikrotik_profile}</TableCell>
                    <TableCell>
                      <div>
                        <p className="font-medium text-sm">{m.package?.display_name ?? "—"}</p>
                        <p className="text-xs text-muted-foreground">৳{m.package?.monthly_price.toFixed(2)}/mo</p>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{m.notes || "—"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDate(m.created_at)}</TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button size="icon" variant="ghost" onClick={() => openEdit(m)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button size="icon" variant="ghost" className="text-destructive" onClick={() => setDeleteId(m.id)}>
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {totalPages > 1 && (
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>Page {page} of {totalPages}</span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>Previous</Button>
              <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>Next</Button>
            </div>
          </div>
        )}
      </div>

      {/* Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editId ? "Edit Mapping" : "Add Profile Mapping"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1">
              <Label>MikroTik Profile <span className="text-destructive">*</span></Label>
              <Input
                placeholder="e.g. pppoe-10mbps"
                value={form.mikrotik_profile}
                disabled={!!editId}
                onChange={(e) => setForm(f => ({ ...f, mikrotik_profile: e.target.value }))}
              />
            </div>
            <div className="space-y-1">
              <Label>Billing Package <span className="text-destructive">*</span></Label>
              <Select value={form.package_id} onValueChange={(v) => setForm(f => ({ ...f, package_id: v }))}>
                <SelectTrigger>
                  <SelectValue placeholder="Select package…" />
                </SelectTrigger>
                <SelectContent>
                  {activePackages.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.display_name} · ৳{p.monthly_price}/mo
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>Notes</Label>
              <Input placeholder="Optional" value={form.notes}
                onChange={(e) => setForm(f => ({ ...f, notes: e.target.value }))} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>Cancel</Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              {editId ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={!!deleteId} onOpenChange={(v) => !v && setDeleteId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete mapping?</AlertDialogTitle>
            <AlertDialogDescription>
              Future syncs will no longer auto-assign subscriptions for this profile.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive hover:bg-destructive/90"
              onClick={() => deleteId && deleteMutation.mutate(deleteId)}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
