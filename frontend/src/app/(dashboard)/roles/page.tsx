"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Edit, Trash2, Shield, ChevronDown, ChevronRight } from "lucide-react";
import { rolesApi } from "@/lib/api";
import { Role, Permission } from "@/types";
import { Topbar } from "@/components/layout/Topbar";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useToast } from "@/components/ui/use-toast";
import { RoleFormDialog } from "@/components/roles/RoleFormDialog";
import { PermissionMatrix } from "@/components/roles/PermissionMatrix";

export default function RolesPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Role | null>(null);
  const [expandedRole, setExpandedRole] = useState<string | null>(null);

  const queryClient = useQueryClient();
  const { toast } = useToast();

  const { data: rolesData, isLoading: rolesLoading } = useQuery<{ data: { data: Role[] } }>({
    queryKey: ["roles-list"],
    queryFn: rolesApi.list,
  });

  const { data: permsData } = useQuery<{ data: { data: Permission[] } }>({
    queryKey: ["permissions-list"],
    queryFn: rolesApi.permissions,
  });

  const roles = rolesData?.data?.data ?? [];
  const allPermissions = permsData?.data?.data ?? [];

  const deleteMutation = useMutation({
    mutationFn: (id: string) => rolesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles-list"] });
      toast({ title: "Role deleted" });
      setDeleteTarget(null);
    },
    onError: () => toast({ title: "Delete failed", variant: "destructive" }),
  });

  const LOCKED_ROLES = ["super_admin", "admin", "operator", "viewer"];

  return (
    <div>
      <Topbar
        title="Roles & Permissions"
        breadcrumbs={[{ label: "Home" }, { label: "Roles & Permissions" }]}
      />

      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">{roles.length} roles configured</p>
          <Button onClick={() => { setEditingRole(null); setDialogOpen(true); }}>
            <Plus className="w-4 h-4 mr-1.5" /> New Role
          </Button>
        </div>

        {rolesLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-20 w-full" />
            ))}
          </div>
        ) : (
          <div className="space-y-3">
            {roles.map((role) => (
              <Card key={role.id} className="overflow-hidden">
                <CardHeader className="py-4 px-5">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className="w-9 h-9 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                        <Shield className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                      </div>
                      <div>
                        <CardTitle className="text-sm font-semibold">{role.display_name}</CardTitle>
                        <p className="text-xs text-muted-foreground mt-0.5">
                          {role.name} • {role.permissions?.length ?? 0} permissions
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      {role.description && (
                        <p className="text-xs text-muted-foreground hidden md:block">
                          {role.description}
                        </p>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          setExpandedRole(expandedRole === role.id ? null : role.id)
                        }
                      >
                        {expandedRole === role.id ? (
                          <ChevronDown className="w-4 h-4" />
                        ) : (
                          <ChevronRight className="w-4 h-4" />
                        )}
                        Permissions
                      </Button>
                      {!LOCKED_ROLES.includes(role.name) && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => { setEditingRole(role); setDialogOpen(true); }}
                          >
                            <Edit className="w-4 h-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setDeleteTarget(role)}
                            className="text-destructive hover:text-destructive"
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                </CardHeader>

                {expandedRole === role.id && (
                  <CardContent className="pb-5 pt-0 px-5 border-t">
                    <PermissionMatrix
                      role={role}
                      allPermissions={allPermissions}
                    />
                  </CardContent>
                )}
              </Card>
            ))}
          </div>
        )}
      </div>

      <RoleFormDialog open={dialogOpen} onClose={() => setDialogOpen(false)} role={editingRole} />

      <AlertDialog open={!!deleteTarget} onOpenChange={(v) => !v && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Role</AlertDialogTitle>
            <AlertDialogDescription>
              Delete <strong>{deleteTarget?.display_name}</strong>? Users with this role will lose
              its permissions.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteTarget && deleteMutation.mutate(deleteTarget.id)}
              className="bg-destructive hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
