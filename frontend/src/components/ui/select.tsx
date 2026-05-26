import { Check, ChevronDown } from "lucide-react";
import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";

// Context to share selected value and update function
const SelectContext = createContext<{
  value: string;
  onChange: (value: string) => void;
  isOpen: boolean;
  setIsOpen: (open: boolean) => void;
  triggerRect: DOMRect | null;
  setTriggerRect: (rect: DOMRect | null) => void;
  contentRef: React.RefObject<HTMLDivElement | null>;
} | null>(null);

interface SelectProps {
  children: ReactNode;
  value: string;
  onValueChange: (value: string) => void;
}

export function Select({ children, value, onValueChange }: SelectProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [triggerRect, setTriggerRect] = useState<DOMRect | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleOutsideClick = (e: MouseEvent) => {
      const isClickInsideTrigger = containerRef.current?.contains(
        e.target as Node,
      );
      const isClickInsideContent = contentRef.current?.contains(
        e.target as Node,
      );
      if (!isClickInsideTrigger && !isClickInsideContent) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleOutsideClick);
    return () => document.removeEventListener("mousedown", handleOutsideClick);
  }, []);

  // Update rect on scroll or resize to keep it positioned correctly
  useEffect(() => {
    if (!isOpen || !containerRef.current) return;

    const updateRect = () => {
      const button = containerRef.current?.querySelector("button");
      if (button) {
        setTriggerRect(button.getBoundingClientRect());
      }
    };

    window.addEventListener("scroll", updateRect, true);
    window.addEventListener("resize", updateRect);

    return () => {
      window.removeEventListener("scroll", updateRect, true);
      window.removeEventListener("resize", updateRect);
    };
  }, [isOpen]);

  return (
    <SelectContext.Provider
      value={{
        value,
        onChange: onValueChange,
        isOpen,
        setIsOpen,
        triggerRect,
        setTriggerRect,
        contentRef,
      }}
    >
      <div ref={containerRef} className="relative w-full">
        {children}
      </div>
    </SelectContext.Provider>
  );
}

interface SelectTriggerProps {
  className?: string;
  children: ReactNode;
  disabled?: boolean;
}

export function SelectTrigger({
  className,
  children,
  disabled,
}: SelectTriggerProps) {
  const context = useContext(SelectContext);
  if (!context) throw new Error("SelectTrigger must be used within a Select");
  const buttonRef = useRef<HTMLButtonElement>(null);

  const handleClick = () => {
    if (disabled) return;
    if (buttonRef.current) {
      context.setTriggerRect(buttonRef.current.getBoundingClientRect());
    }
    context.setIsOpen(!context.isOpen);
  };

  return (
    <button
      ref={buttonRef}
      type="button"
      disabled={disabled}
      onClick={handleClick}
      className={cn(
        "flex h-9 w-full items-center justify-between rounded-lg border border-white/[0.06] bg-[#090b11] px-3.5 text-xs text-slate-100 placeholder-slate-600 outline-none transition-all duration-200 focus:border-[#4f46e5] focus:bg-[#0e111a] focus:shadow-[0_0_12px_rgba(99,102,241,0.06)] disabled:opacity-40 disabled:pointer-events-none text-left cursor-pointer",
        className,
      )}
    >
      {children}
      <ChevronDown className="h-3.5 w-3.5 text-slate-500 shrink-0 ml-2" />
    </button>
  );
}

interface SelectValueProps {
  placeholder?: string;
  displayValue?: string;
}

export function SelectValue({ placeholder, displayValue }: SelectValueProps) {
  const context = useContext(SelectContext);
  if (!context) throw new Error("SelectValue must be used within a Select");

  return (
    <span className={cn(!context.value && "text-slate-500 truncate")}>
      {displayValue || context.value || placeholder}
    </span>
  );
}

interface SelectContentProps {
  className?: string;
  children: ReactNode;
}

export function SelectContent({ className, children }: SelectContentProps) {
  const context = useContext(SelectContext);
  if (!context) throw new Error("SelectContent must be used within a Select");

  if (!context.isOpen || !context.triggerRect) return null;

  const { left, top, width, height } = context.triggerRect;

  return createPortal(
    <div
      ref={context.contentRef}
      className={cn(
        "fixed z-50 mt-1.5 max-h-60 overflow-y-auto rounded-lg border border-white/[0.06] bg-[#0e1017] p-1 shadow-2xl animate-in fade-in slide-in-from-top-1 duration-100 scrollbar-thin",
        className,
      )}
      style={{
        top: top + height,
        left: left,
        width: width,
      }}
    >
      {children}
    </div>,
    document.body,
  );
}

interface SelectItemProps {
  className?: string;
  value: string;
  children: ReactNode;
}

export function SelectItem({ className, value, children }: SelectItemProps) {
  const context = useContext(SelectContext);
  if (!context) throw new Error("SelectItem must be used within a Select");

  const isSelected = context.value === value;

  return (
    <button
      type="button"
      onClick={() => {
        context.onChange(value);
        context.setIsOpen(false);
      }}
      className={cn(
        "relative flex w-full cursor-pointer select-none items-center justify-between rounded-md py-1.5 pl-3.5 pr-8 text-xs text-slate-300 outline-none hover:bg-white/5 hover:text-white transition-colors text-left",
        isSelected && "text-white font-semibold bg-white/[0.02]",
        className,
      )}
    >
      <span className="truncate">{children}</span>
      {isSelected && (
        <Check className="absolute right-3.5 h-3.5 w-3.5 text-slate-300" />
      )}
    </button>
  );
}
