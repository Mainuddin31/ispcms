"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CalendarDays, Search, CheckCircle2, XCircle, RotateCcw,
  ChevronLeft, ChevronRight, Clock, User2, AlertCircle,
  ArrowLeft,
} from "lucide-react";
import Link from "next/link";
import { visitingApi, usersApi } from "@/lib/api";
import { Visit, User } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import { useToast } from "@/components/ui/use-toast";
import { cn } from "@/lib/utils";

const MONTHS = ["","Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];

const STATUS_COLOR: Record<string, string> = {
  Scheduled:   "bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800",
  Rescheduled: "bg-amber-100 text-amber-700 border-amber-200 dark:bg-amber-900/30 dark:text-amber-400 dark:border-amber-800",
  Completed:   "bg-green-100 text-green-700 border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800",
  Cancelled:   "bg-muted text-muted-foreground border-border",
};

function fmtTk(v: number) {
  return "৳" + v.toLocaleString("en-BD", { maximumFractionDigits: 0 });
}

// ── Complete Dialog ──────────────────────────────────────────────────────────

function CompleteDialog({ visit, onClose, onDone }: { visit: Visit; onClose: () => void; onDone: () => void }) {
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);

  async function handleComplete() {
    setLoading(true);
    try {
      await visitingApi.complete(visit.id);
      toast({ title: "Visit marked as completed" });
      onDone(); onClose();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Failed to complete visit";
      toast({ title: msg, variant: "destructive" });
    } finally { setLoading(false); }
  }

  const username = visit.internet_account?.username ?? "—";
  const billStatus = visit.bill?.status ?? "unknown";
  const isPaid = billStatus === "paid";

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Complete Visit</DialogTitle>
        </DialogHeader>
        <div className="py-3 space-y-3">
          <p className="text-sm text-muted-foreground">Customer: <span className="font-semibold text-foreground">{username}</span></p>
          <p className="text-sm text-muted-foreground">
            Bill Status:{" "}
            <span className={cn("font-semibold", isPaid ? "text-green-600" : "text-red-600")}>
              {billStatus}
            </span>
          </p>
          {!isPaid && (
            <div className="rounded-lg border border-red-200 bg-red-50 dark:bg-red-900/20 dark:border-red-800 p-3 flex gap-2 text-sm text-red-700 dark:text-red-400">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              Customer bill is still unpaid. Please collect payment first, then complete the visit.
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={loading}>Cancel</Button>
          <Button onClick={handleComplete} disabled={loading || !isPaid} className="bg-green-600 hover:bg-green-700 text-white">
            {loading ? "Completing…" : "Mark Completed"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Reschedule Dialog ────────────────────────────────────────────────────────

function RescheduleDialog({ visit, onClose, onDone }: { visit: Visit; onClose: () => void; onDone: () => void }) {
  const { toast } = useToast();
  const [date, setDate] = useState("");
  const [time, setTime] = useState("");
  const [staffId, setStaffId] = useState("");
  const [notes, setNotes] = useState("");
  const [loading, setLoading] = useState(false);

  const { data: usersData } = useQuery({
    queryKey: ["users-for-visiting"],
    queryFn: () => usersApi.list({ status: "active", page_size: 200 }),
  });
  const staffList: User[] = usersData?.data?.data ?? [];

  async function handleReschedule() {
    if (!date || !time) {
      toast({ title: "New date and time are required", variant: "destructive" });
      return;
    }
    setLoading(true);
    try {
      await visitingApi.reschedule(visit.id, {
        scheduled_date: date, scheduled_time: time,
        assigned_staff_id: staffId || undefined, notes,
      });
      toast({ title: "Visit rescheduled" });
      onDone(); onClose();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Failed to reschedule";
      toast({ title: msg, variant: "destructive" });
    } finally { setLoading(false); }
  }

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Reschedule Visit</DialogTitle>
        </DialogHeader>
        <div className="py-2 space-y-4">
          <p className="text-sm text-muted-foreground">
            Current: <span className="font-medium text-foreground">{visit.scheduled_date.split("T")[0]} at {visit.scheduled_time}</span>
          </p>
          <div className="space-y-1.5">
            <Label>New Date <span className="text-red-500">*</span></Label>
            <Input type="date" value={date} onChange={e => setDate(e.target.value)} min={new Date().toISOString().split("T")[0]} />
          </div>
          <div className="space-y-1.5">
            <Label>New Time <span className="text-red-500">*</span></Label>
            <Input type="time" value={time} onChange={e => setTime(e.target.value)} />
          </div>
          <div className="space-y-1.5">
            <Label>Staff <span className="text-muted-foreground text-xs">(leave blank to keep current)</span></Label>
            <Select value={staffId} onValueChange={setStaffId}>
              <SelectTrigger>
                <SelectValue placeholder={visit.assigned_staff?.full_name ?? "Keep current staff"} />
              </SelectTrigger>
              <SelectContent>
                {staffList.map(u => (
                  <SelectItem key={u.id} value={u.id}>{u.full_name} ({u.username})</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>Notes <span className="text-muted-foreground text-xs">(optional)</span></Label>
            <Textarea value={notes} onChange={e => setNotes(e.target.value)} rows={2} placeholder="Reason for rescheduling…" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={loading}>Cancel</Button>
          <Button onClick={handleReschedule} disabled={loading} className="bg-blue-600 hover:bg-blue-700 text-white">
            {loading ? "Rescheduling…" : "Reschedule"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Cancel Dialog ────────────────────────────────────────────────────────────

function CancelDialog({ visit, onClose, onDone }: { visit: Visit; onClose: () => void; onDone: () => void }) {
  const { toast } = useToast();
  const [loading, setLoading] = useState(false);
  async function handleCancel() {
    setLoading(true);
    try {
      await visitingApi.cancel(visit.id);
      toast({ title: "Visit cancelled" });
      onDone(); onClose();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Failed to cancel";
      toast({ title: msg, variant: "destructive" });
    } finally { setLoading(false); }
  }
  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-sm">
        <DialogHeader><DialogTitle>Cancel Visit?</DialogTitle></DialogHeader>
        <p className="text-sm text-muted-foreground py-3">
          Cancel visit for <span className="font-semibold text-foreground">{visit.internet_account?.username}</span> scheduled on{" "}
          {visit.scheduled_date.split("T")[0]} at {visit.scheduled_time}?
        </p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={loading}>Back</Button>
          <Button variant="destructive" onClick={handleCancel} disabled={loading}>
            {loading ? "Cancelling…" : "Cancel Visit"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Main Page ────────────────────────────────────────────────────────────────

const DATE_PRESETS = [
  { value: "today",     label: "Today" },
  { value: "tomorrow",  label: "Tomorrow" },
  { value: "this_week", label: "This Week" },
  { value: "all",       label: "All" },
] as const;

export default function VisitSchedulePage() {
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [datePreset, setDatePreset] = useState<string>("today");
  const [staffFilter, setStaffFilter] = useState("all");
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 25;

  const [completing, setCompleting] = useState<Visit | null>(null);
  const [rescheduling, setRescheduling] = useState<Visit | null>(null);
  const [cancelling, setCancelling] = useState<Visit | null>(null);

  const params = {
    page,
    page_size: PAGE_SIZE,
    status: status !== "all" ? status : undefined,
    date_preset: datePreset !== "all" ? (datePreset as "today" | "tomorrow" | "this_week") : undefined,
    assigned_staff_id: staffFilter !== "all" ? staffFilter : undefined,
    search: search || undefined,
  };

  const { data, isLoading } = useQuery({
    queryKey: ["visiting-schedule", params],
    queryFn: () => visitingApi.list(params),
  });

  const { data: usersData } = useQuery({
    queryKey: ["users-for-visiting"],
    queryFn: () => usersApi.list({ status: "active", page_size: 200 }),
  });

  const visits: Visit[] = data?.data?.data ?? [];
  const total: number = data?.data?.total ?? 0;
  const totalPages: number = data?.data?.total_pages ?? 1;
  const staffList: User[] = usersData?.data?.data ?? [];

  function refresh() {
    qc.invalidateQueries({ queryKey: ["visiting-schedule"] });
    qc.invalidateQueries({ queryKey: ["visiting-pending"] });
    qc.invalidateQueries({ queryKey: ["dashboard-stats"] });
  }

  const isActive = (v: Visit) => v.status === "Scheduled" || v.status === "Rescheduled";

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Visit Schedule" subtitle="All scheduled collection visits" />

      <div className="flex-1 overflow-auto p-6 space-y-5">

        {/* Back link */}
        <div>
          <Link href="/visiting" className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors">
            <ArrowLeft className="w-4 h-4" /> Pending Customers
          </Link>
        </div>

        {/* Filters row */}
        <div className="flex gap-3 flex-wrap items-center">
          {/* Date preset tabs */}
          <div className="flex rounded-lg border border-border bg-background overflow-hidden">
            {DATE_PRESETS.map(p => (
              <button
                key={p.value}
                onClick={() => { setDatePreset(p.value); setPage(1); }}
                className={cn(
                  "px-3 py-1.5 text-sm font-medium border-r border-border last:border-0 transition-colors",
                  datePreset === p.value
                    ? "bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-muted"
                )}
              >
                {p.label}
              </button>
            ))}
          </div>

          {/* Status */}
          <Select value={status} onValueChange={v => { setStatus(v); setPage(1); }}>
            <SelectTrigger className="w-36">
              <SelectValue placeholder="Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Status</SelectItem>
              <SelectItem value="Scheduled">Scheduled</SelectItem>
              <SelectItem value="Rescheduled">Rescheduled</SelectItem>
              <SelectItem value="Completed">Completed</SelectItem>
              <SelectItem value="Cancelled">Cancelled</SelectItem>
            </SelectContent>
          </Select>

          {/* Staff filter */}
          <Select value={staffFilter} onValueChange={v => { setStaffFilter(v); setPage(1); }}>
            <SelectTrigger className="w-44">
              <SelectValue placeholder="All Staff" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Staff</SelectItem>
              {staffList.map(u => (
                <SelectItem key={u.id} value={u.id}>{u.full_name}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Search */}
          <div className="relative flex-1 min-w-48">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder="Search by username…"
              value={search}
              onChange={e => { setSearch(e.target.value); setPage(1); }}
            />
          </div>

          <p className="text-sm text-muted-foreground ml-auto">{total} result{total !== 1 ? "s" : ""}</p>
        </div>

        {/* Table */}
        <div className="rounded-lg border border-border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/50">
                <TableHead>Customer</TableHead>
                <TableHead>Bill</TableHead>
                <TableHead><Clock className="w-3.5 h-3.5 inline mr-1" />Date & Time</TableHead>
                <TableHead><User2 className="w-3.5 h-3.5 inline mr-1" />Assigned Staff</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Notes</TableHead>
                <TableHead className="text-right">Actions</TableHead>
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
              ) : visits.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-12 text-muted-foreground">
                    <CalendarDays className="w-8 h-8 mx-auto mb-2 opacity-30" />
                    No visits found for the selected filters
                  </TableCell>
                </TableRow>
              ) : (
                visits.map(v => {
                  const username = v.internet_account?.username ?? "—";
                  const billDue = v.bill?.due_amount ?? 0;
                  const billMonth = v.bill
                    ? `${MONTHS[v.bill.billing_month]} ${v.bill.billing_year}`
                    : `${MONTHS[v.billing_month]} ${v.billing_year}`;
                  return (
                    <TableRow key={v.id}>
                      <TableCell>
                        <p className="font-medium text-foreground">{username}</p>
                        <p className="text-xs text-muted-foreground">
                          {v.internet_account?.comment ?? v.bill?.package?.display_name ?? ""}
                        </p>
                      </TableCell>
                      <TableCell>
                        <p className="text-sm">{billMonth}</p>
                        <p className="text-xs font-semibold text-red-500">{fmtTk(billDue)} due</p>
                      </TableCell>
                      <TableCell>
                        <p className="text-sm font-medium">{v.scheduled_date.split("T")[0]}</p>
                        <p className="text-xs text-muted-foreground">{v.scheduled_time}</p>
                      </TableCell>
                      <TableCell>
                        <p className="text-sm">{v.assigned_staff?.full_name ?? "—"}</p>
                        <p className="text-xs text-muted-foreground">{v.assigned_staff?.username}</p>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className={cn("text-xs", STATUS_COLOR[v.status])}>
                          {v.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="max-w-[160px]">
                        <p className="text-xs text-muted-foreground truncate">{v.notes || "—"}</p>
                      </TableCell>
                      <TableCell className="text-right">
                        {isActive(v) ? (
                          <div className="flex justify-end gap-1.5">
                            <Button
                              size="sm" variant="outline"
                              className="gap-1 text-green-700 border-green-200 hover:bg-green-50 dark:text-green-400 dark:border-green-800 dark:hover:bg-green-900/20"
                              onClick={() => setCompleting(v)}
                            >
                              <CheckCircle2 className="w-3.5 h-3.5" /> Complete
                            </Button>
                            <Button
                              size="sm" variant="outline"
                              className="gap-1 text-amber-700 border-amber-200 hover:bg-amber-50 dark:text-amber-400 dark:border-amber-800 dark:hover:bg-amber-900/20"
                              onClick={() => setRescheduling(v)}
                            >
                              <RotateCcw className="w-3.5 h-3.5" /> Reschedule
                            </Button>
                            <Button
                              size="sm" variant="outline"
                              className="gap-1 text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/20"
                              onClick={() => setCancelling(v)}
                            >
                              <XCircle className="w-3.5 h-3.5" />
                            </Button>
                          </div>
                        ) : (
                          <span className="text-xs text-muted-foreground">{v.status}</span>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between text-sm text-muted-foreground">
            <span>Page {page} of {totalPages}</span>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage(p => p - 1)}>
                <ChevronLeft className="w-4 h-4" />
              </Button>
              <Button size="sm" variant="outline" disabled={page >= totalPages} onClick={() => setPage(p => p + 1)}>
                <ChevronRight className="w-4 h-4" />
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Action Dialogs */}
      {completing && (
        <CompleteDialog visit={completing} onClose={() => setCompleting(null)} onDone={refresh} />
      )}
      {rescheduling && (
        <RescheduleDialog visit={rescheduling} onClose={() => setRescheduling(null)} onDone={refresh} />
      )}
      {cancelling && (
        <CancelDialog visit={cancelling} onClose={() => setCancelling(null)} onDone={refresh} />
      )}
    </div>
  );
}
