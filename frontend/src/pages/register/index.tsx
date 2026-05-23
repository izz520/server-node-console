import { UserPlus } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { register } from "@/api/auth";
import { getErrorMessage } from "@/api/errors";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { APP_NAME } from "@/constants/config";
import { useAuthStore } from "@/stores/auth";

export function RegisterPage() {
  const navigate = useNavigate();
  const setSession = useAuthStore((state) => state.setSession);
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setIsSubmitting(true);
    try {
      const response = await register({ username, email, password });
      setSession(response.token, response.expiresAt, response.user);
      navigate("/");
    } catch (submitError) {
      setError(
        getErrorMessage(
          submitError,
          "注册失败，请检查用户名、邮箱或密码是否已被使用",
        ),
      );
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
              <UserPlus className="h-5 w-5" />
            </div>
            <div>
              <h1 className="font-semibold text-lg text-slate-950">
                {APP_NAME}
              </h1>
              <p className="text-slate-500 text-sm">注册后开始管理节点订阅</p>
            </div>
          </div>
          <form className="space-y-4" onSubmit={handleSubmit}>
            <label className="block" htmlFor="username">
              <span className="mb-1 block text-slate-700 text-sm">用户名</span>
              <Input
                autoComplete="username"
                id="username"
                onChange={(event) => setUsername(event.target.value)}
                placeholder="alice"
                required
                value={username}
              />
            </label>
            <label className="block" htmlFor="email">
              <span className="mb-1 block text-slate-700 text-sm">邮箱</span>
              <Input
                autoComplete="email"
                id="email"
                onChange={(event) => setEmail(event.target.value)}
                placeholder="alice@example.com"
                required
                type="email"
                value={email}
              />
            </label>
            <label className="block" htmlFor="new-password">
              <span className="mb-1 block text-slate-700 text-sm">密码</span>
              <Input
                autoComplete="new-password"
                id="new-password"
                minLength={8}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="至少 8 位"
                required
                type="password"
                value={password}
              />
            </label>
            {error && <p className="text-red-600 text-sm">{error}</p>}
            <Button className="w-full" disabled={isSubmitting} type="submit">
              {isSubmitting ? "注册中..." : "注册"}
            </Button>
          </form>
          <p className="mt-4 text-slate-500 text-xs">
            已有账号？{" "}
            <Link
              className="font-medium text-slate-950 hover:underline"
              to="/login"
            >
              返回登录
            </Link>
          </p>
        </CardContent>
      </Card>
    </main>
  );
}
