"use client";

import * as React from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

interface SheetProps {
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  children?: React.ReactNode;
}

function Sheet({ open, onOpenChange, children }: SheetProps) {
  React.useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape" && open) onOpenChange?.(false);
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onOpenChange]);

  if (!open) return null;

  return (
    <>
      {/* Overlay */}
      <div
        className="fixed inset-0 z-50 bg-black/40 backdrop-blur-sm"
        onClick={() => onOpenChange?.(false)}
      />
      {children}
    </>
  );
}

interface SheetContentProps {
  className?: string;
  children?: React.ReactNode;
  side?: "right" | "left";
}

function SheetContent({ className, children, side = "right" }: SheetContentProps) {
  return (
    <div
      className={cn(
        "fixed top-0 z-50 h-full bg-background shadow-xl flex flex-col",
        side === "right" ? "right-0" : "left-0",
        className
      )}
      onClick={(e) => e.stopPropagation()}
    >
      {children}
    </div>
  );
}

interface SheetHeaderProps { className?: string; children?: React.ReactNode }
function SheetHeader({ className, children }: SheetHeaderProps) {
  return (
    <div className={cn("flex flex-col space-y-1.5 p-6 border-b border-border", className)}>
      {children}
    </div>
  );
}

interface SheetTitleProps { className?: string; children?: React.ReactNode }
function SheetTitle({ className, children }: SheetTitleProps) {
  return (
    <h2 className={cn("text-lg font-semibold text-foreground leading-none tracking-tight", className)}>
      {children}
    </h2>
  );
}

export { Sheet, SheetContent, SheetHeader, SheetTitle };
