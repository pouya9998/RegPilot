import { HardDrive } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { StorageInfo } from "@/lib/api";

const defaultInfo: StorageInfo = {
  available: false,
  message: "Storage information unavailable",
  detail:
    "Docker Registry HTTP API V2 does not expose filesystem or disk usage. RegPilot does not estimate disk usage to avoid misleading metrics.",
};

export function StorageInfoCard({ info }: { info?: StorageInfo | null }) {
  const storage = info ?? defaultInfo;

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base">Storage</CardTitle>
        <Badge variant={storage.available ? "success" : "muted"}>
          {storage.available ? "Available" : "Unavailable"}
        </Badge>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex gap-3 rounded-md border border-dashed bg-muted/30 p-3">
          <HardDrive className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" aria-hidden />
          <div className="space-y-1 text-sm">
            <p className="font-medium text-foreground">{storage.message || defaultInfo.message}</p>
            <p className="text-muted-foreground">{storage.detail || defaultInfo.detail}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
