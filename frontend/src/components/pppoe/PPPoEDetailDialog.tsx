"use client";

import { PPPoESecret } from "@/types";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { formatDate } from "@/lib/utils";

interface Props {
  secret: PPPoESecret | null;
  onClose: () => void;
}

function Row({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="flex items-start gap-2">
      <span className="text-sm text-muted-foreground w-36 shrink-0">{label}</span>
      <span className="text-sm font-medium break-all">{value || "—"}</span>
    </div>
  );
}

export function PPPoEDetailDialog({ secret, onClose }: Props) {
  if (!secret) return null;
  return (
    <Dialog open={!!secret} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="font-mono">{secret.username}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div className="flex items-center gap-2 mb-4">
            <Badge variant={secret.disabled ? "destructive" : "default"}>
              {secret.disabled ? "Disabled" : "Active"}
            </Badge>
            <Badge variant="outline">{secret.profile || "No profile"}</Badge>
          </div>
          <Row label="RouterOS ID" value={secret.routeros_id} />
          <Row label="Service" value={secret.service} />
          <Row label="Local Address" value={secret.local_address} />
          <Row label="Remote Address" value={secret.remote_address} />
          <Row label="Caller ID" value={secret.caller_id} />
          <Row label="Comment" value={secret.comment} />
          <Row label="Router" value={secret.router?.name} />
          <Row label="Last Seen" value={formatDate(secret.last_seen)} />
          <Row label="Sync Time" value={formatDate(secret.sync_time)} />
          <Row label="Created" value={formatDate(secret.created_at)} />
        </div>
      </DialogContent>
    </Dialog>
  );
}
