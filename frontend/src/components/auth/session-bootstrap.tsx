import { type ReactNode, useEffect } from "react";
import { getMe, refreshToken } from "@/api/auth";
import { useAuthStore } from "@/stores/auth";

const RENEW_THRESHOLD_MS = 24 * 60 * 60 * 1000;
const CHECK_INTERVAL_MS = 30 * 60 * 1000;

export function SessionBootstrap({ children }: { children: ReactNode }) {
  const token = useAuthStore((state) => state.token);
  const expiresAt = useAuthStore((state) => state.expiresAt);
  const setSession = useAuthStore((state) => state.setSession);
  const setUser = useAuthStore((state) => state.setUser);
  const clearSession = useAuthStore((state) => state.clearSession);

  useEffect(() => {
    if (!token) {
      return;
    }

    let disposed = false;

    async function restoreAndRenew() {
      try {
        const shouldRenew =
          !expiresAt ||
          new Date(expiresAt).getTime() - Date.now() <= RENEW_THRESHOLD_MS;

        if (shouldRenew) {
          const refreshed = await refreshToken();
          if (!disposed) {
            setSession(refreshed.token, refreshed.expiresAt, refreshed.user);
          }
          return;
        }

        const user = await getMe();
        if (!disposed) {
          setUser(user);
        }
      } catch {
        if (!disposed) {
          clearSession();
        }
      }
    }

    restoreAndRenew();
    const intervalID = window.setInterval(restoreAndRenew, CHECK_INTERVAL_MS);

    return () => {
      disposed = true;
      window.clearInterval(intervalID);
    };
  }, [clearSession, expiresAt, setSession, setUser, token]);

  return children;
}
