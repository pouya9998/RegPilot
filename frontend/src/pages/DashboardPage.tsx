import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import api, { type Dashboard } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/StatusBadge";
import { EmptyState } from "@/components/EmptyState";
import { StorageInfoCard } from "@/components/StorageInfoCard";
import { DashboardLoading, InlineLoading } from "@/components/PageLoading";
import { Badge } from "@/components/ui/badge";

export function DashboardPage() {
  const { data, isLoading, isFetching, error } = useQuery({
    queryKey: ["dashboard"],
    queryFn: async () => {
      const { data } = await api.get("/dashboard");
      return data.data as Dashboard;
    },
    refetchInterval: 60000,
  });

  if (isLoading) {
    return <DashboardLoading />;
  }

  if (error || !data) {
    return <EmptyState title="Failed to load dashboard" description={(error as Error)?.message} />;
  }

  const storage = data.storage ?? {
    available: data.storage_available,
    message: data.storage_message || "Storage information unavailable",
    detail:
      "Docker Registry HTTP API V2 does not expose filesystem or disk usage. RegPilot does not report disk metrics without a reliable registry API.",
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
          <p className="text-sm text-muted-foreground">Overview of connected registries</p>
        </div>
        {isFetching && <InlineLoading label="Refreshing…" />}
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard label="Repositories" value={data.repository_count} />
        <StatCard
          label="Tags / Images"
          value={data.tag_count}
          hint={data.tag_count_partial ? "Count may be partial (large registries)" : undefined}
        />
        <StatCard label="Registries" value={data.registry_count} />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Registry Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {data.registries.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No registries configured.{" "}
                <Link className="text-primary underline" to="/registries">
                  Add one
                </Link>
              </p>
            ) : (
              data.registries.map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-md border px-3 py-2">
                  <div>
                    <Link to={`/registries/${r.id}`} className="font-medium hover:underline">
                      {r.name}
                    </Link>
                    <div className="text-xs text-muted-foreground">
                      {r.repositories} repos · {r.tags} tags
                      {r.latency_ms ? ` · ${r.latency_ms}ms` : ""}
                    </div>
                  </div>
                  <StatusBadge status={r.status} />
                </div>
              ))
            )}
          </CardContent>
        </Card>

        <StorageInfoCard info={storage} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Recent Activity</CardTitle>
        </CardHeader>
        <CardContent>
          {data.recent_activity?.length ? (
            <div className="divide-y">
              {data.recent_activity.map((a) => (
                <div key={a.id} className="flex flex-wrap items-center justify-between gap-2 py-2 text-sm">
                  <div>
                    <span className="font-medium">{a.action}</span>
                    {a.repository && (
                      <span className="text-muted-foreground">
                        {" "}
                        · {a.repository}
                        {a.tag ? `:${a.tag}` : ""}
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {a.username} · {new Date(a.created_at).toLocaleString()}
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">No recent activity yet</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function StatCard({ label, value, hint }: { label: string; value: number; hint?: string }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-semibold tabular-nums">{value}</div>
        {hint && (
          <Badge variant="warning" className="mt-2">
            {hint}
          </Badge>
        )}
      </CardContent>
    </Card>
  );
}
