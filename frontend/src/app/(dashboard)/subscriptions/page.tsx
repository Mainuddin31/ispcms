"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { CreditCard, Plus, Loader2, Search, Zap, Info } from "lucide-react";
import { subscriptionsApi, packagesApi, internetAccountsApi } from "@/lib/api";
import { CustomerSubscription, PaginatedResponse } from "@/types";
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
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { useToast } from "@/components/ui/use-toast";
import { formatDate } from "@/lib/utils";

export default function SubscriptionsPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [page, setPage] = useState(1);
  const [activeOnly, setActiveOnly] = useState(true);
  const [search, setSearch] = useState("");
  const [manualDialogOpen, setManualDialogOpen] = useState(false);
  const [accountSearch, setAccountSearch] = useState("");
  const [selectedAccount, setSelectedAccount] = useState("");
  const [selectedPackage, setSelectedPackage] = useState("");
  const [saving, setSaving] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["subscriptions", page, activeOnly],
    queryFn: () =>
      subscriptionsApi
        .list({ page, page_size: 25, active_only: activeOnly })
        .then((r) => r.data as PaginatedResponse<CustomerSubscription> & { data: CustomerSubscription[] }),
  });

  const { data: packagesData } = useQuery({
    queryKey: ["packages-active"],
    queryFn: () => packagesApi.listActive().then((r) => r.data.data),
  });

  const { data: accountsData } = useQuery({
    queryKey: ["ia-list", accountSearch],
    queryFn: () =>
      internetAccountsApi
        .list({ page: 1, page_size: 30, search: accountSearch || undefined })
        .then((r) => r.data.data),
    enabled: manualDialogOpen,
  });

  // Auto-assign mutation
  const autoAssignMutation = useMutation({
    mutationFn: () => subscriptionsApi.autoAssign(),
    onSuccess: (res) => {
      const result = res.data.data as { assigned: number; skipped: number; total: number };
      qc.invalidateQueries({ queryKey: ["subscriptions"] });
      toast({
        title: "Auto-Assign Complete",
        description: `${result.assigned} subscriptions assigned, ${result.skipped} skipped (already subscribed or no profile mapping).`,
      });
    },
    onError: (e: unknown) => {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Auto-assign failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    },
  });

  async function handleManualAssign() {
    if (!selectedAccount || !selectedPackage) {
      toast({ title: "Select both account and package", variant: "destructive" });
      return;
    }
    setSaving(true);
    try {
      await subscriptionsApi.assign(selectedAccount, selectedPackage);
      toast({ title: "Subscription assigned" });
      qc.invalidateQueries({ queryKey: ["subscriptions"] });
      setManualDialogOpen(false);
      setSelectedAccount("");
      setSelectedPackage("");
      setAccountSearch("");
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Assignment failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    } finally {
      setSaving(false);
    }
  }

  const subs = data?.data ?? [];
  const totalPages = data?.total_pages ?? 1;
  const accounts = (accountsData as { id: string; username: string }[] | undefined) ?? [];
  const packages = (packagesData as { id: string; display_name: string; monthly_price: number }[] | undefined) ?? [];

  const filtered = search
    ? subs.filter(s => s.internet_account?.username?.toLowerCase().includes(search.toLowerCase()))
    : subs;

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Subscriptions" subtitle="Customer package subscriptions" />

      <div className="flex-1 overflow-auto p-6 space-y-4">
        {/* How auto-assign works */}
        <div className="rounded-lg border border-blue-200 bg-blue-50 dark:bg-blue-950/20 dark:border-blue-800 p-4 flex gap-3">
          <Info className="h-5 w-5 text-blue-500 shrink-0 mt-0.5" />
          <div className="text-sm text-blue-800 dark:text-blue-300 space-y-1">
            <p className="font-medium">How subscriptions work</p>
            <p>
              <strong>Auto (recommended):</strong> Set up Profile Mappings, then click
              <strong> Auto-Assign from Profiles</strong> — every active account whose MikroTik
              profile has a mapping but no subscription gets assigned automatically.
              Sync also triggers this automatically.
            </p>
            <p>
              <strong>Manual:</strong> Use <strong>Assign Package</strong> to override a specific account,
              or for accounts with no MikroTik profile.
            </p>
          </div>
        </div>

        {/* Toolbar */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[180px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Filter by username…"
              className="pl-9"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={() => setActiveOnly(true)}
              className={`px-3 py-1.5 text-sm rounded-md font-medium transition-colors ${activeOnly ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"}`}
            >Active</button>
            <button
              onClick={() => setActiveOnly(false)}
              className={`px-3 py-1.5 text-sm rounded-md font-medium transition-colors ${!activeOnly ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"}`}
            >All</button>
          </div>

          {/* Auto-assign button */}
          <Button
            variant="outline"
            onClick={() => autoAssignMutation.mutate()}
            disabled={autoAssignMutation.isPending}
          >
            {autoAssignMutation.isPending
              ? <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              : <Zap className="h-4 w-4 mr-2" />}
            Auto-Assign from Profiles
          </Button>

          {/* Manual assign button */}
          <Button onClick={() => setManualDialogOpen(true)}>
            <Plus className="h-4 w-4 mr-2" /> Assign Package
          </Button>
        </div>

        {/* Table */}
        <div className="rounded-lg border overflow-x-auto">
          <Table className="min-w-[700px]">
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Router</TableHead>
                <TableHead>Package</TableHead>
                <TableHead className="text-right">Price/mo</TableHead>
                <TableHead>Effective From</TableHead>
                <TableHead>Effective Until</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 6 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 7 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : filtered.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-12 text-muted-foreground">
                    <CreditCard className="mx-auto h-8 w-8 mb-2 opacity-40" />
                    No subscriptions found
                  </TableCell>
                </TableRow>
              ) : (
                filtered.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell className="font-medium">{s.internet_account?.username ?? "—"}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{s.internet_account?.router?.name ?? "—"}</TableCell>
                    <TableCell>{s.package?.display_name ?? "—"}</TableCell>
                    <TableCell className="text-right font-mono">৳{s.monthly_price.toFixed(2)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDate(s.effective_from)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{s.effective_until ? formatDate(s.effective_until) : "—"}</TableCell>
                    <TableCell>
                      <Badge variant={s.is_active ? "default" : "secondary"}>
                        {s.is_active ? "Active" : "Closed"}
                      </Badge>
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

      {/* Manual Assign Dialog */}
      <Dialog open={manualDialogOpen} onOpenChange={setManualDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Manually Assign Package</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Use this to override a specific account or assign a package to an account
            with no MikroTik profile.
          </p>
          <div className="space-y-4 py-2">
            <div className="space-y-1">
              <Label>Search Account</Label>
              <Input
                placeholder="Type username to filter…"
                value={accountSearch}
                onChange={(e) => setAccountSearch(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <Label>Internet Account <span className="text-destructive">*</span></Label>
              <Select value={selectedAccount} onValueChange={setSelectedAccount}>
                <SelectTrigger>
                  <SelectValue placeholder="Select account…" />
                </SelectTrigger>
                <SelectContent>
                  {accounts.map((a) => (
                    <SelectItem key={a.id} value={a.id}>{a.username}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>Package <span className="text-destructive">*</span></Label>
              <Select value={selectedPackage} onValueChange={setSelectedPackage}>
                <SelectTrigger>
                  <SelectValue placeholder="Select package…" />
                </SelectTrigger>
                <SelectContent>
                  {packages.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.display_name} · ৳{p.monthly_price}/mo
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setManualDialogOpen(false)}>Cancel</Button>
            <Button onClick={handleManualAssign} disabled={saving}>
              {saving && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Assign
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
