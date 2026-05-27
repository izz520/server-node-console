import { AlertCircle, CheckCircle2, Info, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";
import { type ToastMessage, useToastStore } from "@/stores/toast";

function ToastItem({
  toast,
  onRemove,
}: {
  toast: ToastMessage;
  onRemove: (id: string) => void;
}) {
  const [state, setState] = useState<"entering" | "visible" | "exiting">(
    "entering",
  );
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    // Trigger enter animation on next frame
    const raf = requestAnimationFrame(() => setState("visible"));
    return () => cancelAnimationFrame(raf);
  }, []);

  useEffect(() => {
    // Auto-exit 3.6s after mount (reserve 400ms for exit animation)
    timerRef.current = setTimeout(() => {
      setState("exiting");
    }, 3600);
    return () => clearTimeout(timerRef.current);
  }, []);

  useEffect(() => {
    if (state === "exiting") {
      const t = setTimeout(() => onRemove(toast.id), 400);
      return () => clearTimeout(t);
    }
  }, [state, toast.id, onRemove]);

  const handleClose = () => {
    clearTimeout(timerRef.current);
    setState("exiting");
  };

  const Icon =
    toast.type === "success"
      ? CheckCircle2
      : toast.type === "error"
        ? AlertCircle
        : Info;

  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-xl border p-4 shadow-2xl backdrop-blur-xl pointer-events-auto",
        toast.type === "success" &&
          "border-emerald-500/20 bg-[#0c1913]/90 text-emerald-300",
        toast.type === "error" &&
          "border-rose-500/20 bg-[#1d0e11]/90 text-rose-300",
        toast.type === "info" &&
          "border-white/[0.06] bg-[#0d0f18]/90 text-slate-200",
        toast.type === "warning" &&
          "border-amber-500/20 bg-[#1a1608]/90 text-amber-300",
      )}
      style={{
        transition:
          "transform 0.4s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.4s cubic-bezier(0.16, 1, 0.3, 1)",
        transform:
          state === "entering"
            ? "translateX(100%) scale(0.95)"
            : state === "exiting"
              ? "translateX(60%) scale(0.95)"
              : "translateX(0) scale(1)",
        opacity: state === "visible" ? 1 : 0,
      }}
    >
      <Icon className="h-4.5 w-4.5 shrink-0 mt-0.5" />
      <div className="flex-1 text-xs font-semibold leading-relaxed">
        {toast.message}
      </div>
      <button
        onClick={handleClose}
        className="rounded-lg p-1 text-slate-400 hover:bg-white/5 hover:text-slate-100 transition-colors cursor-pointer"
        type="button"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

export function Toaster() {
  const toasts = useToastStore((state) => state.toasts);
  const removeToast = useToastStore((state) => state.removeToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-6 right-6 z-[100] flex flex-col gap-3 w-full max-w-sm pointer-events-none">
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onRemove={removeToast} />
      ))}
    </div>
  );
}
