"use client";

import React, { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus, Search, Pencil, Trash2, Eye, Loader2, Receipt,
  ChevronLeft, ChevronRight, SlidersHorizontal, X,
  Banknote, TrendingUp, Calendar, BarChart3,
} from "lucide-react";
import { expensesApi, expenseCategoriesApi } from "@/lib/api";
import {
  Expense, ExpenseCategory, ExpenseSummary, PaginatedResponse,
} from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/components/ui/use-toast";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  Sheet, SheetContent, SheetHeader, SheetTitle,
} from "@/components/ui/sheet";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";

const PAYMENT_METHODS = [
  { value: "cash", label: "Cash" },
  { value: "bank", label: "Bank Transfer" },
  { value: "mobile", label: "Mobile Banking" },
  { value: "cheque", label: "Cheque" },
  { value: "card", label: "Card" },
  { value: "other", label: "Other" },
];

const PAYMENT_BADGE: Record<string, string> = {
  cash: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300",
  bank: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300",
  mobile: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300",
  cheque: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300",
  card: "bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300",
  other: "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300",
};

const EMPTY_FORM = {
  expense_date: new Date().toISOString().split("T")[0],
  category_id: "",
  amount: "",
  payment_method: "cash",
  vendor: "",
  reference_number: "",
  description: "",
};

