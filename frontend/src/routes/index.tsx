import { Activity, Database, KeyRound, Share2 } from "lucide-react";
import type { ReactNode } from "react";
import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppLayout } from "@/components/layout/app-layout";
import { DashboardPage } from "@/pages/dashboard";
import { LoginPage } from "@/pages/login";
import { NotFoundPage } from "@/pages/not-found";
import { RegisterPage } from "@/pages/register";
import { ResourcePlaceholder } from "@/pages/resource-placeholder";
import { ServersPage } from "@/pages/servers";
import { useAuthStore } from "@/stores/auth";

function RequireAuth({ children }: { children: ReactNode }) {
  const token = useAuthStore((state) => state.token);
  if (!token) {
    return <Navigate replace to="/login" />;
  }
  return children;
}

export const router = createBrowserRouter([
  {
    path: "/login",
    element: <LoginPage />,
  },
  {
    path: "/register",
    element: <RegisterPage />,
  },
  {
    path: "/",
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      {
        index: true,
        element: <DashboardPage />,
      },
      {
        path: "servers",
        element: <ServersPage />,
      },
      {
        path: "nodes",
        element: (
          <ResourcePlaceholder
            actions={[
              "选择协议并安装节点",
              "导入外部已安装节点",
              "卸载系统安装节点",
              "查看节点安装状态",
            ]}
            description="管理系统安装节点和外部导入节点。"
            icon={Activity}
            title="协议节点"
          />
        ),
      },
      {
        path: "subscriptions",
        element: (
          <ResourcePlaceholder
            actions={[
              "创建多节点订阅",
              "选择客户端订阅格式",
              "启用或禁用订阅",
              "重置订阅 token",
            ]}
            description="为 sing-box、Mihomo、v2rayN 等客户端生成订阅链接。"
            icon={Share2}
            title="订阅管理"
          />
        ),
      },
      {
        path: "tasks",
        element: (
          <ResourcePlaceholder
            actions={[
              "查看安装任务状态",
              "查看卸载任务状态",
              "查看 SSH 测试日志",
              "追踪脚本执行输出",
            ]}
            description="查看安装、卸载和 SSH 连通性测试任务。"
            icon={Database}
            title="任务日志"
          />
        ),
      },
      {
        path: "security",
        element: (
          <ResourcePlaceholder
            actions={[
              "SSH 凭据脱敏展示",
              "敏感协议参数脱敏",
              "订阅 token 重置",
              "用户数据隔离校验",
            ]}
            description="承载凭据安全、token 和访问控制相关能力。"
            icon={KeyRound}
            title="安全中心"
          />
        ),
      },
    ],
  },
  {
    path: "*",
    element: <NotFoundPage />,
  },
]);
