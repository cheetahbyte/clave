import { createFileRoute, redirect } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { getCurrentAdmin, listAuditLogs } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export const Route = createFileRoute("/audit/")({
  beforeLoad: async () => {
    try {
      const admin = await getCurrentAdmin();
      if (!admin.mfaVerified) {
        if (admin.mfaEnabled) throw redirect({ to: "/2fa" });
        throw redirect({ to: "/2fa/setup" });
      }
    } catch (err) {
      if (err instanceof Error && "redirect" in (err as unknown as Record<string, unknown>)) throw err;
      throw redirect({ to: "/login" });
    }
  },
  component: AuditPage,
});

function actionVariant(action: string): "default" | "secondary" | "destructive" {
  if (action.endsWith(".deleted")) return "destructive";
  if (action.endsWith(".created")) return "default";
  return "secondary";
}

function AuditPage() {
  const [page, setPage] = useState(1);
  const pageSize = 50;

  const { data, isLoading } = useQuery({
    queryKey: ["auditLogs", page],
    queryFn: () => listAuditLogs(page, pageSize),
    refetchOnMount: "always",
    staleTime: 0,
  });

  const totalPages = data ? Math.ceil(data.total / data.pageSize) : 0;

  return (
    <AdminShell title="Audit Log">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Audit Log</h1>
        <p className="text-muted-foreground text-sm">
          Administrative actions across this organization.
        </p>
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
                  No activity recorded yet
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
                    <Badge variant={actionVariant(log.action)}>{log.action}</Badge>
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
