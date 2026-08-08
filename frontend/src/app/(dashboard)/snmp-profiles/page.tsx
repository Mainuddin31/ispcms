"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Edit,
  Trash2,
  ChevronDown,
  ChevronRight,
  Loader2,
  Shield,
} from "lucide-react";
import { snmpProfilesApi } from "@/lib/api";
import { SNMPProfile, OIDMap } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/use-toast";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

// ── OID Map Editor ────────────────────────────────────────────────────────────

function OIDMapEditor({
  value,
  onChange,
}: {
  value: OIDMap;
  onChange: (v: OIDMap) => void;
}) {
  const entries = Object.entries(value);

  const setKey = (oldKey: string, newKey: string) => {
    const next: OIDMap = {};
    for (const [k, v] of Object.entries(value)) {
      next[k === oldKey ? newKey : k] = v;
    }
    onChange(next);
  };

  const setValue = (key: string, val: string) => {
    onChange({ ...value, [key]: val });
  };

  const remove = (key: string) => {
    const next = { ...value };
    delete next[key];
    onChange(next);
  };

  const addRow = () => {
    onChange({ ...value, "": "" });
  };

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-[1fr_1fr_auto] gap-1 text-xs text-slate-500 font-medium px-1">
        <span>Key</span>
        <span>OID</span>
        <span />
      </div>
      {entries.map(([k, v], i) => (
        <div key={i} className="grid grid-cols-[1fr_1fr_auto] gap-1 items-center">
          <Input
            value={k}
            onChange={(e) => setKey(k, e.target.value)}
            className="font-mono text-xs h-8"
            placeholder="onu_mac"
          />
          <Input
            value={v}
            onChange={(e) => setValue(k, e.target.value)}
            className="font-mono text-xs h-8"
            placeholder=".1.3.6.1.4.1…"
          />
          <Button
            variant="ghost"
            size="icon"
            className="w-8 h-8 text-red-400 hover:text-red-600"
            onClick={() => remove(k)}
          >
            <Trash2 className="w-3.5 h-3.5" />
          </Button>
        </div>
      ))}
      <Button variant="outline" size="sm" onClick={addRow} className="w-full mt-1">
        <Plus className="w-3.5 h-3.5 mr-1" /> Add OID
      </Button>
    </div>
  );
}

// ── Profile Form Dialog ───────────────────────────────────────────────────────

interface ProfileForm {
  name: string;
  vendor: string;
  technology: "EPON" | "GPON";
  description: string;
  oid_map: OIDMap;
}

const defaultOIDMap: OIDMap = {
  onu_mac: "",
  onu_status: "",
  onu_rx_power: "",
  onu_tx_power: "",
  onu_distance: "",
  onu_serial: "",
  onu_model: "",
  index_port_pos: "0",
  index_onu_pos: "1",
};

