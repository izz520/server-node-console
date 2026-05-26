import { AlertCircle, CheckCircle2, Info, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useToastStore } from "@/stores/toast";

export function Toaster() {
  const toasts = useToastStore((state) => state.toasts);
  const removeToast = useToastStore((state) => state.removeToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-6 right-6 z-[100] flex flex-col gap-3 w-full max-w-sm pointer-events-none">
      {toasts.map((toast) => {
        const Icon =
          toast.type === "success"
            ? CheckCircle2
            : toast.type === "error"
              ? AlertCircle
              : Info;
        return (
          <div
            key={toast.id}
            className={cn(
              "flex items-start gap-3 rounded-xl border p-4 shadow-2xl backdrop-blur-xl pointer-events-auto transition-all duration-300 animate-in slide-in-from-right-5 fade-in duration-200",
              toast.type === "success" &&
                "border-emerald-500/20 bg-[#0c1913]/90 text-emerald-300",
              toast.type === "error" &&
                "border-rose-500/20 bg-[#1d0e11]/90 text-rose-300",
              toast.type === "info" &&
                "border-white/[0.06] bg-[#0d0f18]/90 text-slate-200",
            )}
          >
            <Icon className="h-4.5 w-4.5 shrink-0 mt-0.5" />
            <div className="flex-1 text-xs font-semibold leading-relaxed">
              {toast.message}
            </div>
            <button
              onClick={() => removeToast(toast.id)}
              className="rounded-lg p-1 text-slate-400 hover:bg-white/5 hover:text-slate-100 transition-colors cursor-pointer"
              type="button"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
