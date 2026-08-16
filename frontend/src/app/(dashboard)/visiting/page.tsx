"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CalendarClock, Search, UserCheck, Clock, AlertCircle,
  CheckCircle2, CalendarDays, ChevronRight, User2,
} from "lucide-react";
import Link from "next/link";
import { visitingApi, usersApi } from "@/lib/api";
import { PendingVisitCustomer, User } from "@/types";
import { useAuth } from "@/contexts/AuthContext";
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

const MONTHS = [
  "","January","February","March","April","May","June",
  "July","August","September","October","November","December",
];

const VISIT_STATUS_COLOR: Record<string, string> = {
  Scheduled:   "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  Rescheduled: "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400",
  Completed:   "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  Cancelled:   "bg-muted text-muted-foreground",
};

function fmtTk(v: number) {
  return "৳" + v.toLocaleString("en-BD", { minimumFractionDigits: 0, maximumFractionDigits: 0 });
}

// ── Schedule Dialog ──────────────────────────────────────────────────────────

interface ScheduleDialogProps {
  customer: PendingVisitCustomer | null;
  onClose: () => void;
  onSaved: () => void;
}

function ScheduleDialog({ customer, onClose, onSaved }: ScheduleDialogProps) {
  const { toast } = useToast();
  const { user: currentUser, hasRole } = useAuth();
  const isRestricted = !hasRole("admin") && !hasRole("super_admin");

  const [date, setDate] = useState(customer?.scheduled_date?.split("T")[0] ?? "");
  const [time, setTime] = useState(customer?.scheduled_time ?? "17:00");
  // Non-admin users are always assigned to themselves
  const [staffId, setStaffId] = useState(isRestricted ? (currentUser?.id ?? "") : "");
  const [notes, setNotes] = useState("");
  const [saving, setSaving] = useState(false);

  const { data: usersData } = useQuery({
    queryKey: ["users-for-visiting"],
    queryFn: () => usersApi.list({ status: "active", page_size: 200 }),
    enabled: !!customer && !isRestricted,
  });
  const staffList: User[] = usersData?.data?.data ?? [];

  const isEdit = !!(customer?.existing_visit_id);

  async function handleSave() {
    if (!customer) return;
    if (!date || !time || (!isEdit && !staffId)) {
      toast({ title: "Date, time, and staff are required", variant: "destructive" });
      return;
    }
    setSaving(true);
    try {
      if (isEdit) {
        await visitingApi.update(customer.existing_visit_id!, { scheduled_date: date, scheduled_time: time, assigned_staff_id: staffId, notes });
        toast({ title: "Visit updated" });
      } else {
        await visitingApi.create({
          internet_account_id: customer.internet_account_id,
          bill_id: customer.bill_id,
          billing_month: customer.billing_month,
          billing_year: customer.billing_year,
          assigned_staff_id: staffId,
          scheduled_date: date,
          scheduled_time: time,
          notes,
        });
        toast({ title: "Visit scheduled" });
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error ?? "Failed to save visit";
      toast({ title: msg, variant: "destructive" });
    } finally {
      setSaving(false);
    }
  }

  if (!customer) return null;

  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Visit Schedule" : "Schedule Visit"}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Customer (read-only) */}
          <div className="rounded-lg bg-muted/40 border border-border p-3 space-y-1">
            <p className="font-semibold text-foreground">{customer.username}</p>
            {customer.comment && <p className="text-sm text-muted-foreground">{customer.comment}</p>}
            <div className="flex gap-4 text-sm mt-1">
              <span className="text-muted-foreground">Package: <span className="font-medium text-foreground">{customer.package_name}</span></span>
              <span className="text-red-600 font-medium">Due: {fmtTk(customer.due_amount)}</span>
            </div>
            <div className="flex gap-2 mt-1">
              <span className={cn("text-xs px-2 py-0.5 rounded-full font-medium", {
                "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400": customer.bill_status === "pending",
                "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400": customer.bill_status === "due",
                "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400": customer.bill_status === "partial",
              })}>
                Bill: {customer.bill_status}
              </span>
              <span className="text-xs text-muted-foreground">{MONTHS[customer.billing_month]} {customer.billing_year}</span>
            </div>
          </div>

          {/* Date */}
          <div className="space-y-1.5">
            <Label htmlFor="visit-date">Date <span className="text-red-500">*</span></Label>
            <Input
              id="visit-date"
              type="date"
              value={date}
              onChange={e => setDate(e.target.value)}
            />
          </div>

          {/* Time */}
          <div className="space-y-1.5">
            <Label htmlFor="visit-time">Time <span className="text-red-500">*</span></Label>
            <Input
              id="visit-time"
              type="time"
              value={time}
              onChange={e => setTime(e.target.value)}
            />
          </div>

          {/* Staff */}
          <div className="space-y-1.5">
            <Label>Assigned Staff <span className="text-red-500">*</span></Label>
            {isRestricted ? (
              <div className="flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-2 text-sm">
                <User2 className="w-4 h-4 text-muted-foreground shrink-0" />
                <span className="font-medium">{currentUser?.full_name}</span>
                <span className="text-muted-foreground">({currentUser?.username})</span>
              </div>
            ) : (
              <Select value={staffId} onValueChange={setStaffId}>
                <SelectTrigger>
                  <SelectValue placeholder="Select staff member…" />
                </SelectTrigger>
                <SelectContent>
                  {staffList.map(u => (
                    <SelectItem key={u.id} value={u.id}>
                      {u.full_name} ({u.username})
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label htmlFor="visit-notes">
              Notes <span className="text-muted-foreground text-xs">(optional)</span>
            </Label>
            <Textarea
              id="visit-notes"
              placeholder="e.g. Customer requested visit after 6 PM"
              value={notes}
              onChange={e => setNotes(e.target.value)}
              rows={2}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={saving}>Cancel</Button>
          <Button onClick={handleSave} disabled={saving} className="bg-blue-600 hover:bg-blue-700 text-white">
            {saving ? "Saving…" : isEdit ? "Update Visit" : "Schedule Visit"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function VisitingPage() {
  const qc = useQueryClient();
  const now = new Date();
  const [month, setMonth] = useState(now.getMonth() + 1);
  const [year, setYear] = useState(now.getFullYear());
  const [search, setSearch] = useState("");
  const [scheduleTarget, setScheduleTarget] = useState<PendingVisitCustomer | null>(null);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["visiting-pending", month, year],
    queryFn: () => visitingApi.pendingCustomers({ month, year }),
  });

  const customers: PendingVisitCustomer[] = data?.data?.data ?? [];

  const filtered = customers.filter(c => {
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      c.username.toLowerCase().includes(q) ||
      (c.comment ?? "").toLowerCase().includes(q)
    );
  });

  const scheduledCount = customers.filter(c => c.existing_visit_id).length;
  const unscheduledCount = customers.length - scheduledCount;

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Visiting" subtitle="Collection follow-up for pending bills" />

      <div className="flex-1 overflow-auto p-6 space-y-5">

        {/* Header row */}
        <div className="flex items-center justify-between gap-4 flex-wrap">
          <div className="flex gap-2 flex-wrap">
            <Select value={String(month)} onValueChange={v => setMonth(Number(v))}>
              <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
              <SelectContent>
                {MONTHS.slice(1).map((m, i) => (
                  <SelectItem key={i + 1} value={String(i + 1)}>{m}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={String(year)} onValueChange={v => setYear(Number(v))}>
              <SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
              <SelectContent>
                {[year - 1, year, year + 1].map(y => (
                  <SelectItem key={y} value={String(y)}>{y}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Link href="/visiting/schedule">
            <Button variant="outline" size="sm" className="gap-1.5">
              <CalendarDays className="w-4 h-4" />
              Visit Schedule
              <ChevronRight className="w-3.5 h-3.5" />
            </Button>
          </Link>
        </div>

        {/* Summary cards */}
        {!isLoading && (
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-xl border border-border bg-card p-4 flex items-center gap-3">
              <div className="p-2 rounded-lg bg-red-100 dark:bg-red-900/30">
                <AlertCircle className="w-5 h-5 text-red-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-foreground">{customers.length}</p>
                <p className="text-xs text-muted-foreground">Pending Customers</p>
              </div>
            </div>
            <div className="rounded-xl border border-border bg-card p-4 flex items-center gap-3">
              <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900/30">
                <CalendarClock className="w-5 h-5 text-blue-500" />
              </div>
              <div>
                <p className="text-2xl font-bold text-foreground">{scheduledCount}</p>
                <p className="text-xs text-muted-foreground">Scheduled</p>
              </div>
            </div>
            <div className="rounded-xl border border-border bg-card p-4 flex items-center gap-3">
              <div className="p-2 rounded-lg bg-muted">
                <Clock className="w-5 h-5 text-muted-foreground" />
              </div>
              <div>
                <p className="text-2xl font-bold text-foreground">{unscheduledCount}</p>
                <p className="text-xs text-muted-foreground">Not Yet Scheduled</p>
              </div>
            </div>
          </div>
        )}

        {/* Search */}
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder="Search by username or name…"
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
        </div>

        {/* Table */}
        <div className="rounded-lg border border-border overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="bg-muted/50">
                <TableHead>Customer / PPPoE</TableHead>
                <TableHead>Package</TableHead>
                <TableHead className="text-right">Bill</TableHead>
                <TableHead className="text-right">Paid</TableHead>
                <TableHead className="text-right">Due</TableHead>
                <TableHead>Bill Status</TableHead>
                <TableHead>Scheduled</TableHead>
                <TableHead className="text-right">Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 6 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 8 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : isError ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center text-red-500 py-10">
                    Failed to load pending customers
                  </TableCell>
                </TableRow>
              ) : filtered.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-12">
                    <CheckCircle2 className="w-8 h-8 text-green-400 mx-auto mb-2" />
                    <p className="text-muted-foreground font-medium">
                      {customers.length === 0
                        ? `No pending bills for ${MONTHS[month]} ${year}`
                        : "No results match your search"}
                    </p>
                  </TableCell>
                </TableRow>
              ) : (
                filtered.map(c => (
                  <TableRow key={c.bill_id}>
                    <TableCell>
                      <p className="font-medium text-foreground">{c.username}</p>
                      {c.comment && <p className="text-xs text-muted-foreground">{c.comment}</p>}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{c.package_name}</TableCell>
                    <TableCell className="text-right text-sm">{fmtTk(c.total_amount)}</TableCell>
                    <TableCell className="text-right text-sm text-green-600">{fmtTk(c.paid_amount)}</TableCell>
                    <TableCell className="text-right text-sm font-semibold text-red-600">{fmtTk(c.due_amount)}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className={cn("text-xs", {
                        "border-yellow-300 bg-yellow-50 text-yellow-700 dark:bg-yellow-900/20 dark:text-yellow-400": c.bill_status === "pending",
                        "border-red-300 bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400": c.bill_status === "due",
                        "border-orange-300 bg-orange-50 text-orange-700 dark:bg-orange-900/20 dark:text-orange-400": c.bill_status === "partial",
                      })}>
                        {c.bill_status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {c.existing_visit_id ? (
                        <div>
                          <span className={cn("text-xs px-1.5 py-0.5 rounded font-medium", VISIT_STATUS_COLOR[c.visit_status ?? "Scheduled"])}>
                            {c.visit_status}
                          </span>
                          <p className="text-xs text-muted-foreground mt-0.5">
                            {c.scheduled_date?.split("T")[0]}{c.scheduled_time && ` · ${c.scheduled_time}`}
                          </p>
                          {c.assigned_staff_name && (
                            <p className="text-xs text-muted-foreground">{c.assigned_staff_name}</p>
                          )}
                        </div>
                      ) : (
                        <span className="text-xs text-muted-foreground italic">Not scheduled</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="sm"
                        variant={c.existing_visit_id ? "outline" : "default"}
                        className={cn("gap-1", !c.existing_visit_id && "bg-blue-600 hover:bg-blue-700 text-white")}
                        onClick={() => setScheduleTarget(c)}
                      >
                        <UserCheck className="w-3.5 h-3.5" />
                        {c.existing_visit_id ? "Edit" : "Schedule"}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      <ScheduleDialog
        key={scheduleTarget?.bill_id ?? "none"}
        customer={scheduleTarget}
        onClose={() => setScheduleTarget(null)}
        onSaved={() => qc.invalidateQueries({ queryKey: ["visiting-pending"] })}
      />
    </div>
  );
}
