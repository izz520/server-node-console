import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
      <div className="font-semibold text-4xl text-slate-950">404</div>
      <p className="mt-3 text-slate-600">页面不存在或还没有实现。</p>
      <Link
        className="mt-6 inline-flex h-10 items-center justify-center rounded-md bg-slate-950 px-4 font-medium text-sm text-white transition hover:bg-slate-800"
        to="/"
      >
        回到控制台
      </Link>
    </div>
  );
}
