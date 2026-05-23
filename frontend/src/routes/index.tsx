import type { ReactNode } from "react";
import { createBrowserRouter, Navigate } from "react-router-dom";
import { AppLayout } from "@/components/layout/app-layout";
import { AdminPage } from "@/pages/admin";
import { DashboardPage } from "@/pages/dashboard";
import { LoginPage } from "@/pages/login";
import { NodesPage } from "@/pages/nodes";
import { NotFoundPage } from "@/pages/not-found";
import { RegisterPage } from "@/pages/register";
import { SecurityPage } from "@/pages/security";
import { ServersPage } from "@/pages/servers";
import { SubscriptionsPage } from "@/pages/subscriptions";
import { TasksPage } from "@/pages/tasks";
import { useAuthStore } from "@/stores/auth";

function RequireAuth({ children }: { children: ReactNode }) {
  const token = useAuthStore((state) => state.token);
  if (!token) {
    return <Navigate replace to="/login" />;
  }
  return children;
}

function RequireAdmin({ children }: { children: ReactNode }) {
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  if (token && !user) {
    return null;
  }
  if (user?.role !== "admin") {
    return <Navigate replace to="/" />;
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
        element: <NodesPage />,
      },
      {
        path: "subscriptions",
        element: <SubscriptionsPage />,
      },
      {
        path: "tasks",
        element: <TasksPage />,
      },
      {
        path: "security",
        element: <SecurityPage />,
      },
      {
        path: "admin",
        element: (
          <RequireAdmin>
            <AdminPage />
          </RequireAdmin>
        ),
      },
    ],
  },
  {
    path: "*",
    element: <NotFoundPage />,
  },
]);
