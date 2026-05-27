import {
  Activity,
  Database,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Server,
  Share2,
  ShieldCheck,
  Zap,
} from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { Toaster } from "@/components/ui/toaster";
import { APP_NAME } from "@/constants/config";
import { useAuthStore } from "@/stores/auth";

interface NavItem {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  to: string;
  adminOnly?: boolean;
}

interface NavSection {
  title: string;
  items: NavItem[];
}

const navSections: NavSection[] = [
  {
    title: "控制中心 / CONTROL",
    items: [
      { label: "工作概览", icon: LayoutDashboard, to: "/" },
      { label: "物理服务器", icon: Server, to: "/servers" },
      { label: "协议节点", icon: Activity, to: "/nodes" },
      { label: "客户端订阅", icon: Share2, to: "/subscriptions" },
    ],
  },
  {
    title: "系统运维 / OPERATIONS",
    items: [
      { label: "任务日志", icon: Database, to: "/tasks" },
      { label: "安全中心", icon: KeyRound, to: "/security" },
      { label: "系统管理", icon: ShieldCheck, to: "/admin", adminOnly: true },
    ],
  },
];

export function AppLayout() {
  const clearSession = useAuthStore((state) => state.clearSession);
  const user = useAuthStore((state) => state.user);
  const navigate = useNavigate();

  // Filter sections and items based on role
  const renderedSections = navSections
    .map((section) => ({
      ...section,
      items: section.items.filter(
        (item) => !item.adminOnly || user?.role === "admin",
      ),
    }))
    .filter((section) => section.items.length > 0);

  // Flattened items for mobile view
  const allNavItems = renderedSections.flatMap((section) => section.items);

  return (
    <div className="min-h-screen bg-[#07080e] text-slate-100 flex font-sans">
      {/* Floating Glassmorphic Sidebar for Desktop */}
      <aside className="fixed top-4 left-4 bottom-4 w-64 hidden rounded-2xl border border-white/[0.04] bg-[#0d0f18]/80 backdrop-blur-xl p-5 lg:flex flex-col justify-between z-30 shadow-[0_12px_40px_rgba(0,0,0,0.65)]">
        <div className="flex flex-col gap-6">
          {/* Logo Header */}
          <div className="group flex items-center gap-3 px-3 py-3 rounded-xl bg-white/[0.01] border border-white/[0.02] hover:border-white/[0.06] hover:bg-white/[0.03] transition-all duration-300 shadow-[0_4px_12px_rgba(0,0,0,0.15)]">
            <div className="relative flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-indigo-500/20 bg-indigo-500/5 text-indigo-400 group-hover:border-indigo-500/40 group-hover:bg-indigo-500/10 transition-all duration-300">
              <Zap className="h-4 w-4 text-indigo-400 group-hover:scale-110 group-hover:rotate-12 transition-all duration-300 drop-shadow-[0_0_6px_rgba(99,102,241,0.4)]" />
              {/* Inner ambient pulse */}
              <span className="absolute -inset-0.5 rounded-xl bg-indigo-500/10 opacity-0 group-hover:opacity-100 blur-sm transition-all duration-300" />
            </div>
            <div>
              <div className="font-bold text-slate-200 text-xs tracking-tight font-display group-hover:text-white transition-colors duration-300">
                {APP_NAME}
              </div>
              <div className="mt-1 flex items-center gap-1.5">
                <span className="relative flex h-1.5 w-1.5">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75" />
                  <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-emerald-500" />
                </span>
                <span className="text-[8px] font-bold text-slate-500 uppercase tracking-widest leading-none">
                  Multi-Node Core
                </span>
              </div>
            </div>
          </div>

          {/* Navigation Sections */}
          <div className="space-y-6">
            {renderedSections.map((section) => (
              <div key={section.title} className="space-y-2">
                <div className="px-3.5 text-[8.5px] font-bold text-slate-500 uppercase tracking-widest">
                  {section.title}
                </div>
                <nav className="space-y-0.5">
                  {section.items.map((item) => (
                    <NavLink
                      className={({ isActive }) =>
                        [
                          "group relative flex h-9.5 items-center gap-3 rounded-xl px-3.5 text-xs font-medium tracking-wide transition-all duration-200 btn-interactive border border-transparent",
                          isActive
                            ? "bg-indigo-500/10 text-indigo-200 border-indigo-500/15 shadow-[inset_0_1px_1px_rgba(255,255,255,0.03)]"
                            : "text-slate-400 hover:bg-white/[0.03] hover:text-slate-100 hover:translate-x-0.5",
                        ].join(" ")
                      }
                      key={item.to}
                      to={item.to}
                    >
                      {({ isActive }) => (
                        <>
                          {/* Active left indicator line */}
                          {isActive && (
                            <span className="absolute left-0 w-1 h-4 rounded-r-md bg-indigo-500 shadow-[0_0_8px_#6366f1]" />
                          )}
                          <item.icon
                            className={[
                              "h-4 w-4 shrink-0 transition-all duration-200",
                              isActive
                                ? "text-indigo-400 drop-shadow-[0_0_4px_rgba(99,102,241,0.4)] opacity-100"
                                : "opacity-60 group-hover:opacity-100 group-hover:text-slate-200 group-hover:scale-105",
                            ].join(" ")}
                          />
                          <span
                            className={
                              isActive ? "font-semibold text-slate-200" : ""
                            }
                          >
                            {item.label}
                          </span>
                        </>
                      )}
                    </NavLink>
                  ))}
                </nav>
              </div>
            ))}
          </div>
        </div>

        {/* Footer User Info Profile */}
        <div className="border-t border-white/[0.06] pt-4 mt-auto">
          <div className="flex items-center justify-between gap-3 p-2 rounded-xl bg-white/[0.01] border border-white/[0.02] hover:border-white/[0.06] hover:bg-white/[0.03] transition-all duration-300">
            <div className="flex items-center gap-2.5 min-w-0">
              <div className="relative flex h-8.5 w-8.5 shrink-0 items-center justify-center rounded-full bg-gradient-to-tr from-indigo-500/20 to-purple-500/20 border border-indigo-500/30 text-[11px] font-bold text-indigo-300 uppercase shadow-inner">
                {user?.username?.substring(0, 2) || "US"}
                {/* Active pulse dot on user avatar */}
                <span className="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full bg-emerald-500 border-2 border-[#0d0f18] shadow-[0_0_6px_#10b981]" />
              </div>
              <div className="min-w-0">
                <div className="text-slate-200 font-semibold text-xs truncate leading-none">
                  {user?.username || "Guest"}
                </div>
                <div className="text-slate-500 text-[8px] font-bold uppercase tracking-widest mt-1.5 leading-none">
                  {user?.role === "admin" ? "系统管理员" : "标准用户"}
                </div>
              </div>
            </div>
            <button
              onClick={() => {
                clearSession();
                navigate("/login");
              }}
              className="p-2 rounded-lg border border-white/[0.04] bg-white/[0.02] hover:bg-red-500/10 hover:border-red-500/20 text-slate-400 hover:text-red-400 transition-all duration-200 cursor-pointer shadow-sm hover:scale-105 active:scale-95"
              title="退出登录"
              type="button"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </aside>

      {/* Main Container */}
      <div className="flex-1 lg:pl-72 flex flex-col min-w-0">
        {/* Main Content Area */}
        <main className="px-5 pt-6 pb-28 lg:px-10 lg:pb-10 flex-1">
          <Outlet />
        </main>
      </div>

      {/* Mobile Sticky Bar Navigation */}
      <nav className="fixed inset-x-4 bottom-4 z-40 border border-white/[0.06] bg-[#0d0f18]/90 p-1.5 shadow-[0_12px_40px_rgba(0,0,0,0.8)] backdrop-blur-xl rounded-2xl lg:hidden">
        <div className="flex gap-1 overflow-x-auto justify-between no-scrollbar items-center">
          {allNavItems.map((item) => (
            <NavLink
              className={({ isActive }) =>
                [
                  "flex flex-col items-center justify-center gap-1 rounded-xl px-2.5 py-1.5 text-[9px] font-semibold transition-all duration-200 min-w-13 flex-1",
                  isActive
                    ? "bg-indigo-500/10 text-indigo-400 shadow-inner"
                    : "text-slate-400 hover:bg-white/[0.02] hover:text-slate-200",
                ].join(" ")
              }
              key={item.to}
              to={item.to}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              <span className="whitespace-nowrap scale-90">{item.label}</span>
            </NavLink>
          ))}
          <button
            onClick={() => {
              clearSession();
              navigate("/login");
            }}
            className="flex flex-col items-center justify-center gap-1 rounded-xl px-2.5 py-1.5 text-[9px] font-semibold text-slate-400 hover:bg-red-500/10 hover:text-red-400 min-w-13 flex-1 cursor-pointer transition-colors duration-200"
            type="button"
          >
            <LogOut className="h-4 w-4 shrink-0" />
            <span className="whitespace-nowrap scale-90">退出</span>
          </button>
        </div>
      </nav>
      <Toaster />
    </div>
  );
}
