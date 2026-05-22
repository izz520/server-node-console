import type { LucideIcon } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";

interface ResourcePlaceholderProps {
  title: string;
  description: string;
  icon: LucideIcon;
  actions: string[];
}

export function ResourcePlaceholder({
  title,
  description,
  icon: Icon,
  actions,
}: ResourcePlaceholderProps) {
  return (
    <div className="space-y-6">
      <section className="flex items-center gap-4">
        <div className="flex h-11 w-11 items-center justify-center rounded-md bg-slate-950 text-white">
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <h1 className="font-semibold text-2xl text-slate-950">{title}</h1>
          <p className="mt-1 text-slate-600 text-sm">{description}</p>
        </div>
      </section>

      <Card>
        <CardHeader>
          <div className="font-medium text-slate-950">待实现能力</div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 md:grid-cols-2">
            {actions.map((action) => (
              <div
                className="rounded-md border border-slate-100 p-3 text-slate-700 text-sm"
                key={action}
              >
                {action}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