function ProfileFormDialog({
  open,
  onClose,
  profile,
  onSaved,
}: {
  open: boolean;
  onClose: () => void;
  profile?: SNMPProfile | null;
  onSaved: () => void;
}) {
  const { toast } = useToast();
  const [form, setForm] = useState<ProfileForm>({
    name: "", vendor: "", technology: "EPON", description: "", oid_map: { ...defaultOIDMap },
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      setForm(profile
        ? {
            name: profile.name,
            vendor: profile.vendor,
            technology: profile.technology,
            description: profile.description ?? "",
            oid_map: { ...profile.oid_map },
          }
        : {
            name: "",
            vendor: "",
            technology: "EPON",
            description: "",
            oid_map: { ...defaultOIDMap },
          }
      );
    }
  }, [open, profile]);

  const set = <K extends keyof ProfileForm>(k: K, v: ProfileForm[K]) =>
    setForm((f) => ({ ...f, [k]: v }));

  const handleSave = async () => {
    if (!form.name || !form.vendor) {
      toast({ title: "Name and vendor are required", variant: "destructive" });
      return;
    }
    setSaving(true);
    try {
      const payload = {
        name: form.name,
        vendor: form.vendor,
        technology: form.technology,
        description: form.description || undefined,
        oid_map: form.oid_map,
      };
      if (profile) {
        await snmpProfilesApi.update(profile.id, payload);
        toast({ title: "Profile updated" });
      } else {
        await snmpProfilesApi.create(payload);
        toast({ title: "Profile created" });
      }
      onSaved();
      onClose();
    } catch {
      toast({ title: "Error saving profile", variant: "destructive" });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{profile ? "Edit SNMP Profile" : "New SNMP Profile"}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <Label>Name *</Label>
              <Input value={form.name} onChange={(e) => set("name", e.target.value)} placeholder="BDCOM EPON" />
            </div>
            <div className="space-y-1">
              <Label>Vendor *</Label>
              <Input value={form.vendor} onChange={(e) => set("vendor", e.target.value)} placeholder="BDCOM" />
            </div>
            <div className="space-y-1">
              <Label>Technology</Label>
              <Select value={form.technology} onValueChange={(v) => set("technology", v as "EPON" | "GPON")}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="EPON">EPON</SelectItem>
                  <SelectItem value="GPON">GPON</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>Description</Label>
              <Input value={form.description} onChange={(e) => set("description", e.target.value)} />
            </div>
          </div>

          <div className="space-y-2">
            <Label>OID Map</Label>
            <p className="text-xs text-slate-500">
              Define the OID for each attribute. Use <code>index_port_pos</code> and <code>index_onu_pos</code> to specify which dot-segment of the SNMP index encodes the port and ONU slot (0-based).
            </p>
            <div className="border rounded-lg p-3">
              <OIDMapEditor value={form.oid_map} onChange={(v) => set("oid_map", v)} />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
            {profile ? "Update" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Profile Card ──────────────────────────────────────────────────────────────

function ProfileCard({
  profile,
  onEdit,
  onDelete,
}: {
  profile: SNMPProfile;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const oidEntries = Object.entries(profile.oid_map);

  return (
    <div className="border rounded-lg overflow-hidden">
      <div className="flex items-center justify-between p-4">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-purple-100 flex items-center justify-center">
            <Shield className="w-4 h-4 text-purple-600" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <p className="font-medium text-sm">{profile.name}</p>
              {profile.is_default && (
                <span className="text-xs bg-blue-100 text-blue-700 px-1.5 py-0.5 rounded">Default</span>
              )}
            </div>
            <p className="text-xs text-slate-500">{profile.vendor} · {profile.technology} · {oidEntries.length} OIDs</p>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" className="w-8 h-8" onClick={() => setExpanded((v) => !v)}>
            {expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </Button>
          <Button variant="ghost" size="icon" className="w-8 h-8" onClick={onEdit}>
            <Edit className="w-4 h-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="w-8 h-8 text-red-400 hover:text-red-600"
            onClick={onDelete}
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      </div>

      {expanded && (
        <div className="border-t bg-slate-50 p-4">
          {profile.description && (
            <p className="text-sm text-slate-500 mb-3">{profile.description}</p>
          )}
          <div className="grid grid-cols-[1fr_2fr] gap-x-4 gap-y-1 text-xs">
            <div className="font-medium text-slate-400 uppercase tracking-wide">Key</div>
            <div className="font-medium text-slate-400 uppercase tracking-wide">OID</div>
            {oidEntries.map(([k, v]) => (
              <>
                <div key={k + "k"} className="text-slate-600 font-mono py-0.5">{k}</div>
                <div key={k + "v"} className="font-mono text-slate-700 py-0.5 truncate">{v || <span className="text-slate-300">—</span>}</div>
              </>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ── Main Page ────────────────────────────────────────────────────────────────

export default function SNMPProfilesPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingProfile, setEditingProfile] = useState<SNMPProfile | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<SNMPProfile | null>(null);

  const { toast } = useToast();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery<{ data: { data: SNMPProfile[] } }>({
    queryKey: ["snmp-profiles"],
    queryFn: () => snmpProfilesApi.list(),
  });

  const profiles = data?.data?.data ?? [];

  const deleteMutation = useMutation({
    mutationFn: (id: string) => snmpProfilesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["snmp-profiles"] });
      toast({ title: "Profile deleted" });
      setDeleteTarget(null);
    },
    onError: () => toast({ title: "Delete failed", variant: "destructive" }),
  });

  return (
    <div className="flex flex-col h-full">
      <Topbar title="SNMP Profiles" />
      <div className="flex-1 overflow-auto p-6">
        <div className="flex items-center justify-between mb-4">
          <p className="text-sm text-slate-500">
            Vendor-specific OID mappings for SNMP polling.
          </p>
          <Button onClick={() => { setEditingProfile(null); setDialogOpen(true); }} className="gap-2">
            <Plus className="w-4 h-4" /> New Profile
          </Button>
        </div>

        {isLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => <Skeleton key={i} className="h-20 rounded-lg" />)}
          </div>
        ) : profiles.length === 0 ? (
          <div className="text-center py-16 text-slate-400">
            <Shield className="w-10 h-10 mx-auto mb-3 opacity-30" />
            <p>No SNMP profiles yet. Create one to get started.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {profiles.map((profile) => (
              <ProfileCard
                key={profile.id}
                profile={profile}
                onEdit={() => { setEditingProfile(profile); setDialogOpen(true); }}
                onDelete={() => setDeleteTarget(profile)}
              />
            ))}
          </div>
        )}
      </div>

      <ProfileFormDialog
        open={dialogOpen}
        onClose={() => { setDialogOpen(false); setEditingProfile(null); }}
        profile={editingProfile}
        onSaved={() => queryClient.invalidateQueries({ queryKey: ["snmp-profiles"] })}
      />

      <AlertDialog open={!!deleteTarget} onOpenChange={(v) => !v && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete profile?</AlertDialogTitle>
            <AlertDialogDescription>
              Delete <strong>{deleteTarget?.name}</strong>? OLTs using this profile will lose their SNMP configuration.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-red-600 hover:bg-red-700"
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
