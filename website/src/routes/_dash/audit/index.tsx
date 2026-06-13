import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import { listAuditLogs, type AuditLogFilters } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Search, X } from "lucide-react";

const ACTION_OPTIONS = [
  "admin.login",
  "admin.login_failed",
  "admin.logout",
  "admin.2fa_setup_started",
  "admin.2fa_enabled",
  "admin.2fa_verified",
  "license.created",
  "license.updated",
  "license.deleted",
  "product.created",
  "product.updated",
  "product.deleted",
  "organization.created",
  "organization.switched",
  "organization.invite_created",
  "organization.invite_deleted",
  "organization.member_removed",
  "update_config.saved",
  "update_config.deleted",
  "storage_config.saved",
  "storage_config.tested",
  "channel.created",
  "channel.updated",
  "channel.deleted",
  "release.created",
  "release.artifact_uploaded",
  "release.published",
  "release.yanked",
  "release.deleted",
  "changelog.created",
  "changelog.updated",
  "changelog.deleted",
  "release.changelog_attached",
  "release.changelog_detached",
];

const RESOURCE_TYPES = [
  "admin",
  "license",
  "product",
  "organization",
  "update_config",
  "storage_config",
  "channel",
  "release",
  "changelog",
];

function actionVariant(action: string): "default" | "secondary" | "destructive" {
  if (action.endsWith(".deleted")) return "destructive";
  if (action.endsWith(".created")) return "default";
  if (action.endsWith("failed")) return "destructive";
  return "secondary";
}

function actionLabel(action: string): string {
  return action.replace(/_/g, " ").replace(/\./g, " › ");
}

export const Route = createFileRoute("/_dash/audit/")({
  component: AuditPage,
});

function AuditPage() {
  const [page, setPage] = useState(1);
  const pageSize = 50;
  const [search, setSearch] = useState("");
  const [actionFilter, setActionFilter] = useState("all");
  const [resourceTypeFilter, setResourceTypeFilter] = useState("all");
  const [actorFilter, setActorFilter] = useState("");
  const [appliedFilters, setAppliedFilters] = useState<AuditLogFilters>({});

  const hasFilters =
    appliedFilters.q ||
    appliedFilters.action ||
    appliedFilters.resourceType ||
    appliedFilters.adminEmail;

  const { data, isLoading } = useQuery({
    queryKey: ["auditLogs", page, appliedFilters],
    queryFn: () => listAuditLogs(page, pageSize, hasFilters ? appliedFilters : undefined),
    refetchOnMount: "always",
    staleTime: 0,
  });

  const applyFilters = useCallback(() => {
    const f: AuditLogFilters = {};
    if (search.trim()) f.q = search.trim();
    if (actionFilter !== "all") f.action = actionFilter;
    if (resourceTypeFilter !== "all") f.resourceType = resourceTypeFilter;
    if (actorFilter.trim()) f.adminEmail = actorFilter.trim();
    setAppliedFilters(f);
    setPage(1);
  }, [search, actionFilter, resourceTypeFilter, actorFilter]);

  const resetFilters = useCallback(() => {
    setSearch("");
    setActionFilter("all");
    setResourceTypeFilter("all");
    setActorFilter("");
    setAppliedFilters({});
    setPage(1);
  }, []);

  const totalPages = data ? Math.ceil(data.total / data.pageSize) : 0;

  return (
    <AdminShell title="Audit Log">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Audit Log</h1>
        <p className="text-muted-foreground text-sm">
          Administrative actions across this organization.
        </p>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="flex-1 min-w-[200px] max-w-sm">
          <Label htmlFor="audit-search" className="sr-only">Search</Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
            <Input
              id="audit-search"
              placeholder="Search actions, resources, actors…"
              className="pl-8"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && applyFilters()}
            />
          </div>
        </div>
        <div className="w-44">
          <Label htmlFor="audit-action" className="sr-only">Action</Label>
          <Select value={actionFilter} onValueChange={setActionFilter}>
            <SelectTrigger id="audit-action"><SelectValue placeholder="All actions" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All actions</SelectItem>
              {ACTION_OPTIONS.map((a) => (
                <SelectItem key={a} value={a}>{actionLabel(a)}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="w-40">
          <Label htmlFor="audit-resource" className="sr-only">Resource</Label>
          <Select value={resourceTypeFilter} onValueChange={setResourceTypeFilter}>
            <SelectTrigger id="audit-resource"><SelectValue placeholder="All resources" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All resources</SelectItem>
              {RESOURCE_TYPES.map((r) => (
                <SelectItem key={r} value={r} className="capitalize">{r.replace(/_/g, " ")}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="w-44">
          <Label htmlFor="audit-actor" className="sr-only">Actor</Label>
          <Input
            id="audit-actor"
            placeholder="Actor email…"
            value={actorFilter}
            onChange={(e) => setActorFilter(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && applyFilters()}
          />
        </div>
        <Button variant="secondary" onClick={applyFilters}>Filter</Button>
        {hasFilters && (
          <Button variant="ghost" size="icon" onClick={resetFilters} title="Clear filters">
            <X className="size-4" />
          </Button>
        )}
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Time</TableHead>
              <TableHead>Actor</TableHead>
              <TableHead>Action</TableHead>
              <TableHead>Resource</TableHead>
              <TableHead>ID</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 8 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-36" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-24 rounded-full" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-48" /></TableCell>
                </TableRow>
              ))
            ) : !data?.items.length ? (
              <TableRow>
                <TableCell colSpan={5} className="text-muted-foreground h-24 text-center">
                  {hasFilters ? "No matching audit entries" : "No activity recorded yet"}
                </TableCell>
              </TableRow>
            ) : (
              data.items.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                    {log.createdAt ? new Date(log.createdAt).toLocaleString() : "—"}
                  </TableCell>
                  <TableCell className="text-sm">{log.adminEmail ?? "—"}</TableCell>
                  <TableCell>
                    <Badge variant={actionVariant(log.action)}>{actionLabel(log.action)}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm capitalize">
                    {log.resourceType}
                  </TableCell>
                  <TableCell className="text-muted-foreground font-mono text-xs">
                    {log.resourceId ?? "—"}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-muted-foreground text-sm">
            Page {page} of {totalPages} ({data?.total} total)
          </div>
          <div className="flex gap-2">
            <Button variant="outline" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              Previous
            </Button>
            <Button
              variant="outline"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </AdminShell>
  );
}
