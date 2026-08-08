"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FileText, Loader2, Search, Zap, ChevronLeft, ChevronRight, History } from "lucide-react";
import { billsApi, paymentHistoryApi } from "@/lib/api";
import { MonthlyBill, PaginatedResponse, PaymentRecord } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Label } from "@/components/ui/label";
import { useToast } from "@/components/ui/use-toast";
import { formatDate } from "@/lib/utils";
import { cn } from "@/lib/utils";

const STATUS_COLOR: Record<string, string> = {
  pending: "bg-slate-100 text-slate-700",
  due: "bg-red-100 text-red-700",
  partial: "bg-amber-100 text-amber-700",
  paid: "bg-green-100 text-green-700",
  cancelled: "bg-zinc-100 text-zinc-500",
};

const MONTHS = [
  "January","February","March","April","May","June",
  "July","August","September","October","November","December",
];

export default function BillsPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const now = new Date();
  const currentMonth = now.getMonth() + 1;
  const currentYear = now.getFullYear();
  const currentMonthLabel = `${MONTHS[currentMonth - 1]} ${currentYear}`;

  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [monthFilter, setMonthFilter] = useState(String(currentMonth));
  const [yearFilter, setYearFilter] = useState(String(currentYear));
  const [page, setPage] = useState(1);
  const [confirmGenerateOpen, setConfirmGenerateOpen] = useState(false);
  const [generating, setGenerating] = useState(false);

  // Record Payment dialog
  const [payDialogOpen, setPayDialogOpen] = useState(false);
  const [selectedBill, setSelectedBill] = useState<MonthlyBill | null>(null);
  const [paidAmount, setPaidAmount] = useState("");
  const [payNotes, setPayNotes] = useState("");
  const [paymentMethod, setPaymentMethod] = useState("cash");
  const [receiptNumber, setReceiptNumber] = useState("");
  const [paying, setPaying] = useState(false);

  // Payment history dialog
  const [historyDialogOpen, setHistoryDialogOpen] = useState(false);
  const [historyAccount, setHistoryAccount] = useState<{ id: string; username: string } | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["bills", page, search, statusFilter, monthFilter, yearFilter],
    queryFn: () =>
      billsApi.list({
        page, page_size: 25,
        search: search || undefined,
        status: statusFilter === "all" ? undefined : statusFilter,
        month: monthFilter ? parseInt(monthFilter) : undefined,
        year: yearFilter ? parseInt(yearFilter) : undefined,
      }).then((r) => r.data as PaginatedResponse<MonthlyBill> & { data: MonthlyBill[] }),
  });

  const { data: historyData, isLoading: historyLoading } = useQuery({
    queryKey: ["payment-history", historyAccount?.id],
    queryFn: () =>
      paymentHistoryApi
        .listByAccount(historyAccount!.id)
        .then((r) => r.data.data as PaymentRecord[]),
    enabled: !!historyAccount && historyDialogOpen,
  });

  async function handleGenerate() {
    setGenerating(true);
    try {
      const res = await billsApi.generate(currentMonth, currentYear);
      const genLog = res.data.data;
      toast({
        title: "Bills Generated",
        description: `${genLog.bills_generated} bills created, ${genLog.bills_skipped} skipped.`,
      });
      qc.invalidateQueries({ queryKey: ["bills"] });
      setConfirmGenerateOpen(false);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Generation failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    } finally {
      setGenerating(false);
    }
  }

  function openPayDialog(bill: MonthlyBill) {
    setSelectedBill(bill);
    setPaidAmount(String(bill.due_amount));
    setPayNotes("");
    setPaymentMethod("cash");
    setReceiptNumber("");
    setPayDialogOpen(true);
  }

  function openHistory(bill: MonthlyBill) {
    if (!bill.internet_account) return;
    setHistoryAccount({ id: bill.internet_account_id, username: bill.internet_account.username });
    setHistoryDialogOpen(true);
  }

  async function handlePayment() {
    if (!selectedBill) return;
    setPaying(true);
    try {
      await billsApi.updateStatus(selectedBill.id, {
        paid_amount: selectedBill.paid_amount + parseFloat(paidAmount),
        notes: payNotes,
        payment_method: paymentMethod,
        receipt_number: receiptNumber,
      });
      toast({ title: "Payment recorded" });
      qc.invalidateQueries({ queryKey: ["bills"] });
      setPayDialogOpen(false);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Update failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    } finally {
      setPaying(false);
    }
  }

  const bills = data?.data ?? [];
  const totalPages = data?.total_pages ?? 1;
  const years = [String(currentYear), String(currentYear - 1), String(currentYear - 2)];
  const history = historyData ?? [];

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Bills" subtitle="Monthly billing records" />

      <div className="flex-1 overflow-auto p-6 space-y-4">
        {/* Toolbar */}
        <div className="flex items-center gap-2 flex-wrap">
          <div className="relative flex-1 min-w-[180px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input placeholder="Search username / bill no…" className="pl-9" value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }} />
          </div>

          <Select value={monthFilter} onValueChange={(v) => { setMonthFilter(v); setPage(1); }}>
            <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="">All Months</SelectItem>
              {MONTHS.map((m, i) => (
                <SelectItem key={i} value={String(i + 1)}>{m}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={yearFilter} onValueChange={(v) => { setYearFilter(v); setPage(1); }}>
            <SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="">All Years</SelectItem>
              {years.map((y) => <SelectItem key={y} value={y}>{y}</SelectItem>)}
            </SelectContent>
          </Select>

          <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1); }}>
            <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Status</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="due">Due</SelectItem>
              <SelectItem value="partial">Partial</SelectItem>
              <SelectItem value="paid">Paid</SelectItem>
              <SelectItem value="cancelled">Cancelled</SelectItem>
            </SelectContent>
          </Select>

          <Button onClick={() => setConfirmGenerateOpen(true)}>
            <Zap className="h-4 w-4 mr-2" />
            Generate Bills — {currentMonthLabel}
          </Button>
        </div>

        {/* Table */}
        <div className="rounded-lg border overflow-x-auto">
          <Table className="min-w-[860px]">
            <TableHeader>
              <TableRow>
                <TableHead>Bill #</TableHead>
                <TableHead>Customer</TableHead>
                <TableHead>Package</TableHead>
                <TableHead>Period</TableHead>
                <TableHead className="text-right">Total</TableHead>
                <TableHead className="text-right">Due</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Due Date</TableHead>
                <TableHead className="w-44">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 9 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : bills.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9} className="text-center py-12 text-muted-foreground">
                    <FileText className="mx-auto h-8 w-8 mb-2 opacity-40" />
                    No bills found
                  </TableCell>
                </TableRow>
              ) : (
                bills.map((b) => (
                  <TableRow key={b.id}>
                    <TableCell className="font-mono text-xs">{b.bill_number}</TableCell>
                    <TableCell>
                      <button
                        className="font-medium hover:underline text-primary text-left"
                        onClick={() => openHistory(b)}
                        title="View payment history"
                      >
                        {b.internet_account?.username ?? "—"}
                      </button>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{b.package?.display_name ?? "—"}</TableCell>
                    <TableCell className="text-sm">{MONTHS[b.billing_month - 1]} {b.billing_year}</TableCell>
                    <TableCell className="text-right font-mono">৳{b.total_amount.toFixed(2)}</TableCell>
                    <TableCell
                      className="text-right font-mono font-medium"
                      style={{ color: b.due_amount > 0 ? "rgb(220 38 38)" : undefined }}
                    >
                      ৳{b.due_amount.toFixed(2)}
                    </TableCell>
                    <TableCell>
                      <span className={cn("px-2 py-0.5 rounded text-xs font-medium", STATUS_COLOR[b.status] ?? "bg-slate-100")}>
                        {b.status}
                      </span>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {b.due_date ? formatDate(b.due_date) : "—"}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        {b.status !== "paid" && b.status !== "cancelled" && (
                          <Button size="sm" variant="outline" onClick={() => openPayDialog(b)}>
                            Record Payment
                          </Button>
                        )}
                        <Button size="sm" variant="ghost" onClick={() => openHistory(b)} title="Payment history">
                          <History className="h-4 w-4" />
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
            <span>Page {page} of {totalPages} · {data?.total} bills</span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Generate Confirm Dialog */}
      <AlertDialog open={confirmGenerateOpen} onOpenChange={setConfirmGenerateOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Generate Bills for {currentMonthLabel}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will create one bill for every active customer who has a subscription.
              Customers already billed this month are automatically skipped.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={generating}>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleGenerate} disabled={generating}>
              {generating && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Generate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Record Payment Dialog */}
      <Dialog open={payDialogOpen} onOpenChange={setPayDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Record Payment</DialogTitle>
          </DialogHeader>
          {selectedBill && (
            <div className="space-y-4">
              <div className="rounded bg-muted p-3 text-sm space-y-1">
                <p><span className="text-muted-foreground">Bill:</span> {selectedBill.bill_number}</p>
                <p><span className="text-muted-foreground">Customer:</span> {selectedBill.internet_account?.username}</p>
                <p><span className="text-muted-foreground">Total:</span> ৳{selectedBill.total_amount.toFixed(2)}</p>
                <p><span className="text-muted-foreground">Already Paid:</span> ৳{selectedBill.paid_amount.toFixed(2)}</p>
                <p><span className="text-muted-foreground">Due:</span> ৳{selectedBill.due_amount.toFixed(2)}</p>
              </div>
              <div className="space-y-1">
                <Label>Amount Received Now (৳)</Label>
                <Input type="number" min="0" step="0.01" value={paidAmount}
                  onChange={(e) => setPaidAmount(e.target.value)} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label>Payment Method</Label>
                  <Select value={paymentMethod} onValueChange={setPaymentMethod}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="cash">Cash</SelectItem>
                      <SelectItem value="bkash">bKash</SelectItem>
                      <SelectItem value="bank">Bank Transfer</SelectItem>
                      <SelectItem value="card">Card</SelectItem>
                      <SelectItem value="other">Other</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1">
                  <Label>Receipt Number</Label>
                  <Input placeholder="Optional" value={receiptNumber}
                    onChange={(e) => setReceiptNumber(e.target.value)} />
                </div>
              </div>
              <div className="space-y-1">
                <Label>Notes</Label>
                <Input placeholder="Optional" value={payNotes}
                  onChange={(e) => setPayNotes(e.target.value)} />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setPayDialogOpen(false)}>Cancel</Button>
            <Button onClick={handlePayment} disabled={paying}>
              {paying && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Save Payment
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Payment History Dialog */}
      <Dialog open={historyDialogOpen} onOpenChange={setHistoryDialogOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <History className="h-5 w-5" />
              Payment History — {historyAccount?.username}
            </DialogTitle>
          </DialogHeader>

          {historyLoading ? (
            <div className="space-y-2 py-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full" />
              ))}
            </div>
          ) : history.length === 0 ? (
            <div className="text-center py-10 text-muted-foreground">
              <History className="mx-auto h-8 w-8 mb-2 opacity-40" />
              No payment records found
            </div>
          ) : (
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Month</TableHead>
                    <TableHead className="text-right">Bill Total</TableHead>
                    <TableHead className="text-right">Debit (Due)</TableHead>
                    <TableHead className="text-right">Credit (Paid)</TableHead>
                    <TableHead>Method</TableHead>
                    <TableHead>Receipt #</TableHead>
                    <TableHead>Comments</TableHead>
                    <TableHead>Received By</TableHead>
                    <TableHead>Date</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {history.map((rec) => (
                    <TableRow key={rec.id}>
                      <TableCell className="text-sm whitespace-nowrap">
                        {rec.bill
                          ? `${MONTHS[rec.bill.billing_month - 1]} ${rec.bill.billing_year}`
                          : "—"}
                      </TableCell>
                      <TableCell className="text-right font-mono text-sm">
                        {rec.bill ? `৳${rec.bill.total_amount.toFixed(2)}` : "—"}
                      </TableCell>
                      <TableCell className="text-right font-mono text-sm text-red-600">
                        {rec.bill ? `৳${rec.bill.due_amount.toFixed(2)}` : "—"}
                      </TableCell>
                      <TableCell className="text-right font-mono text-sm text-green-600 font-medium">
                        ৳{rec.amount.toFixed(2)}
                      </TableCell>
                      <TableCell className="text-xs capitalize">
                        {rec.payment_method || "cash"}
                      </TableCell>
                      <TableCell className="text-xs font-mono">
                        {rec.receipt_number || "—"}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground max-w-[120px] truncate">
                        {rec.notes || "—"}
                      </TableCell>
                      <TableCell className="text-sm whitespace-nowrap">
                        {rec.received_by?.full_name ?? rec.received_by?.username ?? "—"}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                        {formatDate(rec.paid_at)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setHistoryDialogOpen(false)}>Close</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
