"use client";

import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { routersApi } from "@/lib/api";
import { Router } from "@/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { useToast } from "@/components/ui/use-toast";

const schema = z.object({
  name: z.string().min(1, "Name is required"),
  ip_address: z.string().min(1, "IP address is required"),
  api_port: z.coerce.number().min(1).max(65535).default(8728),
  username: z.string().min(1, "Username is required"),
  password: z.string().optional(),
  location: z.string().optional(),
  pop_name: z.string().optional(),
  description: z.string().optional(),
  sync_interval: z.coerce.number().min(0).default(60),
});

type FormData = z.infer<typeof schema>;

interface Props {
  open: boolean;
  onClose: () => void;
  router?: Router | null;
}

export function RouterFormDialog({ open, onClose, router }: Props) {
  const isEdit = !!router;
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { api_port: 8728, sync_interval: 60 },
  });

  useEffect(() => {
    if (open) {
      reset(
        router
          ? {
              name: router.name,
              ip_address: router.ip_address,
              api_port: router.api_port,
              username: router.username,
              location: router.location ?? "",
              pop_name: router.pop_name ?? "",
              description: router.description ?? "",
              sync_interval: router.sync_interval ?? 60,
            }
          : { api_port: 8728, sync_interval: 60 }
      );
    }
  }, [open, router, reset]);

  const mutation = useMutation({
    mutationFn: (data: FormData) =>
      isEdit ? routersApi.update(router!.id, data) : routersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["routers"] });
      toast({ title: isEdit ? "Router updated" : "Router created" });
      onClose();
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        "Operation failed";
      toast({ title: "Error", description: msg, variant: "destructive" });
    },
  });

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit Router" : "Add Router"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit((d) => mutation.mutate(d))} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="col-span-2 space-y-1.5">
              <Label>Router Name *</Label>
              <Input placeholder="e.g. Router-Main-POP1" {...register("name")} />
              {errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
            </div>
            <div className="space-y-1.5">
              <Label>IP Address *</Label>
              <Input placeholder="192.168.1.1" {...register("ip_address")} />
              {errors.ip_address && (
                <p className="text-xs text-destructive">{errors.ip_address.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>API Port</Label>
              <Input type="number" {...register("api_port")} />
            </div>
            <div className="space-y-1.5">
              <Label>Username *</Label>
              <Input placeholder="admin" {...register("username")} />
              {errors.username && (
                <p className="text-xs text-destructive">{errors.username.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>{isEdit ? "Password (leave blank to keep)" : "Password *"}</Label>
              <Input type="password" placeholder="••••••••" {...register("password")} />
            </div>
            <div className="space-y-1.5">
              <Label>Location</Label>
              <Input placeholder="Dhaka" {...register("location")} />
            </div>
            <div className="space-y-1.5">
              <Label>POP Name</Label>
              <Input placeholder="POP-01" {...register("pop_name")} />
            </div>
            <div className="space-y-1.5">
              <Label>Auto-Sync Interval (minutes)</Label>
              <Input type="number" min={0} placeholder="60" {...register("sync_interval")} />
              <p className="text-xs text-muted-foreground">0 = manual only; default 60</p>
            </div>
            <div className="col-span-2 space-y-1.5">
              <Label>Description</Label>
              <Input placeholder="Optional notes…" {...register("description")} />
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting || mutation.isPending}>
              {(isSubmitting || mutation.isPending) && (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              )}
              {isEdit ? "Save Changes" : "Create Router"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
