import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Plus, Trash2 } from "lucide-react";
import api, { type Registry } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useToast } from "@/components/ui/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
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
import { EmptyState } from "@/components/EmptyState";
import { CardPageLoading } from "@/components/PageLoading";

export function RegistriesPage() {
  const { isAdmin } = useAuth();
  const { toast } = useToast();
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [deleteId, setDeleteId] = useState<number | null>(null);
  const [form, setForm] = useState({
    name: "",
    url: "",
    username: "",
    password: "",
    insecure_skip_verify: false,
  });

  const { data, isLoading } = useQuery({
    queryKey: ["registries"],
    queryFn: async () => {
      const { data } = await api.get("/registries");
      return data.data as Registry[];
    },
  });

  const createMut = useMutation({
    mutationFn: async () => {
      await api.post("/registries", form);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["registries"] });
      setOpen(false);
      setForm({ name: "", url: "", username: "", password: "", insecure_skip_verify: false });
      toast({ title: "Registry added" });
    },
    onError: (e: Error) => toast({ title: "Failed", description: e.message, variant: "destructive" }),
  });

  const deleteMut = useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/registries/${id}`);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["registries"] });
      setDeleteId(null);
      toast({ title: "Registry deleted" });
    },
    onError: (e: Error) => toast({ title: "Failed", description: e.message, variant: "destructive" }),
  });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    createMut.mutate();
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Registries</h1>
          <p className="text-sm text-muted-foreground">Configure Docker Registry connections</p>
        </div>
        {isAdmin && (
          <Button onClick={() => setOpen(true)}>
            <Plus className="h-4 w-4" />
            Add Registry
          </Button>
        )}
      </div>

      {isLoading ? (
        <CardPageLoading />
      ) : !data?.length ? (
        <EmptyState
          title="No registries yet"
          description="Add a Docker Distribution Registry to start managing repositories and tags."
          action={
            isAdmin ? (
              <Button onClick={() => setOpen(true)}>
                <Plus className="h-4 w-4" />
                Add Registry
              </Button>
            ) : undefined
          }
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {data.map((r) => (
            <Card key={r.id}>
              <CardHeader className="flex-row items-start justify-between space-y-0">
                <div>
                  <CardTitle>
                    <Link to={`/registries/${r.id}`} className="hover:underline">
                      {r.name}
                    </Link>
                  </CardTitle>
                  <p className="mt-1 font-mono text-xs text-muted-foreground">{r.url}</p>
                </div>
                {isAdmin && (
                  <Button variant="ghost" size="icon" onClick={() => setDeleteId(r.id)}>
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                )}
              </CardHeader>
              <CardContent className="flex gap-2">
                <Button asChild variant="outline" size="sm">
                  <Link to={`/registries/${r.id}`}>Open</Link>
                </Button>
                <Button asChild variant="secondary" size="sm">
                  <Link to={`/registries/${r.id}/repositories`}>Repositories</Link>
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Registry</DialogTitle>
            <DialogDescription>Credentials are encrypted at rest and never returned by the API.</DialogDescription>
          </DialogHeader>
          <form className="space-y-3" onSubmit={onSubmit}>
            <div className="space-y-1">
              <Label>Name</Label>
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
            </div>
            <div className="space-y-1">
              <Label>URL</Label>
              <Input
                placeholder="https://registry.example.com"
                value={form.url}
                onChange={(e) => setForm({ ...form, url: e.target.value })}
                required
              />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>Username</Label>
                <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
              </div>
              <div className="space-y-1">
                <Label>Password</Label>
                <Input
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                />
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.insecure_skip_verify}
                onChange={(e) => setForm({ ...form, insecure_skip_verify: e.target.checked })}
              />
              Allow self-signed TLS certificate (this registry only)
            </label>
            <Button type="submit" disabled={createMut.isPending}>
              {createMut.isPending ? "Saving…" : "Save"}
            </Button>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteId != null} onOpenChange={(o) => !o && setDeleteId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Registry?</AlertDialogTitle>
            <AlertDialogDescription>
              This removes the connection from RegPilot. Images in the registry are not deleted.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => deleteId && deleteMut.mutate(deleteId)}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
