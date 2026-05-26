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

const navItems = [
  { label: "工作概览", icon: LayoutDashboard, to: "/" },
  { label: "物理服务器", icon: Server, to: "/servers" },
  { label: "协议节点", icon: Activity, to: "/nodes" },
  { label: "客户端订阅", icon: Share2, to: "/subscriptions" },
  { label: "任务日志", icon: Database, to: "/tasks" },
  { label: "安全中心", icon: KeyRound, to: "/security" },
];

export function AppLayout() {
  const clearSession = useAuthStore((state) => state.clearSession);
  const user = useAuthStore((state) => state.user);
  const navigate = useNavigate();
  const visibleNavItems =
    user?.role === "admin"
      ? [...navItems, { label: "系统管理", icon: ShieldCheck, to: "/admin" }]
      : navItems;

  return (
    <div className="min-h-screen bg-[#07080e] text-slate-100 flex font-sans">
      {/* Floating Glassmorphic Sidebar for Desktop */}
      <aside className="fixed top-4 left-4 bottom-4 w-60 hidden rounded-2xl border border-white/[0.04] bg-[#0d0f18]/80 backdrop-blur-xl p-5 lg:flex flex-col justify-between z-30 shadow-[0_12px_40px_rgba(0,0,0,0.6)]">
        <div>
          {/* Logo Header */}
          <div className="flex items-center gap-3 px-1.5 py-2.5 mb-7">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.08] bg-slate-900 text-slate-100 shadow-inner">
              <Zap className="h-4 w-4 text-[#6366f1]" />
            </div>
            <div>
              <div className="font-bold text-slate-100 text-sm tracking-tight font-display">
                {APP_NAME}
              </div>
              <div className="mt-0.5 text-slate-500 text-[8px] font-bold uppercase tracking-widest">
                Multi-Node Core
              </div>
            </div>
          </div>

          {/* Navigation Links */}
          <nav className="space-y-1">
            {visibleNavItems.map((item) => (
              <NavLink
                className={({ isActive }) =>
                  [
                    "flex h-9 items-center gap-3 rounded-lg px-3 text-xs font-semibold tracking-wide transition-all duration-200 btn-interactive",
                    isActive
                      ? "bg-white/[0.06] text-white shadow-inner border-l border-white/[0.2]"
                      : "text-slate-400 hover:bg-white/[0.02] hover:text-slate-100 hover:translate-x-0.5",
                  ].join(" ")
                }
                key={item.to}
                to={item.to}
              >
                <item.icon className="h-4 w-4 shrink-0 opacity-70" />
                <span>{item.label}</span>
              </NavLink>
            ))}
          </nav>
        </div>

        {/* Footer User Info Profile */}
        <div className="border-t border-white/[0.04] pt-4.5">
          <div className="flex items-center justify-between gap-3 px-1">
            <div className="flex items-center gap-2.5 min-w-0">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-800 border border-white/[0.06] text-[11px] font-bold text-slate-300 uppercase">
                {user?.username?.substring(0, 2) || "US"}
              </div>
              <div className="min-w-0">
                <div className="text-slate-200 font-semibold text-xs truncate">
                  {user?.username || "Guest"}
                </div>
                <div className="text-slate-500 text-[9px] font-bold uppercase tracking-widest mt-0.5">
                  {user?.role === "admin" ? "系统管理员" : "标准用户"}
                </div>
              </div>
            </div>
            <button
              onClick={() => {
                clearSession();
                navigate("/login");
              }}
              className="p-1.5 rounded-lg border border-white/[0.04] bg-white/[0.02] hover:bg-red-500/10 hover:border-red-500/20 text-slate-400 hover:text-red-400 transition-all duration-200 cursor-pointer"
              title="退出登录"
              type="button"
            >
              <LogOut className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </aside>

      {/* Main Container */}
      <div className="flex-1 lg:pl-68 flex flex-col min-w-0">
        {/* Main Content Area */}
        <main className="px-5 pt-6 pb-28 lg:px-10 lg:pb-10 flex-1">
          <Outlet />
        </main>
      </div>

      {/* Mobile Sticky Bar Navigation */}
      <nav className="fixed inset-x-4 bottom-4 z-40 border border-white/[0.04] bg-[#0d0f18]/90 p-2 shadow-2xl backdrop-blur-xl rounded-2xl lg:hidden">
        <div className="flex gap-1 overflow-x-auto justify-between no-scrollbar">
          {visibleNavItems.map((item) => (
            <NavLink
              className={({ isActive }) =>
                [
                  "flex flex-col items-center justify-center gap-1 rounded-xl px-3 py-2 text-[9px] font-bold transition-all duration-200 min-w-14",
                  isActive
                    ? "bg-white/[0.06] text-white"
                    : "text-slate-400 hover:bg-white/[0.02] hover:text-slate-200",
                ].join(" ")
              }
              key={item.to}
              to={item.to}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              <span className="whitespace-nowrap">{item.label}</span>
            </NavLink>
          ))}
          <button
            onClick={() => {
              clearSession();
              navigate("/login");
            }}
            className="flex flex-col items-center justify-center gap-1 rounded-xl px-3 py-2 text-[9px] font-bold text-slate-400 hover:bg-red-500/10 hover:text-red-400 min-w-14 cursor-pointer"
            type="button"
          >
            <LogOut className="h-4 w-4 shrink-0" />
            <span>退出</span>
          </button>
        </div>
      </nav>
      <Toaster />
    </div>
  );
}
