import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api, { type GCResult, type Health, type Registry } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useToast } from "@/components/ui/toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/StatusBadge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StorageInfoCard } from "@/components/StorageInfoCard";
import { CardPageLoading, InlineLoading } from "@/components/PageLoading";
import { EmptyState } from "@/components/EmptyState";
import type { StorageInfo } from "@/lib/api";

export function RegistryDetailPage() {
  const { id } = useParams();
  const { isAdmin } = useAuth();
  const { toast } = useToast();
  const qc = useQueryClient();
  const [gcResult, setGcResult] = useState<GCResult | null>(null);
  const [cleanup, setCleanup] = useState({ repository: "", keep_latest: 10, delete_older_days: 30 });
  const [preview, setPreview] = useState<{ count: number; estimated_reclaim_human: string; tags_would_delete: string[] } | null>(null);

  const {
    data: registry,
    isLoading: registryLoading,
    error: registryError,
  } = useQuery({
    queryKey: ["registry", id],
    queryFn: async () => {
      const { data } = await api.get(`/registries/${id}`);
      return data.data as Registry;
    },
  });

  const { data: storage } = useQuery({
    queryKey: ["registry-storage", id],
    queryFn: async () => {
      const { data } = await api.get(`/registries/${id}/storage`);
      return data.data as StorageInfo;
    },
  });

  const { data: health, isLoading: healthLoading } = useQuery({
    queryKey: ["registry-health", id],
    queryFn: async () => {
      const { data } = await api.get(`/registries/${id}/health`);
      return data.data as Health;
    },
    refetchInterval: 15000,
  });

  const testMut = useMutation({
    mutationFn: async () => {
      await api.post(`/registries/${id}/test`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["registry-health", id] });
      toast({ title: "Connection successful" });
    },
    onError: (e: Error) => toast({ title: "Connection failed", description: e.message, variant: "destructive" }),
  });

  const gcMut = useMutation({
    mutationFn: async (dry_run: boolean) => {
      const { data } = await api.post(`/registries/${id}/gc`, { dry_run });
      return data.data as GCResult;
    },
    onSuccess: (res) => {
      setGcResult(res);
      toast({ title: res.dry_run ? "Dry run complete" : "GC finished", description: res.message });
    },
    onError: (e: Error) => toast({ title: "GC failed", description: e.message, variant: "destructive" }),
  });

  const previewMut = useMutation({
    mutationFn: async () => {
      const { data } = await api.post(`/registries/${id}/cleanup/preview`, cleanup);
      return data.data;
    },
    onSuccess: (res) => {
      setPreview(res);
      toast({ title: "Cleanup preview ready" });
    },
    onError: (e: Error) => toast({ title: "Preview failed", description: e.message, variant: "destructive" }),
  });

  if (registryLoading) {
    return <CardPageLoading />;
  }

  if (registryError || !registry) {
    return (
      <EmptyState
        title="Failed to load registry"
        description={(registryError as Error)?.message ?? "Registry not found"}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{registry.name}</h1>
          <p className="font-mono text-sm text-muted-foreground">{registry.url}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {healthLoading && !health ? (
            <InlineLoading label="Checking health…" />
          ) : (
            health && <StatusBadge status={health.status} />
          )}
          {isAdmin && (
            <Button variant="outline" onClick={() => testMut.mutate()} disabled={testMut.isPending}>
              {testMut.isPending ? "Testing…" : "Test Connection"}
            </Button>
          )}
          <Button asChild>
            <Link to={`/registries/${id}/repositories`}>Repositories</Link>
          </Button>
        </div>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Details</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <Row label="Username" value={registry.username || "—"} />
            <Row label="Password" value={registry.has_password ? "Configured (hidden)" : "None"} />
            <Row label="Self-signed TLS" value={registry.insecure_skip_verify ? "Allowed" : "Strict"} />
            <Row label="Latency" value={healthLoading && !health ? "Checking…" : health ? `${health.latency_ms} ms` : "—"} />
          </CardContent>
        </Card>

        <StorageInfoCard info={storage} />
      </div>

      {isAdmin && (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Garbage Collection</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Garbage collection removes unreferenced blobs from the registry storage. Make sure the registry is
                configured appropriately and that no push/delete operations are running during the operation.
              </p>
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" onClick={() => gcMut.mutate(true)} disabled={gcMut.isPending}>
                  Dry Run
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => {
                    if (confirm("Run garbage collection? This may permanently remove unreferenced blobs.")) {
                      gcMut.mutate(false);
                    }
                  }}
                  disabled={gcMut.isPending}
                >
                  Run GC
                </Button>
              </div>
              {gcResult && (
                <div className="rounded-md border bg-muted/40 p-3 text-sm">
                  <div className="font-medium">{gcResult.dry_run ? "Dry Run Result" : "GC Result"}</div>
                  <p className="mt-1 text-muted-foreground">{gcResult.message}</p>
                  {gcResult.supported && (
                    <ul className="mt-2 space-y-1">
                      <li>Reclaimable blobs: {gcResult.reclaimable_blobs ?? "—"}</li>
                      <li>Estimated reclaimable space: {gcResult.reclaimable_human || "unknown"}</li>
                    </ul>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Retention Policy Preview</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Preview which tags would be deleted. Nothing is deleted automatically in this MVP.
              </p>
              <div className="grid gap-3 md:grid-cols-3">
                <div className="space-y-1 md:col-span-1">
                  <Label>Repository</Label>
                  <Input
                    value={cleanup.repository}
                    onChange={(e) => setCleanup({ ...cleanup, repository: e.target.value })}
                    placeholder="prime-core-api"
                  />
                </div>
                <div className="space-y-1">
                  <Label>Keep latest</Label>
                  <Input
                    type="number"
                    value={cleanup.keep_latest}
                    onChange={(e) => setCleanup({ ...cleanup, keep_latest: Number(e.target.value) })}
                  />
                </div>
                <div className="space-y-1">
                  <Label>Delete older than (days)</Label>
                  <Input
                    type="number"
                    value={cleanup.delete_older_days}
                    onChange={(e) => setCleanup({ ...cleanup, delete_older_days: Number(e.target.value) })}
                  />
                </div>
              </div>
              <Button onClick={() => previewMut.mutate()} disabled={previewMut.isPending || !cleanup.repository}>
                Preview Cleanup
              </Button>
              {preview && (
                <div className="rounded-md border bg-muted/40 p-3 text-sm">
                  <div>
                    {preview.count} tags would be deleted
                  </div>
                  <div className="text-muted-foreground">
                    Estimated reclaimable size: {preview.estimated_reclaim_human}
                  </div>
                  {preview.tags_would_delete?.length > 0 && (
                    <div className="mt-2 font-mono text-xs">{preview.tags_would_delete.slice(0, 20).join(", ")}</div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right">{value}</span>
    </div>
  );
}
