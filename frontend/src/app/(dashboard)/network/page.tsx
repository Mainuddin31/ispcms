"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import {
  Server,
  Wifi,
  WifiOff,
  Network,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Loader2,
  RefreshCw,
  Activity,
} from "lucide-react";
import { oltsApi } from "@/lib/api";
import { OLTStats, OLTSyncLog, OLT } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

function StatCard({
  label,
  value,
  icon: Icon,
  color,
  sub,
}: {
  label: string;
  value: number | string;
  icon: React.ElementType;
  color: string;
  sub?: string;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-slate-500">{label}</p>
            <p className="text-2xl font-bold mt-1">{value}</p>
            {sub && <p className="text-xs text-slate-400 mt-0.5">{sub}</p>}
          </div>
          <div className={cn("w-10 h-10 rounded-lg flex items-center justify-center", color)}>
            <Icon className="w-5 h-5 text-white" />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    success: "bg-green-100 text-green-700",
    failed: "bg-red-100 text-red-700",
    running: "bg-blue-100 text-blue-700",
    active: "bg-green-100 text-green-700",
    maintenance: "bg-yellow-100 text-yellow-700",
    offline: "bg-red-100 text-red-700",
    disabled: "bg-slate-100 text-slate-600",
  };
  return (
    <span className={cn("px-2 py-0.5 rounded text-xs font-medium capitalize", map[status] ?? "bg-slate-100 text-slate-600")}>
      {status}
    </span>
  );
}

function formatDuration(ms: number) {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatRelative(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export default function NetworkDashboardPage() {
  const { data: statsData, isLoading: statsLoading } = useQuery<{ data: { data: OLTStats } }>({
    queryKey: ["olt-stats"],
    queryFn: () => oltsApi.stats(),
    refetchInterval: 60000,
  });

  const { data: syncLogsData, isLoading: logsLoading } = useQuery<{ data: { data: OLTSyncLog[] } }>({
    queryKey: ["olt-recent-sync-logs"],
    queryFn: () => oltsApi.recentSyncLogs(20),
    refetchInterval: 30000,
  });

  const { data: oltsData, isLoading: oltsLoading } = useQuery<{ data: { data: { data: OLT[] } } }>({
    queryKey: ["olts-health"],
    queryFn: () => oltsApi.list({}),
    refetchInterval: 60000,
  });

  const stats = statsData?.data?.data;
  const syncLogs = syncLogsData?.data?.data ?? [];
  const olts = oltsData?.data?.data?.data ?? [];

  return (
    <div className="flex flex-col h-full">
      <Topbar title="Network Dashboard" />
      <div className="flex-1 overflow-auto p-6 space-y-6">
        {/* Stats Grid */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {statsLoading ? (
            Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} className="h-24 rounded-xl" />)
          ) : (
            <>
              <StatCard label="Total OLTs" value={stats?.total_olts ?? 0} icon={Server} color="bg-blue-500" />
              <StatCard label="Active OLTs" value={stats?.active_olts ?? 0} icon={CheckCircle2} color="bg-green-500" />
              <StatCard label="PON Ports" value={stats?.total_pon_ports ?? 0} icon={Network} color="bg-purple-500" />
              <StatCard
                label="Port Utilization"
                value={`${stats?.port_utilization_pct?.toFixed(1) ?? 0}%`}
                icon={Activity}
                color="bg-indigo-500"
              />
              <StatCard label="Online ONUs" value={stats?.online_onus ?? 0} icon={Wifi} color="bg-emerald-500" />
              <StatCard label="Offline ONUs" value={stats?.offline_onus ?? 0} icon={WifiOff} color="bg-red-500" />
              <StatCard
                label="Total ONUs"
                value={stats?.total_onus ?? 0}
                icon={Network}
                color="bg-cyan-500"
              />
              <StatCard
                label="Unassigned ONUs"
                value={stats?.unassigned_onus ?? 0}
                icon={AlertCircle}
                color="bg-amber-500"
                sub="not linked to accounts"
              />
            </>
          )}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* OLT Health Table */}
          <Card>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">OLT Health</CardTitle>
                <Link href="/olts">
                  <Button variant="ghost" size="sm" className="text-xs">Manage OLTs →</Button>
                </Link>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {oltsLoading ? (
                <div className="p-4 space-y-2">
                  {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10" />)}
                </div>
              ) : olts.length === 0 ? (
                <div className="text-center py-8 text-slate-500 text-sm">No OLTs configured</div>
              ) : (
                <div className="divide-y divide-slate-100">
                  {olts.slice(0, 10).map((olt) => (
                    <div key={olt.id} className="flex items-center justify-between px-4 py-3 hover:bg-slate-50">
                      <div>
                        <p className="text-sm font-medium">{olt.name}</p>
                        <p className="text-xs text-slate-400">{olt.management_ip}</p>
                      </div>
                      <div className="flex items-center gap-3">
                        {olt.last_sync_at && (
                          <span className="text-xs text-slate-400">{formatRelative(olt.last_sync_at)}</span>
                        )}
                        <StatusBadge status={olt.status} />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Recent Sync Logs */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">Recent Sync Activity</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {logsLoading ? (
                <div className="p-4 space-y-2">
                  {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10" />)}
                </div>
              ) : syncLogs.length === 0 ? (
                <div className="text-center py-8 text-slate-500 text-sm">No sync activity yet</div>
              ) : (
                <div className="divide-y divide-slate-100">
                  {syncLogs.map((log) => (
                    <div key={log.id} className="flex items-center justify-between px-4 py-3">
                      <div className="flex items-center gap-2">
                        {log.status === "success" ? (
                          <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0" />
                        ) : log.status === "failed" ? (
                          <XCircle className="w-4 h-4 text-red-500 shrink-0" />
                        ) : (
                          <Loader2 className="w-4 h-4 text-blue-500 animate-spin shrink-0" />
                        )}
                        <div>
                          <p className="text-sm font-medium">{log.olt?.name ?? "OLT"}</p>
                          <p className="text-xs text-slate-400">
                            {log.onus_discovered} ONUs · {log.new_onus} new · {log.archived_onus} archived
                          </p>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className="text-xs text-slate-400">{formatRelative(log.started_at)}</p>
                        {log.duration_ms > 0 && (
                          <p className="text-xs text-slate-400">{formatDuration(log.duration_ms)}</p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Quick links */}
        <div className="flex gap-3">
          <Link href="/olts">
            <Button variant="outline" className="gap-2">
              <Server className="w-4 h-4" /> Manage OLTs
            </Button>
          </Link>
          <Link href="/onus">
            <Button variant="outline" className="gap-2">
              <Wifi className="w-4 h-4" /> ONU Inventory
            </Button>
          </Link>
          <Link href="/snmp-profiles">
            <Button variant="outline" className="gap-2">
              <RefreshCw className="w-4 h-4" /> SNMP Profiles
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
