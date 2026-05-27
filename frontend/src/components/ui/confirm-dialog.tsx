import { AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog } from "@/components/ui/dialog";

interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  isPending?: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export function ConfirmDialog({
  isOpen,
  title,
  description,
  confirmLabel = "确认",
  cancelLabel = "取消",
  isPending = false,
  onClose,
  onConfirm,
}: ConfirmDialogProps) {
  return (
    <Dialog isOpen={isOpen} onClose={onClose} title={title} size="sm">
      <div className="space-y-5">
        <div className="flex gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-red-500/20 bg-red-500/10 text-red-400">
            <AlertTriangle className="h-5 w-5" />
          </div>
          <p className="pt-0.5 text-slate-300 text-sm leading-6">
            {description}
          </p>
        </div>
        <div className="flex justify-end gap-2 border-white/[0.06] border-t pt-4">
          <Button
            className="h-9"
            disabled={isPending}
            onClick={onClose}
            variant="secondary"
          >
            {cancelLabel}
          </Button>
          <Button
            className="h-9"
            disabled={isPending}
            onClick={onConfirm}
            variant="danger"
          >
            {isPending ? "处理中..." : confirmLabel}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
