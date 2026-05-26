import { X } from "lucide-react";
import { type ReactNode, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";

interface DialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  className?: string;
  size?: "sm" | "md" | "lg" | "xl" | "2xl" | "wide";
}

const sizeClasses = {
  sm: "max-w-md",
  md: "max-w-lg",
  lg: "max-w-2xl",
  xl: "max-w-4xl",
  "2xl": "max-w-6xl",
  wide: "max-w-7xl w-[92vw]",
};

export function Dialog({
  isOpen,
  onClose,
  title,
  children,
  className,
  size = "md",
}: DialogProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  // Esc key to close
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, onClose]);

  // Lock body scroll
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop overlay with blur */}
      {/* biome-ignore lint/a11y/useKeyWithClickEvents: Backdrop key clicks are managed by the document escape key handler */}
      {/* biome-ignore lint/a11y/noStaticElementInteractions: Backdrop div serves only to catch overlay mouse clicks */}
      <div
        ref={overlayRef}
        onClick={(e) => {
          if (e.target === overlayRef.current) onClose();
        }}
        className="fixed inset-0 bg-black/75 backdrop-blur-[6px] transition-all duration-300 animate-in fade-in"
      />

      {/* Dialog container */}
      <div
        className={cn(
          "relative z-10 w-full overflow-hidden rounded-xl border border-white/[0.06] bg-[#0e1017] shadow-2xl transition-all duration-300 animate-in fade-in zoom-in-95 duration-200 flex flex-col max-h-[85vh]",
          sizeClasses[size],
          className,
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-white/[0.06] px-5 py-4 shrink-0">
          <h2 className="font-display font-bold text-slate-100 text-lg tracking-tight">
            {title}
          </h2>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-slate-400 hover:bg-white/5 hover:text-slate-100 transition-colors cursor-pointer"
            type="button"
            aria-label="关闭"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content body */}
        <div className="overflow-y-auto p-6 scrollbar-thin">{children}</div>
      </div>
    </div>,
    document.body,
  );
}
