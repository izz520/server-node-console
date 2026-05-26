import { LockKeyhole } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { login } from "@/api/auth";
import { getErrorMessage } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { APP_NAME } from "@/constants/config";
import { useAuthStore } from "@/stores/auth";

export function LoginPage() {
  const navigate = useNavigate();
  const setSession = useAuthStore((state) => state.setSession);
  const [account, setAccount] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setIsSubmitting(true);
    try {
      const response = await login({ account, password });
      setSession(response.token, response.expiresAt, response.user);
      navigate("/");
    } catch (submitError) {
      setError(getErrorMessage(submitError, "账号或密码不正确"));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="relative flex min-h-screen items-center justify-center bg-[#07080e] px-4 overflow-hidden">
      {/* High-End Background Dots */}
      <div className="absolute inset-0 bg-[radial-gradient(rgba(255,255,255,0.05)_1px,transparent_1px)] [background-size:24px_24px] pointer-events-none" />

      {/* Subtle Space Glow Orbs */}
      <div className="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 h-[380px] w-[380px] rounded-full bg-indigo-500/5 blur-[120px] pointer-events-none" />

      <Card className="relative z-10 w-full max-w-sm border-white/[0.04] bg-[#0d0f18]/85 backdrop-blur-md shadow-[0_20px_50px_rgba(0,0,0,0.6)]">
        <CardContent className="p-8">
          <div className="mb-8 flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-white/[0.08] bg-slate-900 text-slate-100 shadow-inner shrink-0">
              <LockKeyhole className="h-4 w-4 text-[#6366f1]" />
            </div>
            <div>
              <h1 className="font-bold text-slate-100 text-base tracking-tight font-display">
                {APP_NAME}
              </h1>
              <p className="text-slate-500 text-[10px] font-semibold uppercase tracking-wider mt-0.5">
                Multi-Node Core Console
              </p>
            </div>
          </div>

          <form className="space-y-4" onSubmit={handleSubmit}>
            <label className="block" htmlFor="account">
              <span className="mb-1.5 block text-slate-500 text-[9px] font-bold uppercase tracking-widest">
                用户名或电子邮箱
              </span>
              <Input
                autoComplete="username"
                id="account"
                onChange={(event) => setAccount(event.target.value)}
                placeholder="admin@example.com"
                required
                value={account}
              />
            </label>
            <label className="block" htmlFor="password">
              <span className="mb-1.5 block text-slate-500 text-[9px] font-bold uppercase tracking-widest">
                密码
              </span>
              <Input
                autoComplete="current-password"
                id="password"
                onChange={(event) => setPassword(event.target.value)}
                placeholder="请输入登录密码"
                required
                type="password"
                value={password}
              />
            </label>

            {error && (
              <p className="text-rose-400 text-xs font-semibold bg-red-500/5 border border-red-500/10 px-3.5 py-2 rounded-lg">
                {error}
              </p>
            )}

            <Button
              className="w-full mt-3 h-10 bg-white text-slate-950 hover:bg-slate-100 font-bold"
              disabled={isSubmitting}
              type="submit"
            >
              {isSubmitting ? "正在验证凭据..." : "安全登录系统"}
            </Button>
          </form>

          <p className="mt-6 text-slate-500 text-[11px] font-semibold text-center">
            还没有系统账号？{" "}
            <Link
              className="font-bold text-slate-200 hover:text-white hover:underline transition-all duration-200"
              to="/register"
            >
              注册新账号
            </Link>
          </p>
        </CardContent>
      </Card>
    </main>
  );
}
