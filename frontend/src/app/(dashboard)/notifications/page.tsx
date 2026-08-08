"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Bell, CheckCheck, Loader2 } from "lucide-react";
import { notificationsApi } from "@/lib/api";
import { Notification, PaginatedResponse } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/use-toast";
import { formatRelative } from "@/lib/utils";
import { cn } from "@/lib/utils";
import { useState } from "react";

const SEVERITY_STYLES: Record<string, string> = {
  info: "border-l-blue-400",
  warning: "border-l-amber-400",
  error: "border-l-red-400",
  success: "border-l-green-400",
};

const SEVERITY_BADGE: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  info: "secondary",
  warning: "outline",
  error: "destructive",
  success: "default",
};

export default function NotificationsPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["notifications", page, unreadOnly],
    queryFn: () =>
      notificationsApi
        .list({ page, page_size: 25, unread_only: unreadOnly })
        .then((r) => r.data as PaginatedResponse<Notification> & { data: Notification[] }),
  });

  const markReadMutation = useMutation({
    mutationFn: (id: string) => notificationsApi.markRead(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
      qc.invalidateQueries({ queryKey: ["notif-count"] });
    },
  });

  const markAllMutation = useMutation({
    mutationFn: () => notificationsApi.markAllRead(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["notifications"] });
      qc.invalidateQueries({ queryKey: ["notif-count"] });
      toast({ title: "All notifications marked as read" });
    },
  });

  const notifications = data?.data ?? [];
  const totalPages = data?.total_pages ?? 1;

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Notifications" subtitle="System alerts and billing events" />

      <div className="flex-1 overflow-auto p-6 space-y-4">
        {/* Toolbar */}
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <button
              onClick={() => { setUnreadOnly(false); setPage(1); }}
              className={cn(
                "px-3 py-1.5 rounded-md text-sm font-medium transition-colors",
                !unreadOnly ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"
              )}
            >
              All
            </button>
            <button
              onClick={() => { setUnreadOnly(true); setPage(1); }}
              className={cn(
                "px-3 py-1.5 rounded-md text-sm font-medium transition-colors",
                unreadOnly ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:text-foreground"
              )}
            >
              Unread
            </button>
          </div>
          <div className="flex-1" />
          <Button
            variant="outline"
            size="sm"
            onClick={() => markAllMutation.mutate()}
            disabled={markAllMutation.isPending}
          >
            {markAllMutation.isPending
              ? <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              : <CheckCheck className="h-4 w-4 mr-2" />}
            Mark all read
          </Button>
        </div>

        {/* List */}
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full rounded-lg" />
            ))}
          </div>
        ) : notifications.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-muted-foreground">
            <Bell className="h-12 w-12 mb-3 opacity-30" />
            <p className="font-medium">No notifications</p>
            <p className="text-sm">You&apos;re all caught up!</p>
          </div>
        ) : (
          <div className="space-y-2">
            {notifications.map((n) => (
              <div
                key={n.id}
                className={cn(
                  "rounded-lg border border-l-4 p-4 transition-colors",
                  SEVERITY_STYLES[n.severity] ?? "border-l-slate-300",
                  n.is_read
                    ? "bg-muted/30 opacity-70"
                    : "bg-card"
                )}
              >
                <div className="flex items-start gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <p className={cn("font-medium text-sm", !n.is_read && "font-semibold")}>
                        {n.title}
                      </p>
                      <Badge variant={SEVERITY_BADGE[n.severity] ?? "secondary"} className="capitalize text-xs">
                        {n.severity}
                      </Badge>
                      {!n.is_read && (
                        <span className="inline-block w-2 h-2 rounded-full bg-blue-500" />
                      )}
                    </div>
                    <p className="text-sm text-muted-foreground mt-1">{n.message}</p>
                    <p className="text-xs text-muted-foreground mt-2">{formatRelative(n.created_at)}</p>
                  </div>
                  {!n.is_read && (
                    <Button
                      size="sm"
                      variant="ghost"
                      className="shrink-0"
                      onClick={() => markReadMutation.mutate(n.id)}
                      disabled={markReadMutation.isPending}
                    >
                      Mark read
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}

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
    </div>
  );
}
