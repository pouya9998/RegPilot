import { Badge } from "@/components/ui/badge";

const map: Record<string, { label: string; variant: "success" | "warning" | "destructive" | "muted" }> = {
  healthy: { label: "Healthy", variant: "success" },
  unhealthy: { label: "Unhealthy", variant: "destructive" },
  unauthorized: { label: "Unauthorized", variant: "warning" },
  unreachable: { label: "Unreachable", variant: "destructive" },
};

export function StatusBadge({ status }: { status?: string }) {
  const conf = map[status || ""] || { label: status || "Unknown", variant: "muted" as const };
  return <Badge variant={conf.variant}>{conf.label}</Badge>;
}
