import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import api, { type AuditLog } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";

export function AuditLogsPage() {
  const [user, setUser] = useState("");
  const [action, setAction] = useState("");
  const [registryId, setRegistryId] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["audit", user, action, registryId],
    queryFn: async () => {
      const { data } = await api.get("/audit-logs", {
        params: {
          user: user || undefined,
          action: action || undefined,
          registry_id: registryId || undefined,
          limit: 100,
        },
      });
      return data.data as { items: AuditLog[]; total: number };
    },
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Audit Logs</h1>
        <p className="text-sm text-muted-foreground">Track authentication and destructive actions</p>
      </div>

      <div className="grid gap-3 md:grid-cols-3">
        <Input placeholder="Filter by user" value={user} onChange={(e) => setUser(e.target.value)} />
        <Input placeholder="Filter by action (e.g. DELETE_IMAGE)" value={action} onChange={(e) => setAction(e.target.value)} />
        <Input placeholder="Registry ID" value={registryId} onChange={(e) => setRegistryId(e.target.value)} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{data?.total ?? 0} events</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto p-0">
          {isLoading ? (
            <Skeleton className="m-5 h-40" />
          ) : (
            <table className="w-full text-sm">
              <thead className="border-y bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th className="px-5 py-3 font-medium">Time</th>
                  <th className="px-5 py-3 font-medium">User</th>
                  <th className="px-5 py-3 font-medium">Action</th>
                  <th className="px-5 py-3 font-medium">Target</th>
                  <th className="px-5 py-3 font-medium">Result</th>
                </tr>
              </thead>
              <tbody>
                {data?.items?.map((a) => (
                  <tr key={a.id} className="border-b last:border-0">
                    <td className="px-5 py-3 text-xs text-muted-foreground">
                      {new Date(a.created_at).toLocaleString()}
                    </td>
                    <td className="px-5 py-3">{a.username || "—"}</td>
                    <td className="px-5 py-3 font-medium">{a.action}</td>
                    <td className="px-5 py-3 font-mono text-xs">
                      {[a.repository, a.tag].filter(Boolean).join(":") || a.digest || "—"}
                    </td>
                    <td className="px-5 py-3">
                      <Badge variant={a.result === "success" ? "success" : a.result === "failure" ? "destructive" : "muted"}>
                        {a.result || "—"}
                      </Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
