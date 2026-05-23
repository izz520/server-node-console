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
    <main className="flex min-h-screen items-center justify-center bg-slate-50 px-4">
      <Card className="w-full max-w-sm">
        <CardContent className="p-6">
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-md bg-slate-950 text-white">
              <LockKeyhole className="h-5 w-5" />
            </div>
            <div>
              <h1 className="font-semibold text-lg text-slate-950">
                {APP_NAME}
              </h1>
              <p className="text-slate-500 text-sm">登录后管理服务器和订阅</p>
            </div>
          </div>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <label className="block" htmlFor="account">
              <span className="mb-1 block text-slate-700 text-sm">
                用户名或邮箱
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
              <span className="mb-1 block text-slate-700 text-sm">密码</span>
              <Input
                autoComplete="current-password"
                id="password"
                onChange={(event) => setPassword(event.target.value)}
                placeholder="请输入密码"
                required
                type="password"
                value={password}
              />
            </label>
            {error && <p className="text-red-600 text-sm">{error}</p>}
            <Button className="w-full" disabled={isSubmitting} type="submit">
              {isSubmitting ? "登录中..." : "登录"}
            </Button>
          </form>
          <p className="mt-4 text-slate-500 text-xs">
            还没有账号？{" "}
            <Link
              className="font-medium text-slate-950 hover:underline"
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
