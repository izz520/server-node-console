import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export function Badge({
  className,
  ...props
}: HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full border border-white/[0.04] bg-[#0e1017] px-2.5 py-0.5 text-slate-300 text-[10px] font-semibold tracking-wide shadow-inner select-none",
        className,
      )}
      {...props}
    />
  );
}
