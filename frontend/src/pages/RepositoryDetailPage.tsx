import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api, { type Manifest, type TagInfo } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useToast } from "@/components/ui/toast";
import { formatBytes, formatRelative, shortDigest } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { TablePageLoading } from "@/components/PageLoading";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/EmptyState";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";

export function RepositoryDetailPage() {
  const { id, repository = "" } = useParams();
  const repoName = decodeURIComponent(repository);
  const { isAdmin } = useAuth();
  const { toast } = useToast();
  const qc = useQueryClient();
  const [selected, setSelected] = useState<TagInfo | null>(null);
  const [manifest, setManifest] = useState<Manifest | null>(null);
  const [deleteTag, setDeleteTag] = useState<TagInfo | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["tags", id, repoName],
    queryFn: async () => {
      const { data } = await api.get(`/registries/${id}/tags`, { params: { repository: repoName } });
      return data.data as TagInfo[];
    },
  });

  const manifestMut = useMutation({
    mutationFn: async (tag: string) => {
      const { data } = await api.get(`/registries/${id}/manifests`, {
        params: { repository: repoName, reference: tag },
      });
      return data.data as Manifest;
    },
    onSuccess: (m) => setManifest(m),
    onError: (e: Error) => toast({ title: "Failed to load manifest", description: e.message, variant: "destructive" }),
  });

  const deleteMut = useMutation({
    mutationFn: async (tag: TagInfo) => {
      await api.delete(`/registries/${id}/images`, { data: { repository: repoName, tag: tag.tag } });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["tags", id, repoName] });
      setDeleteTag(null);
      toast({ title: "Image deleted" });
    },
    onError: (e: Error) => toast({ title: "Delete failed", description: e.message, variant: "destructive" }),
  });

  return (
    <div className="space-y-6">
      <div>
        <div className="mb-1 text-sm text-muted-foreground">
          <Link to={`/registries/${id}/repositories`} className="hover:underline">
            Repositories
          </Link>{" "}
          / {repoName}
        </div>
        <h1 className="text-2xl font-semibold tracking-tight">{repoName}</h1>
        <p className="text-sm text-muted-foreground">Tags and image manifests</p>
      </div>

      {isLoading ? (
        <TablePageLoading rows={6} />
      ) : error ? (
        <EmptyState title="Failed to load tags" description={(error as Error).message} />
      ) : !data?.length ? (
        <EmptyState title="No tags" description="This repository has no tags." />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{data.length} tags</CardTitle>
          </CardHeader>
          <CardContent className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead className="border-y bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th className="px-5 py-3 font-medium">Tag</th>
                  <th className="px-5 py-3 font-medium">Digest</th>
                  <th className="px-5 py-3 font-medium">Created</th>
                  <th className="px-5 py-3 font-medium">Size</th>
                  <th className="px-5 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {data.map((t) => (
                  <tr key={t.tag} className="border-b last:border-0">
                    <td className="px-5 py-3 font-medium">{t.tag}</td>
                    <td className="px-5 py-3 font-mono text-xs">{shortDigest(t.digest)}</td>
                    <td className="px-5 py-3">{formatRelative(t.created)}</td>
                    <td className="px-5 py-3">{formatBytes(t.size)}</td>
                    <td className="px-5 py-3">
                      <div className="flex gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setSelected(t);
                            setManifest(null);
                            manifestMut.mutate(t.tag);
                          }}
                        >
                          Details
                        </Button>
                        {isAdmin && (
                          <Button size="sm" variant="destructive" onClick={() => setDeleteTag(t)}>
                            Delete
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      )}

      <Dialog open={!!selected} onOpenChange={(o) => !o && setSelected(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>
              {repoName}:{selected?.tag}
            </DialogTitle>
            <DialogDescription>Image manifest details from the registry</DialogDescription>
          </DialogHeader>
          {manifestMut.isPending && <Skeleton className="h-40" />}
          {manifest && (
            <div className="space-y-3 text-sm">
              <div>
                <div className="text-muted-foreground">Digest</div>
                <div className="break-all font-mono text-xs">{manifest.digest || "—"}</div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <div className="text-muted-foreground">Media type</div>
                  <div className="font-mono text-xs">{manifest.media_type || "—"}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">Platform</div>
                  <div>
                    {manifest.os || "—"}/{manifest.architecture || "—"}
                  </div>
                </div>
                <div>
                  <div className="text-muted-foreground">Layers</div>
                  <div>{manifest.layers?.length ?? 0}</div>
                </div>
                <div>
                  <div className="text-muted-foreground">Total size</div>
                  <div>{formatBytes(manifest.total_size)}</div>
                </div>
                <div className="col-span-2">
                  <div className="text-muted-foreground">Created</div>
                  <div>{manifest.created ? formatRelative(manifest.created) : "—"}</div>
                </div>
              </div>
              <div>
                <div className="mb-1 text-muted-foreground">Layers</div>
                <div className="max-h-48 overflow-auto rounded border">
                  <table className="w-full text-xs">
                    <thead className="bg-muted/40 text-left">
                      <tr>
                        <th className="px-2 py-1">Digest</th>
                        <th className="px-2 py-1">Size</th>
                      </tr>
                    </thead>
                    <tbody>
                      {(manifest.layers || []).map((l) => (
                        <tr key={l.digest} className="border-t">
                          <td className="px-2 py-1 font-mono">{shortDigest(l.digest)}</td>
                          <td className="px-2 py-1">{formatBytes(l.size)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <AlertDialog open={!!deleteTag} onOpenChange={(o) => !o && setDeleteTag(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Image?</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div>
                  <span className="font-medium text-foreground">Repository:</span> {repoName}
                </div>
                <div>
                  <span className="font-medium text-foreground">Tag:</span> {deleteTag?.tag}
                </div>
                <div>
                  <span className="font-medium text-foreground">Digest:</span>{" "}
                  <span className="font-mono text-xs">{deleteTag?.digest || "(will be resolved)"}</span>
                </div>
                <p>This operation cannot be undone from RegPilot.</p>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => deleteTag && deleteMut.mutate(deleteTag)}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
