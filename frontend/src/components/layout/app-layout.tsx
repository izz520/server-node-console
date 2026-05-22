import {
  Activity,
  Database,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Server,
  Share2,
} from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { APP_NAME } from "@/constants/config";
import { useAuthStore } from "@/stores/auth";

const navItems = [
  { label: "概览", icon: LayoutDashboard, to: "/" },
  { label: "服务器", icon: Server, to: "/servers" },
  { label: "协议节点", icon: Activity, to: "/nodes" },
  { label: "订阅", icon: Share2, to: "/subscriptions" },
  { label: "任务日志", icon: Database, to: "/tasks" },
  { label: "安全", icon: KeyRound, to: "/security" },
];

export function AppLayout() {
  const clearSession = useAuthStore((state) => state.clearSession);
  const navigate = useNavigate();

  return (
    <div className="min-h-screen bg-slate-50">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-slate-200 border-r bg-white px-4 py-5 lg:block">
        <div className="mb-8">
          <div className="font-semibold text-lg text-slate-950">{APP_NAME}</div>
          <div className="mt-1 text-slate-500 text-xs">
            SaaS 节点与订阅控制台
          </div>
        </div>
        <nav className="space-y-1">
          {navItems.map((item) => (
            <NavLink
              className={({ isActive }) =>
                [
                  "flex h-10 items-center gap-3 rounded-md px-3 text-sm transition",
                  isActive
                    ? "bg-slate-950 text-white"
                    : "text-slate-600 hover:bg-slate-100 hover:text-slate-950",
                ].join(" ")
              }
              key={item.to}
              to={item.to}
            >
              <item.icon className="h-4 w-4" />
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="lg:pl-64">
        <header className="sticky top-0 z-10 flex h-16 items-center justify-between border-slate-200 border-b bg-white px-4 lg:px-8">
          <div>
            <div className="font-medium text-slate-950">控制台</div>
            <div className="text-slate-500 text-xs">
              管理服务器、协议节点和订阅
            </div>
          </div>
          <Button
            onClick={() => {
              clearSession();
              navigate("/login");
            }}
            variant="secondary"
          >
            <LogOut className="h-4 w-4" />
            退出
          </Button>
        </header>
        <main className="px-4 py-6 lg:px-8">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
