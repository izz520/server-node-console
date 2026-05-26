import type { InputHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export function Input({
  className,
  ...props
}: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={cn(
        "h-9 w-full rounded-lg border border-white/[0.06] bg-[#090b11] px-3.5 text-xs text-slate-100 placeholder-slate-600 outline-none transition-all duration-200 focus:border-[#4f46e5] focus:bg-[#0e111a] focus:shadow-[0_0_12px_rgba(99,102,241,0.06)] disabled:opacity-40 disabled:pointer-events-none",
        className,
      )}
      {...props}
    />
  );
}