function fmt(n: number) {
  return n.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function SummaryCard({
  icon: Icon, label, value, color,
}: { icon: React.ElementType; label: string; value: number; color: string }) {
  return (
    <div className="bg-card border border-border rounded-xl p-4 flex items-center gap-4">
      <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${color}`}>
        <Icon className="w-5 h-5 text-white" />
      </div>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-lg font-bold text-foreground">৳{fmt(value)}</p>
      </div>
    </div>
  );
}

export default function ExpensesPage() {
  const { toast } = useToast();
  const qc = useQueryClient();

  // List state
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("all");
  const [paymentFilter, setPaymentFilter] = useState("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [showFilters, setShowFilters] = useState(false);
  const [sortBy, setSortBy] = useState("date");
  const [sortDir, setSortDir] = useState("desc");

  // Dialog state
  const [formOpen, setFormOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [editId, setEditId] = useState<string | null>(null);
  const [detailExpense, setDetailExpense] = useState<Expense | null>(null);
  const [form, setForm] = useState(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [saveAndNew, setSaveAndNew] = useState(false);

  const listParams = {
    page, page_size: 25,
    search: search || undefined,
    category_id: categoryFilter !== "all" ? categoryFilter : undefined,
    payment_method: paymentFilter !== "all" ? paymentFilter : undefined,
    date_from: dateFrom || undefined,
    date_to: dateTo || undefined,
    sort_by: sortBy,
    sort_dir: sortDir,
  };

  const { data, isLoading } = useQuery({
    queryKey: ["expenses", listParams],
    queryFn: () => expensesApi.list(listParams).then((r) => r.data as PaginatedResponse<Expense> & { data: Expense[] }),
  });

  const { data: summaryData } = useQuery({
    queryKey: ["expenses-summary"],
    queryFn: () => expensesApi.summary().then((r) => r.data.data as ExpenseSummary),
  });

  const { data: catsData } = useQuery({
    queryKey: ["expense-categories", "all"],
    queryFn: () => expenseCategoriesApi.list("active").then((r) => r.data.data as ExpenseCategory[]),
  });
  const categories = catsData ?? [];

  const deleteMutation = useMutation({
    mutationFn: (id: string) => expensesApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["expenses"] });
      qc.invalidateQueries({ queryKey: ["expenses-summary"] });
      toast({ title: "Expense deleted" });
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
    setFormOpen(true);
  }

  function openEdit(exp: Expense) {
    setForm({
      expense_date: exp.expense_date.split("T")[0],
      category_id: exp.category_id,
      amount: String(exp.amount),
      payment_method: exp.payment_method,
      vendor: exp.vendor ?? "",
      reference_number: exp.reference_number ?? "",
      description: exp.description ?? "",
    });
    setEditId(exp.id);
    setFormOpen(true);
  }

  function openDetail(exp: Expense) {
    setDetailExpense(exp);
    setDetailOpen(true);
  }

  const handleSave = useCallback(async (andNew: boolean) => {
    if (!form.category_id) {
      toast({ title: "Category is required", variant: "destructive" }); return;
    }
    const amount = parseFloat(form.amount);
    if (!form.amount || isNaN(amount) || amount <= 0) {
      toast({ title: "Amount must be greater than zero", variant: "destructive" }); return;
    }
    if (!form.expense_date) {
      toast({ title: "Expense date is required", variant: "destructive" }); return;
    }
    setSaving(true);
    setSaveAndNew(andNew);
    try {
      const payload = {
        expense_date: form.expense_date,
        category_id: form.category_id,
        amount,
        payment_method: form.payment_method,
        vendor: form.vendor,
        reference_number: form.reference_number,
        description: form.description,
      };
      if (editId) {
        await expensesApi.update(editId, payload);
        toast({ title: `Expense updated` });
      } else {
        await expensesApi.create(payload);
        toast({ title: "Expense recorded" });
      }
      qc.invalidateQueries({ queryKey: ["expenses"] });
      qc.invalidateQueries({ queryKey: ["expenses-summary"] });
      if (andNew) {
        setForm(EMPTY_FORM);
        setEditId(null);
      } else {
        setFormOpen(false);
      }
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Save failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    } finally {
      setSaving(false);
    }
  }, [form, editId, toast, qc]);

  function toggleSort(col: string) {
    if (sortBy === col) {
      setSortDir((d) => d === "desc" ? "asc" : "desc");
    } else {
      setSortBy(col);
      setSortDir("desc");
    }
    setPage(1);
  }

  const SortBtn = ({ col, label }: { col: string; label: string }) => (
    <button
      onClick={() => toggleSort(col)}
      className="flex items-center gap-1 hover:text-foreground transition-colors"
    >
      {label}
      {sortBy === col && <span className="text-xs">{sortDir === "desc" ? "↓" : "↑"}</span>}
    </button>
  );

  const summary = summaryData;
  const expenses = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = data?.total_pages ?? 1;

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Expenses" subtitle="Business expense records" />

      <div className="flex-1 overflow-auto p-6 space-y-5">

        {/* Summary cards */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <SummaryCard icon={Calendar} label="Today" value={summary?.today_total ?? 0} color="bg-blue-500" />
          <SummaryCard icon={TrendingUp} label="This Month" value={summary?.month_total ?? 0} color="bg-emerald-500" />
          <SummaryCard icon={BarChart3} label="This Year" value={summary?.year_total ?? 0} color="bg-violet-500" />
          <SummaryCard icon={Banknote} label="All Time" value={summary?.all_time_total ?? 0} color="bg-orange-500" />
        </div>

        {/* Category totals (this month) */}
        {summary && summary.category_totals && summary.category_totals.length > 0 && (
          <div className="bg-card border border-border rounded-xl p-4">
            <p className="text-sm font-semibold text-foreground mb-3">This Month by Category</p>
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2">
              {summary.category_totals.map((ct) => (
                <div key={ct.category_id} className="flex items-center justify-between bg-muted/40 rounded-lg px-3 py-2">
                  <span className="text-xs text-muted-foreground truncate">{ct.category_name}</span>
                  <span className="text-xs font-semibold text-foreground ml-2">৳{fmt(ct.total)}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Toolbar */}
        <div className="flex flex-col sm:flex-row gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search by expense no., vendor, description…"
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              className="pl-9"
            />
          </div>
          <Button variant="outline" onClick={() => setShowFilters((v) => !v)} className="gap-2">
            <SlidersHorizontal className="w-4 h-4" />
            Filters
            {(categoryFilter !== "all" || paymentFilter !== "all" || dateFrom || dateTo) && (
              <Badge className="ml-1 h-4 px-1 text-xs bg-blue-600 text-white">
                {[categoryFilter !== "all", paymentFilter !== "all", dateFrom, dateTo].filter(Boolean).length}
              </Badge>
            )}
          </Button>
          <Button onClick={openCreate} className="gap-2 bg-blue-600 hover:bg-blue-700 text-white">
            <Plus className="w-4 h-4" /> Add Expense
          </Button>
        </div>

        {/* Filter panel */}
        {showFilters && (
          <div className="bg-muted/30 border border-border rounded-xl p-4 grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="space-y-1.5">
              <Label className="text-xs">Category</Label>
              <Select value={categoryFilter} onValueChange={(v) => { setCategoryFilter(v); setPage(1); }}>
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue placeholder="All" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Categories</SelectItem>
                  {categories.map((c) => (
                    <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Payment Method</Label>
              <Select value={paymentFilter} onValueChange={(v) => { setPaymentFilter(v); setPage(1); }}>
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue placeholder="All" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Methods</SelectItem>
                  {PAYMENT_METHODS.map((m) => (
                    <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Date From</Label>
              <Input type="date" className="h-8 text-xs" value={dateFrom}
                onChange={(e) => { setDateFrom(e.target.value); setPage(1); }} />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Date To</Label>
              <Input type="date" className="h-8 text-xs" value={dateTo}
                onChange={(e) => { setDateTo(e.target.value); setPage(1); }} />
            </div>
            <div className="col-span-full flex justify-end">
              <Button variant="ghost" size="sm" className="gap-1 text-xs" onClick={() => {
                setCategoryFilter("all"); setPaymentFilter("all"); setDateFrom(""); setDateTo(""); setPage(1);
              }}>
                <X className="w-3 h-3" /> Clear Filters
              </Button>
            </div>
          </div>
        )}

        {/* Table */}
        <div className="bg-card border border-border rounded-xl overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/30">
                <TableHead className="text-xs font-semibold">Expense No.</TableHead>
                <TableHead className="text-xs font-semibold">
                  <SortBtn col="date" label="Date" />
                </TableHead>
                <TableHead className="text-xs font-semibold">Category</TableHead>
                <TableHead className="text-xs font-semibold">
                  <SortBtn col="vendor" label="Vendor" />
                </TableHead>
                <TableHead className="text-xs font-semibold text-right">
                  <SortBtn col="amount" label="Amount" />
                </TableHead>
                <TableHead className="text-xs font-semibold">Payment</TableHead>
                <TableHead className="text-xs font-semibold">Entered By</TableHead>
                <TableHead className="text-xs font-semibold">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 8 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : expenses.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-12">
                    <Receipt className="w-10 h-10 text-muted-foreground mx-auto mb-2 opacity-40" />
                    <p className="text-muted-foreground text-sm">No expenses found</p>
                  </TableCell>
                </TableRow>
              ) : (
                expenses.map((exp) => (
                  <TableRow key={exp.id} className="hover:bg-muted/20 cursor-pointer"
                    onClick={() => openDetail(exp)}>
                    <TableCell className="font-mono text-xs text-blue-600 dark:text-blue-400 font-medium">
                      {exp.expense_number}
                    </TableCell>
                    <TableCell className="text-xs whitespace-nowrap">
                      {new Date(exp.expense_date).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-xs">
                      {exp.category?.name ?? "—"}
                    </TableCell>
                    <TableCell className="text-xs max-w-[120px] truncate">
                      {exp.vendor || <span className="text-muted-foreground">—</span>}
                    </TableCell>
                    <TableCell className="text-right font-semibold text-sm">
                      ৳{fmt(exp.amount)}
                    </TableCell>
                    <TableCell>
                      <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${PAYMENT_BADGE[exp.payment_method] ?? PAYMENT_BADGE.other}`}>
                        {PAYMENT_METHODS.find((m) => m.value === exp.payment_method)?.label ?? exp.payment_method}
                      </span>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {exp.created_by?.full_name ?? "—"}
                    </TableCell>
                    <TableCell onClick={(e) => e.stopPropagation()}>
                      <div className="flex items-center gap-1">
                        <Button variant="ghost" size="icon" className="h-7 w-7"
                          onClick={() => openDetail(exp)}>
                          <Eye className="w-3.5 h-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-7 w-7"
                          onClick={() => openEdit(exp)}>
                          <Pencil className="w-3.5 h-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" className="h-7 w-7 text-destructive hover:text-destructive"
                          onClick={() => setDeleteId(exp.id)}>
                          <Trash2 className="w-3.5 h-3.5" />
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
            <span>{total} expense{total !== 1 ? "s" : ""}</span>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}>
                <ChevronLeft className="w-4 h-4" />
              </Button>
              <span>Page {page} of {totalPages}</span>
              <Button variant="outline" size="sm" disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}>
                <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Add / Edit Dialog */}
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editId ? "Edit Expense" : "Add Expense"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label>Expense Date <span className="text-destructive">*</span></Label>
                <Input type="date" value={form.expense_date}
                  onChange={(e) => setForm((f) => ({ ...f, expense_date: e.target.value }))} />
              </div>
              <div className="space-y-1.5">
                <Label>Amount (৳) <span className="text-destructive">*</span></Label>
                <Input type="number" min="0.01" step="0.01" placeholder="0.00"
                  value={form.amount}
                  onChange={(e) => setForm((f) => ({ ...f, amount: e.target.value }))} />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label>Category <span className="text-destructive">*</span></Label>
              <Select value={form.category_id}
                onValueChange={(v) => setForm((f) => ({ ...f, category_id: v }))}>
                <SelectTrigger>
                  <SelectValue placeholder="Select category…" />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((c) => (
                    <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Payment Method</Label>
              <Select value={form.payment_method}
                onValueChange={(v) => setForm((f) => ({ ...f, payment_method: v }))}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PAYMENT_METHODS.map((m) => (
                    <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label>Vendor / Supplier</Label>
                <Input placeholder="Vendor name"
                  value={form.vendor}
                  onChange={(e) => setForm((f) => ({ ...f, vendor: e.target.value }))} />
              </div>
              <div className="space-y-1.5">
                <Label>Reference Number</Label>
                <Input placeholder="Invoice / receipt no."
                  value={form.reference_number}
                  onChange={(e) => setForm((f) => ({ ...f, reference_number: e.target.value }))} />
              </div>
            </div>
            <div className="space-y-1.5">
              <Label>Description</Label>
              <Textarea placeholder="Additional notes…" rows={2}
                value={form.description}
                onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setForm((f) => ({ ...f, description: e.target.value }))} />
            </div>
          </div>
          <DialogFooter className="gap-2">
            <Button variant="outline" onClick={() => setFormOpen(false)}>Cancel</Button>
            {!editId && (
              <Button variant="outline" onClick={() => handleSave(true)} disabled={saving}>
                {saving && saveAndNew ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : null}
                Save & New
              </Button>
            )}
            <Button onClick={() => handleSave(false)} disabled={saving}
              className="bg-blue-600 hover:bg-blue-700 text-white">
              {saving && !saveAndNew ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : null}
              Save
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Detail Sheet */}
      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent className="w-[420px] sm:w-[500px] overflow-y-auto">
          <SheetHeader>
            <SheetTitle>Expense Details</SheetTitle>
          </SheetHeader>
          {detailExpense && (
            <div className="mt-6 space-y-5">
              <div className="bg-muted/30 rounded-xl p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-sm text-blue-600 dark:text-blue-400 font-medium">
                    {detailExpense.expense_number}
                  </span>
                  <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${PAYMENT_BADGE[detailExpense.payment_method] ?? PAYMENT_BADGE.other}`}>
                    {PAYMENT_METHODS.find((m) => m.value === detailExpense.payment_method)?.label ?? detailExpense.payment_method}
                  </span>
                </div>
                <div className="text-3xl font-bold text-foreground">৳{fmt(detailExpense.amount)}</div>
                <div className="text-sm text-muted-foreground">
                  {new Date(detailExpense.expense_date).toLocaleDateString("en-US", { year: "numeric", month: "long", day: "numeric" })}
                </div>
              </div>

              <Section title="General Information">
                <Row label="Category" value={detailExpense.category?.name ?? "—"} />
                <Row label="Payment Method" value={PAYMENT_METHODS.find((m) => m.value === detailExpense.payment_method)?.label ?? detailExpense.payment_method} />
                {detailExpense.description && (
                  <Row label="Description" value={detailExpense.description} />
                )}
              </Section>

              <Section title="Vendor Information">
                <Row label="Vendor / Supplier" value={detailExpense.vendor || "—"} />
                <Row label="Reference Number" value={detailExpense.reference_number || "—"} />
              </Section>

              <Section title="Audit Information">
                <Row label="Entered By" value={detailExpense.created_by?.full_name ?? "—"} />
                <Row label="Created At" value={new Date(detailExpense.created_at).toLocaleString()} />
                {detailExpense.updated_by && (
                  <Row label="Updated By" value={detailExpense.updated_by.full_name} />
                )}
                <Row label="Updated At" value={new Date(detailExpense.updated_at).toLocaleString()} />
              </Section>

              <div className="flex gap-2 pt-2">
                <Button variant="outline" className="flex-1" onClick={() => {
                  setDetailOpen(false);
                  openEdit(detailExpense);
                }}>
                  <Pencil className="w-4 h-4 mr-2" /> Edit
                </Button>
                <Button variant="destructive" className="flex-1" onClick={() => {
                  setDetailOpen(false);
                  setDeleteId(detailExpense.id);
                }}>
                  <Trash2 className="w-4 h-4 mr-2" /> Delete
                </Button>
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>

      {/* Delete confirm */}
      <AlertDialog open={!!deleteId} onOpenChange={() => setDeleteId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Expense</AlertDialogTitle>
            <AlertDialogDescription>
              This expense will be soft-deleted and excluded from all reports. This action can be reviewed in audit logs.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => deleteId && deleteMutation.mutate(deleteId)}>
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">{title}</p>
      <div className="bg-muted/20 rounded-lg divide-y divide-border">{children}</div>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start justify-between px-3 py-2 gap-2">
      <span className="text-xs text-muted-foreground shrink-0">{label}</span>
      <span className="text-xs font-medium text-foreground text-right">{value}</span>
    </div>
  );
}
