"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, CheckCircle, XCircle, Eye } from "lucide-react";
import { pppoeApi, routersApi } from "@/lib/api";
import { PPPoESecret, Router, PaginatedResponse } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
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
import { PPPoEDetailDialog } from "@/components/pppoe/PPPoEDetailDialog";

export default function PPPoEPage() {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [routerFilter, setRouterFilter] = useState<string>("");
  const [disabledFilter, setDisabledFilter] = useState<string>("");
  const [selected, setSelected] = useState<PPPoESecret | null>(null);

  const { data, isLoading } = useQuery<{ data: PaginatedResponse<PPPoESecret> }>({
    queryKey: ["pppoe-secrets", page, search, routerFilter, disabledFilter],
    queryFn: () =>
      pppoeApi.secrets({
        page,
        page_size: 20,
        search: search || undefined,
        router_id: routerFilter || undefined,
        disabled:
          disabledFilter === "true" ? true : disabledFilter === "false" ? false : undefined,
      }),
  });

  const { data: routerData } = useQuery<{ data: PaginatedResponse<Router> }>({
    queryKey: ["routers", 1, ""],
    queryFn: () => routersApi.list({ page: 1, page_size: 100 }),
  });

  const secrets = data?.data?.data ?? [];
  const total = data?.data?.total ?? 0;
  const totalPages = data?.data?.total_pages ?? 1;
  const routers = routerData?.data?.data ?? [];

  return (
    <div>
      <Topbar title="PPPoE Secrets" breadcrumbs={[{ label: "Home" }, { label: "PPPoE Secrets" }]} />

      <div className="p-6 space-y-4">
        {/* Toolbar */}
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input
              placeholder="Search username or comment…"
              value={search}
              onChange={(e) => { setSearch(e.target.value); setPage(1); }}
              className="pl-9"
            />
          </div>
          <Select value={routerFilter} onValueChange={(v) => { setRouterFilter(v); setPage(1); }}>
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
          <Select value={disabledFilter} onValueChange={(v) => { setDisabledFilter(v); setPage(1); }}>
            <SelectTrigger className="w-[140px]">
              <SelectValue placeholder="All status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">All status</SelectItem>
              <SelectItem value="false">Active</SelectItem>
              <SelectItem value="true">Disabled</SelectItem>
            </SelectContent>
          </Select>
          <span className="text-sm text-muted-foreground ml-auto">{total} records</span>
        </div>

        {/* Table */}
        <div className="rounded-lg border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Username</TableHead>
                <TableHead>Profile</TableHead>
                <TableHead>Router</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Remote Address</TableHead>
                <TableHead>Comment</TableHead>
                <TableHead>Sync Time</TableHead>
                <TableHead className="w-10"></TableHead>
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
              ) : secrets.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={8} className="text-center py-12 text-muted-foreground">
                    No PPPoE secrets found. Sync a router first.
                  </TableCell>
                </TableRow>
              ) : (
                secrets.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell className="font-medium font-mono">{s.username}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{s.profile || "—"}</Badge>
                    </TableCell>
                    <TableCell className="text-sm">{s.router?.name ?? "—"}</TableCell>
                    <TableCell>
                      <span className="flex items-center gap-1.5 text-sm">
                        {s.disabled ? (
                          <><XCircle className="w-3.5 h-3.5 text-red-400" /> Disabled</>
                        ) : (
                          <><CheckCircle className="w-3.5 h-3.5 text-green-500" /> Active</>
                        )}
                      </span>
                    </TableCell>
                    <TableCell className="font-mono text-sm">{s.remote_address || "—"}</TableCell>
                    <TableCell className="text-sm max-w-[150px] truncate">{s.comment || "—"}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatDate(s.sync_time)}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setSelected(s)}
                      >
                        <Eye className="w-4 h-4" />
                      </Button>
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
            <span>{total} secrets</span>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" disabled={page === 1} onClick={() => setPage((p) => p - 1)}>
                Previous
              </Button>
              <span className="flex items-center px-2">{page} / {totalPages}</span>
              <Button variant="outline" size="sm" disabled={page === totalPages} onClick={() => setPage((p) => p + 1)}>
                Next
              </Button>
            </div>
          </div>
        )}
      </div>

      <PPPoEDetailDialog secret={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
