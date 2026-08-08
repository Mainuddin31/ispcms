"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Pencil, Trash2, Loader2, Search, Package } from "lucide-react";
import { packagesApi } from "@/lib/api";
import { Package as PkgType, PaginatedResponse } from "@/types";
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

const EMPTY_FORM = {
  package_name: "",
  display_name: "",
  speed: "",
  monthly_price: "",
  vat_percent: "0",
  installation_charge: "0",
  description: "",
  status: "active",
};

export default function PackagesPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [page, setPage] = useState(1);
  const [form, setForm] = useState(EMPTY_FORM);
  const [editId, setEditId] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["packages", page, search, statusFilter],
    queryFn: () =>
      packagesApi
        .list({ page, page_size: 20, search: search || undefined, status: statusFilter === "all" ? undefined : statusFilter })
        .then((r) => r.data as PaginatedResponse<PkgType> & { data: PkgType[] }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => packagesApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["packages"] });
      toast({ title: "Package deleted" });
      setDeleteId(null);
    },
    onError: (e: unknown) => {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Delete failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    },
  });

  function openCreate() {
    setForm(EMPTY_FORM);
    setEditId(null);
    setDialogOpen(true);
  }

  function openEdit(pkg: PkgType) {
    setForm({
      package_name: pkg.package_name,
      display_name: pkg.display_name,
      speed: pkg.speed ?? "",
      monthly_price: String(pkg.monthly_price),
      vat_percent: String(pkg.vat_percent),
      installation_charge: String(pkg.installation_charge),
      description: pkg.description ?? "",
      status: pkg.status,
    });
    setEditId(pkg.id);
    setDialogOpen(true);
  }

  async function handleSave() {
    if (!form.package_name || !form.display_name || !form.monthly_price) {
      toast({ title: "Validation", description: "Package name, display name and price are required", variant: "destructive" });
      return;
    }
    setSaving(true);
    const payload = {
      ...form,
      monthly_price: parseFloat(form.monthly_price),
      vat_percent: parseFloat(form.vat_percent),
      installation_charge: parseFloat(form.installation_charge),
    };
    try {
      if (editId) {
        await packagesApi.update(editId, payload);
        toast({ title: "Package updated" });
      } else {
        await packagesApi.create(payload);
        toast({ title: "Package created" });
      }
      qc.invalidateQueries({ queryKey: ["packages"] });
      setDialogOpen(false);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Save failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    } finally {
      setSaving(false);
    }
  }

  const packages = data?.data ?? [];
  const totalPages = data?.total_pages ?? 1;

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Packages" subtitle="Manage billing packages" />

      <div className="flex-1 overflow-auto p-6 space-y-4">
        {/* Toolbar */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search packages…"
              className="pl-9"
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            />
          </div>
          <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1); }}>
            <SelectTrigger className="w-36">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Status</SelectItem>
              <SelectItem value="active">Active</SelectItem>
              <SelectItem value="inactive">Inactive</SelectItem>
            </SelectContent>
          </Select>
          <Button onClick={openCreate}>
            <Plus className="h-4 w-4 mr-2" /> Add Package
          </Button>
        </div>

        {/* Table */}
        <div className="rounded-lg border overflow-x-auto">
          <Table className="min-w-[700px]">
            <TableHeader>
              <TableRow>
                <TableHead>Package Name</TableHead>
                <TableHead>Display Name</TableHead>
                <TableHead>Speed</TableHead>
                <TableHead className="text-right">Price/mo</TableHead>
                <TableHead className="text-right">VAT %</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-24">Actions</TableHead>
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
              ) : packages.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-12 text-muted-foreground">
                    <Package className="mx-auto h-8 w-8 mb-2 opacity-40" />
                    No packages found
                  </TableCell>
                </TableRow>
              ) : (
                packages.map((pkg) => (
                  <TableRow key={pkg.id}>
                    <TableCell className="font-mono text-xs">{pkg.package_name}</TableCell>
                    <TableCell className="font-medium">{pkg.display_name}</TableCell>
                    <TableCell className="text-muted-foreground">{pkg.speed || "—"}</TableCell>
                    <TableCell className="text-right font-mono">৳{pkg.monthly_price.toFixed(2)}</TableCell>
                    <TableCell className="text-right">{pkg.vat_percent}%</TableCell>
                    <TableCell>
                      <Badge variant={pkg.status === "active" ? "default" : "secondary"}>
                        {pkg.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button size="icon" variant="ghost" onClick={() => openEdit(pkg)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button size="icon" variant="ghost" className="text-destructive" onClick={() => setDeleteId(pkg.id)}>
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

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>Page {page} of {totalPages} · {data?.total} total</span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>Previous</Button>
              <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>Next</Button>
            </div>
          </div>
        )}
      </div>

      {/* Create / Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editId ? "Edit Package" : "Add Package"}</DialogTitle>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-4 py-2">
            <div className="col-span-2 space-y-1">
              <Label>Package Name <span className="text-destructive">*</span></Label>
              <Input placeholder="e.g. BASIC_10MBPS" value={form.package_name}
                onChange={(e) => setForm(f => ({ ...f, package_name: e.target.value }))} />
            </div>
            <div className="col-span-2 space-y-1">
              <Label>Display Name <span className="text-destructive">*</span></Label>
              <Input placeholder="e.g. Basic 10 Mbps" value={form.display_name}
                onChange={(e) => setForm(f => ({ ...f, display_name: e.target.value }))} />
            </div>
            <div className="space-y-1">
              <Label>Speed</Label>
              <Input placeholder="e.g. 10 Mbps" value={form.speed}
                onChange={(e) => setForm(f => ({ ...f, speed: e.target.value }))} />
            </div>
            <div className="space-y-1">
              <Label>Monthly Price (৳) <span className="text-destructive">*</span></Label>
              <Input type="number" min="0" step="0.01" value={form.monthly_price}
                onChange={(e) => setForm(f => ({ ...f, monthly_price: e.target.value }))} />
            </div>
            <div className="space-y-1">
              <Label>VAT %</Label>
              <Input type="number" min="0" step="0.01" value={form.vat_percent}
                onChange={(e) => setForm(f => ({ ...f, vat_percent: e.target.value }))} />
            </div>
            <div className="space-y-1">
              <Label>Installation Charge (৳)</Label>
              <Input type="number" min="0" step="0.01" value={form.installation_charge}
                onChange={(e) => setForm(f => ({ ...f, installation_charge: e.target.value }))} />
            </div>
            <div className="col-span-2 space-y-1">
              <Label>Status</Label>
              <Select value={form.status} onValueChange={(v) => setForm(f => ({ ...f, status: v }))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="inactive">Inactive</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="col-span-2 space-y-1">
              <Label>Description</Label>
              <Input placeholder="Optional notes" value={form.description}
                onChange={(e) => setForm(f => ({ ...f, description: e.target.value }))} />
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

      {/* Delete Confirm */}
      <AlertDialog open={!!deleteId} onOpenChange={(v) => !v && setDeleteId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete package?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. Existing subscriptions and bills will remain.
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
