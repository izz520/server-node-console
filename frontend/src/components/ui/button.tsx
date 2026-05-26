import type { ButtonHTMLAttributes } from "react";
import { cn } from "@/lib/utils";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
}

const variants: Record<ButtonVariant, string> = {
  primary:
    "bg-white text-slate-950 hover:bg-slate-100 shadow-[0_4px_12px_rgba(255,255,255,0.06)] active:scale-[0.97] font-semibold tracking-wide border border-transparent cursor-pointer",
  secondary:
    "bg-[#0e1017] border border-white/[0.06] text-slate-300 hover:bg-white/5 hover:text-slate-100 hover:border-white/[0.12] active:scale-[0.97] font-medium cursor-pointer",
  ghost:
    "text-slate-400 hover:bg-white/[0.02] hover:text-slate-100 active:scale-[0.97] cursor-pointer",
  danger:
    "bg-red-500/10 border border-red-500/20 text-red-400 hover:bg-red-500/20 active:scale-[0.97] font-medium cursor-pointer",
};

export function Button({
  className,
  variant = "primary",
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex h-9 items-center justify-center gap-2 rounded-lg px-4 text-xs transition-all duration-200 disabled:pointer-events-none disabled:opacity-40 select-none",
        variants[variant],
        className,
      )}
      type={type}
      {...props}
    />
  );
}
