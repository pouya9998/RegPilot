import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import api, { type RepoSummary } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { TablePageLoading } from "@/components/PageLoading";
import { EmptyState } from "@/components/EmptyState";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

type SortKey = "name" | "tag_count";

export function RepositoriesPage() {
  const { id } = useParams();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<SortKey>("name");
  const [asc, setAsc] = useState(true);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const { data, isLoading, isFetching, error } = useQuery({
    queryKey: ["repos", id, search],
    queryFn: async () => {
      const { data } = await api.get(`/registries/${id}/repositories`, { params: { search } });
      return data.data as RepoSummary[];
    },
    placeholderData: (prev) => prev,
  });

  const sorted = useMemo(() => {
    const list = [...(data || [])];
    list.sort((a, b) => {
      const av = sort === "name" ? a.name : a.tag_count;
      const bv = sort === "name" ? b.name : b.tag_count;
      if (av < bv) return asc ? -1 : 1;
      if (av > bv) return asc ? 1 : -1;
      return 0;
    });
    return list;
  }, [data, sort, asc]);

  const totalPages = Math.max(1, Math.ceil(sorted.length / pageSize));
  const pageItems = sorted.slice((page - 1) * pageSize, page * pageSize);

  function toggleSort(key: SortKey) {
    if (sort === key) setAsc(!asc);
    else {
      setSort(key);
      setAsc(true);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="mb-1 text-sm text-muted-foreground">
            <Link to={`/registries/${id}`} className="hover:underline">
              Registry
            </Link>{" "}
            / Repositories
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Repositories</h1>
          {isFetching && !isLoading && (
            <p className="text-xs text-muted-foreground">Updating list…</p>
          )}
        </div>
        <Input
          className="max-w-xs"
          placeholder="Search repositories…"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setPage(1);
          }}
        />
      </div>

      {isLoading ? (
        <TablePageLoading />
      ) : error ? (
        <EmptyState title="Failed to load repositories" description={(error as Error).message} />
      ) : !sorted.length ? (
        <EmptyState title="No repositories" description="This registry has no repositories matching your search." />
      ) : (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{sorted.length} repositories</CardTitle>
          </CardHeader>
          <CardContent className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead className="border-y bg-muted/40 text-left text-muted-foreground">
                <tr>
                  <th className="cursor-pointer px-5 py-3 font-medium" onClick={() => toggleSort("name")}>
                    Repository {sort === "name" ? (asc ? "↑" : "↓") : ""}
                  </th>
                  <th className="cursor-pointer px-5 py-3 font-medium" onClick={() => toggleSort("tag_count")}>
                    Tags {sort === "tag_count" ? (asc ? "↑" : "↓") : ""}
                  </th>
                  <th className="px-5 py-3 font-medium">Latest Tag</th>
                  <th className="px-5 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {pageItems.map((r) => (
                  <tr key={r.name} className="border-b last:border-0">
                    <td className="px-5 py-3 font-medium">
                      <Link
                        className="hover:underline"
                        to={`/registries/${id}/repositories/${encodeURIComponent(r.name)}`}
                      >
                        {r.name}
                      </Link>
                    </td>
                    <td className="px-5 py-3 tabular-nums">{r.tag_count}</td>
                    <td className="px-5 py-3 font-mono text-xs">{r.latest_tag || "—"}</td>
                    <td className="px-5 py-3">
                      <Button asChild size="sm" variant="outline">
                        <Link to={`/registries/${id}/repositories/${encodeURIComponent(r.name)}`}>View</Link>
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="flex items-center justify-between px-5 py-3 text-sm text-muted-foreground">
              <span>
                Page {page} of {totalPages}
              </span>
              <div className="flex gap-2">
                <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                  Previous
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
