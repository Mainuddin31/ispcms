"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Router,
  Users,
  Shield,
  Network,
  Wifi,
  Globe,
  ChevronLeft,
  ChevronRight,
  Package,
  ArrowLeftRight,
  FileText,
  Bell,
  CreditCard,
  Receipt,
  Tag,
  Server,
  Radio,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useState } from "react";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/routers", label: "Routers", icon: Router },
  { href: "/internet-accounts", label: "Internet Accounts", icon: Globe },
  { href: "/pppoe", label: "PPPoE Secrets", icon: Network },
  { href: "/sessions", label: "Active Sessions", icon: Wifi },
  // Billing
  { href: "/packages", label: "Packages", icon: Package, divider: true },
  { href: "/profile-mappings", label: "Profile Mappings", icon: ArrowLeftRight },
  { href: "/subscriptions", label: "Subscriptions", icon: CreditCard },
  { href: "/bills", label: "Bills", icon: FileText },
  { href: "/notifications", label: "Notifications", icon: Bell },
  // Expenses
  { href: "/expenses", label: "Expenses", icon: Receipt, divider: true },
  { href: "/expense-categories", label: "Expense Categories", icon: Tag },
  // Network / OLT
  { href: "/network", label: "Network", icon: Radio, divider: true },
  { href: "/olts", label: "OLTs", icon: Server },
  { href: "/onus", label: "ONU Inventory", icon: Wifi },
  { href: "/snmp-profiles", label: "SNMP Profiles", icon: Network },
  // Admin
  { href: "/users", label: "Users", icon: Users, divider: true },
  { href: "/roles", label: "Roles & Permissions", icon: Shield },
];

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <aside
      className={cn(
        "relative flex flex-col h-full bg-slate-900 border-r border-slate-800 transition-all duration-300",
        collapsed ? "w-16" : "w-64"
      )}
    >
      {/* Logo */}
      <div className="flex items-center gap-3 px-4 py-5 border-b border-slate-800">
        <div className="w-8 h-8 rounded-lg bg-blue-600 flex items-center justify-center shrink-0">
          <Wifi className="w-4 h-4 text-white" />
        </div>
        {!collapsed && (
          <div>
            <p className="text-white font-semibold text-sm leading-none">IBMS</p>
            <p className="text-slate-500 text-xs mt-0.5">ISP Management</p>
          </div>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 py-4 overflow-y-auto">
        <ul className="space-y-1 px-2">
          {navItems.map(({ href, label, icon: Icon, divider }) => {
            const active = pathname === href || pathname.startsWith(href + "/");
            return (
              <li key={href}>
                {divider && !collapsed && (
                  <div className="border-t border-slate-800 my-2" />
                )}
                <Link
                  href={href}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                    active
                      ? "bg-blue-600 text-white"
                      : "text-slate-400 hover:text-white hover:bg-slate-800"
                  )}
                >
                  <Icon className="w-4 h-4 shrink-0" />
                  {!collapsed && <span>{label}</span>}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Collapse toggle */}
      <button
        onClick={() => setCollapsed((v) => !v)}
        className="absolute -right-3 top-20 w-6 h-6 rounded-full bg-slate-700 border border-slate-600 flex items-center justify-center text-slate-400 hover:text-white hover:bg-slate-600 transition-colors z-10"
      >
        {collapsed ? (
          <ChevronRight className="w-3 h-3" />
        ) : (
          <ChevronLeft className="w-3 h-3" />
        )}
      </button>
    </aside>
  );
}
