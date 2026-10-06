import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import api, { type User } from "@/lib/api";
import { useToast } from "@/components/ui/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";

export function UsersPage() {
  const { toast } = useToast();
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [resetId, setResetId] = useState<number | null>(null);
  const [password, setPassword] = useState("");
  const [form, setForm] = useState({ username: "", password: "", role: "VIEWER" });

  const { data, isLoading } = useQuery({
    queryKey: ["users"],
    queryFn: async () => {
      const { data } = await api.get("/users");
      return data.data as User[];
    },
  });

  const createMut = useMutation({
    mutationFn: async () => api.post("/users", form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      setOpen(false);
      setForm({ username: "", password: "", role: "VIEWER" });
      toast({ title: "User created" });
    },
    onError: (e: Error) => toast({ title: "Failed", description: e.message, variant: "destructive" }),
  });

  const disableMut = useMutation({
    mutationFn: async ({ id, disabled }: { id: number; disabled: boolean }) =>
      api.patch(`/users/${id}`, { disabled }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast({ title: "User updated" });
    },
    onError: (e: Error) => toast({ title: "Failed", description: e.message, variant: "destructive" }),
  });

  const deleteMut = useMutation({
    mutationFn: async (id: number) => api.delete(`/users/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["users"] });
      toast({ title: "User deleted" });
    },
    onError: (e: Error) => toast({ title: "Failed", description: e.message, variant: "destructive" }),
  });

  const resetMut = useMutation({
    mutationFn: async () => api.post(`/users/${resetId}/reset-password`, { password }),
    onSuccess: () => {
      setResetId(null);
      setPassword("");
      toast({ title: "Password reset" });
    },
    onError: (e: Error) => toast({ title: "Failed", description: e.message, variant: "destructive" }),
  });

  function onCreate(e: FormEvent) {
    e.preventDefault();
    createMut.mutate();
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Users</h1>
          <p className="text-sm text-muted-foreground">Manage ADMIN and VIEWER accounts</p>
        </div>
        <Button onClick={() => setOpen(true)}>Create User</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">All users</CardTitle>
        </CardHeader>
        <CardContent className="overflow-x-auto p-0">
          {isLoading ? (
            <Skeleton className="m-5 h-32" />
          ) : (
            <table className="w-full text-sm">
              <thead className="border-y bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th className="px-5 py-3 font-medium">Username</th>
                  <th className="px-5 py-3 font-medium">Role</th>
                  <th className="px-5 py-3 font-medium">Status</th>
                  <th className="px-5 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {data?.map((u) => (
                  <tr key={u.id} className="border-b last:border-0">
                    <td className="px-5 py-3 font-medium">{u.username}</td>
                    <td className="px-5 py-3">
                      <Badge variant="secondary">{u.role}</Badge>
                    </td>
                    <td className="px-5 py-3">
                      <Badge variant={u.disabled ? "destructive" : "success"}>
                        {u.disabled ? "Disabled" : "Active"}
                      </Badge>
                    </td>
                    <td className="px-5 py-3">
                      <div className="flex flex-wrap gap-2">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => disableMut.mutate({ id: u.id, disabled: !u.disabled })}
                        >
                          {u.disabled ? "Enable" : "Disable"}
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => setResetId(u.id)}>
                          Reset Password
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => {
                            if (confirm(`Delete user ${u.username}?`)) deleteMut.mutate(u.id);
                          }}
                        >
                          Delete
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create User</DialogTitle>
          </DialogHeader>
          <form className="space-y-3" onSubmit={onCreate}>
            <div className="space-y-1">
              <Label>Username</Label>
              <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} required />
            </div>
            <div className="space-y-1">
              <Label>Password</Label>
              <Input
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                required
                minLength={8}
              />
            </div>
            <div className="space-y-1">
              <Label>Role</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-card px-3 text-sm"
                value={form.role}
                onChange={(e) => setForm({ ...form, role: e.target.value })}
              >
                <option value="VIEWER">VIEWER</option>
                <option value="ADMIN">ADMIN</option>
              </select>
            </div>
            <Button type="submit" disabled={createMut.isPending}>
              Create
            </Button>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={resetId != null} onOpenChange={(o) => !o && setResetId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reset Password</DialogTitle>
          </DialogHeader>
          <form
            className="space-y-3"
            onSubmit={(e) => {
              e.preventDefault();
              resetMut.mutate();
            }}
          >
            <div className="space-y-1">
              <Label>New password</Label>
              <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} minLength={8} required />
            </div>
            <Button type="submit">Reset</Button>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
