"use client";

import { useState, useRef, KeyboardEvent } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { X, Plus } from "lucide-react";
import { rolesApi } from "@/lib/api";
import { Role } from "@/types";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/use-toast";

interface Props {
  role: Role;
}

export function AccountPrefixEditor({ role }: Props) {
  const [prefixes, setPrefixes] = useState<string[]>(role.account_prefixes ?? []);
  const [input, setInput] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const queryClient = useQueryClient();
  const { toast } = useToast();

  const mutation = useMutation({
    mutationFn: (p: string[]) => rolesApi.setAccountPrefixes(role.id, p),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["roles-list"] });
      toast({ title: "Account prefixes saved" });
    },
    onError: () => toast({ title: "Failed to save prefixes", variant: "destructive" }),
  });

  const addPrefix = () => {
    const val = input.trim();
    if (!val || prefixes.includes(val)) { setInput(""); return; }
    setPrefixes((prev) => [...prev, val]);
    setInput("");
    inputRef.current?.focus();
  };

  const removePrefix = (p: string) => setPrefixes((prev) => prev.filter((x) => x !== p));

  const handleKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") { e.preventDefault(); addPrefix(); }
    if (e.key === "Backspace" && input === "" && prefixes.length > 0) {
      setPrefixes((prev) => prev.slice(0, -1));
    }
  };

  const isDirty =
    input.trim() !== "" ||
    JSON.stringify(prefixes) !== JSON.stringify(role.account_prefixes ?? []);

  return (
    <div className="space-y-3">
      <div>
        <p className="text-xs font-medium text-muted-foreground mb-1.5">
          Account Prefix Filter
        </p>
        <p className="text-xs text-muted-foreground">
          Users with this role can only see Internet Accounts whose username starts with
          one of these prefixes. Leave empty to block all accounts.
        </p>
      </div>

      {/* Tag display */}
      <div
        className="flex flex-wrap gap-1.5 min-h-[36px] p-2 rounded-md border bg-background cursor-text"
        onClick={() => inputRef.current?.focus()}
      >
        {prefixes.map((p) => (
          <Badge key={p} variant="secondary" className="gap-1 pr-1 text-xs">
            {p}
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); removePrefix(p); }}
              className="ml-0.5 rounded-sm opacity-60 hover:opacity-100 focus:outline-none"
            >
              <X className="w-3 h-3" />
            </button>
          </Badge>
        ))}
        <input
          ref={inputRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKey}
          placeholder={prefixes.length === 0 ? "Type prefix, press Enter or click Save…" : "Add another…"}
          className="flex-1 min-w-[120px] bg-transparent outline-none text-xs placeholder:text-muted-foreground"
        />
      </div>

      <div className="flex items-center gap-2">
        <Button
          size="sm"
          onClick={() => {
            // Auto-add any pending input before saving
            const val = input.trim();
            const final = val && !prefixes.includes(val) ? [...prefixes, val] : prefixes;
            if (val) { setPrefixes(final); setInput(""); }
            mutation.mutate(final);
          }}
          disabled={!isDirty || mutation.isPending}
        >
          {mutation.isPending ? "Saving…" : "Save Prefixes"}
        </Button>
        {isDirty && (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => setPrefixes(role.account_prefixes ?? [])}
          >
            Reset
          </Button>
        )}
      </div>
    </div>
  );
}
