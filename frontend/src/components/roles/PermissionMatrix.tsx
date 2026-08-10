"use client";

import { useState, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { rolesApi } from "@/lib/api";
import { Role, Permission } from "@/types";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/use-toast";
import { cn } from "@/lib/utils";

const ACTIONS = ["view", "create", "update", "delete"];

const MODULE_LABELS: Record<string, string> = {
  profile_mappings: "Profile Mappings",
};

function moduleLabel(mod: string): string {
  return MODULE_LABELS[mod] ?? mod.charAt(0).toUpperCase() + mod.slice(1);
}

interface Props {
  role: Role;
  allPermissions: Permission[];
}

export function PermissionMatrix({ role, allPermissions }: Props) {
  const qc = useQueryClient();
  const { toast } = useToast();

  const modules = Array.from(new Set(allPermissions.map((p) => p.module))).sort();

  const [selected, setSelected] = useState<Set<string>>(
    new Set(role.permissions?.map((p) => p.id) ?? [])
  );

  useEffect(() => {
    setSelected(new Set(role.permissions?.map((p) => p.id) ?? []));
  }, [role]);

  const mutation = useMutation({
    mutationFn: () =>
      rolesApi.setPermissions(role.id, Array.from(selected)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["roles-list"] });
      toast({ title: "Permissions saved" });
    },
    onError: () => toast({ title: "Failed to save", variant: "destructive" }),
  });

  const toggle = (permId: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(permId) ? next.delete(permId) : next.add(permId);
      return next;
    });
  };

  const getPermId = (module: string, action: string) =>
    allPermissions.find((p) => p.module === module && p.action === action)?.id;

  return (
    <div className="mt-4 space-y-3">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="text-left text-muted-foreground font-medium pb-2 w-40">Module</th>
              {ACTIONS.map((a) => (
                <th key={a} className="text-center text-muted-foreground font-medium pb-2 capitalize">
                  {a}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {modules.map((mod) => (
              <tr key={mod} className="border-t">
                <td className="py-2 font-medium">{moduleLabel(mod)}</td>
                {ACTIONS.map((action) => {
                  const permId = getPermId(mod, action);
                  const checked = permId ? selected.has(permId) : false;
                  return (
                    <td key={action} className="text-center py-2">
                      {permId ? (
                        <button
                          onClick={() => toggle(permId)}
                          className={cn(
                            "w-5 h-5 rounded border-2 transition-colors mx-auto flex items-center justify-center",
                            checked
                              ? "bg-blue-600 border-blue-600"
                              : "border-muted-foreground/30 hover:border-blue-400"
                          )}
                        >
                          {checked && (
                            <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                            </svg>
                          )}
                        </button>
                      ) : (
                        <span className="text-muted-foreground/30">—</span>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex justify-end pt-2">
        <Button size="sm" onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          {mutation.isPending && <Loader2 className="w-3.5 h-3.5 mr-1.5 animate-spin" />}
          Save Permissions
        </Button>
      </div>
    </div>
  );
}
