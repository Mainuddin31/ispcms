"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { pppoeApi, routersApi } from "@/lib/api";
import { PPPoESession, Router, PaginatedResponse } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { formatDate } from "@/lib/utils";

export default function SessionsPage() {
  const [search, setSearch] = useState("");
  const [routerFilter, setRouterFilter] = useState<string>("");

  const { data, isLoading } = useQuery<{ data: { data: PPPoESession[] } }>({
    queryKey: ["pppoe-sessions", routerFilter],
    queryFn: () => pppoeApi.sessions({ router_id: routerFilter || undefined }),
    refetchInterval: 30_000,
  });

  const { data: routerData } = useQuery<{ data: PaginatedResponse<Router> }>({
    queryKey: ["routers", 1, ""],
    queryFn: () => routersApi.list({ page: 1, page_size: 100 }),
  });

  const sessions = (data?.data?.data ?? []).filter((s) =>
    search ? s.username.toLowerCase().includes(search.toLowerCase()) : true
  );
  const routers = routerData?.data?.data ?? [];

  return (
    <div>
      <Topbar
        title="Active Sessions"
        breadcrumbs={[{ label: "Home" }, { label: "Active Sessions" }]}
      />

      <div className="p-6 space-y-4">
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Filter by username…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
          <Select value={routerFilter} onValueChange={setRouterFilter}>
            <SelectTrigger className="w-[180px]">
              <SelectValue placeholder="All routers" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">All routers</SelectItem>
              {routers.map((r) => (
                <SelectItem key={r.id} value={r.id}>{r.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <span className="text-sm text-muted-foreground ml-auto">{sessions.length} sessions</span>
        </div>

        <div className="rounded-lg border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>IP Address</TableHead>
                <TableHead>Uptime</TableHead>
                <TableHead>Connected Since</TableHead>
                <TableHead>Session ID</TableHead>
                <TableHead>Last Sync</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                Array.from({ length: 6 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 6 }).map((_, j) => (
                      <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
                    ))}
                  </TableRow>
                ))
              ) : sessions.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-12 text-muted-foreground">
                    No active sessions. Sync a router to see live sessions.
                  </TableCell>
                </TableRow>
              ) : (
                sessions.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell className="font-medium font-mono">{s.username}</TableCell>
                    <TableCell className="font-mono text-sm">{s.current_ip || "—"}</TableCell>
                    <TableCell className="text-sm">{s.uptime || "—"}</TableCell>
                    <TableCell className="text-sm">{formatDate(s.connected_since)}</TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {s.session_id || "—"}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatDate(s.sync_time)}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}
