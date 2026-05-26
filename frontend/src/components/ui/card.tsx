import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "rounded-xl border border-white/[0.04] bg-[#0d0f18]/85 backdrop-blur-md shadow-[0_8px_30px_rgb(0,0,0,0.5)] transition-all duration-300 hover:border-white/[0.08] hover:shadow-[0_8px_30px_rgba(99,102,241,0.02)]",
        className,
      )}
      {...props}
    />
  );
}

export function CardHeader({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("border-white/[0.04] border-b p-6", className)}
      {...props}
    />
  );
}

export function CardContent({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("p-6", className)} {...props} />;
}
