import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { listAdminTrials, getAdminOverview } from "@/features/admin/api";
import type { AdminLicenseItem } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { Badge } from "@/components/ui/badge";
import { Eye, Search, FlaskConical, CheckCircle2, Clock } from "lucide-react";

export const Route = createFileRoute("/_dash/trials/")({
  component: TrialsPage,
});

function isExpired(t: AdminLicenseItem): boolean {
  if (!t.isActive) return true;
  if (t.expiresAt) return new Date(t.expiresAt) < new Date();
  return false;
}

function daysLeft(expiresAt: string | null): number | null {
  if (!expiresAt) return null;
  const ms = new Date(expiresAt).getTime() - Date.now();
  if (Number.isNaN(ms)) return null;
  return Math.ceil(ms / 86_400_000);
}

function fmtDate(d: string | null): string {
  if (!d) return "—";
  const date = new Date(d);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleDateString();
}

function TrialsPage() {
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("all");

  const params = { q, status };

  const { data: trials, isLoading } = useQuery({
    queryKey: ["adminTrials", params],
    queryFn: () => listAdminTrials(params),
  });

  const { data: overview } = useQuery({
    queryKey: ["adminOverview"],
    queryFn: getAdminOverview,
  });

  const rows = trials ?? [];

  const stats = [
    { label: "Total Trials", value: overview?.totalTrials, icon: FlaskConical },
    { label: "Active", value: overview?.activeTrials, icon: CheckCircle2 },
    {
      label: "Expired",
      value:
        overview?.totalTrials != null && overview?.activeTrials != null
          ? overview.totalTrials - overview.activeTrials
          : undefined,
      icon: Clock,
    },
  ];

  return (
    <AdminShell title="Trials">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Trials</h1>
        <p className="text-muted-foreground text-sm">
          Trial licenses across your products. Create trials from the Licenses page.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        {stats.map((stat) => (
          <Card key={stat.label}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-muted-foreground text-sm font-medium">
                {stat.label}
              </CardTitle>
              <stat.icon className="text-muted-foreground size-4" />
            </CardHeader>
            <CardContent>
              {overview ? (
                <div className="text-2xl font-bold tabular-nums">{stat.value ?? 0}</div>
              ) : (
                <Skeleton className="h-8 w-16" />
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="text-muted-foreground absolute left-2.5 top-2.5 size-4" />
          <Input
            placeholder="Search by email or product…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            className="pl-9"
          />
        </div>
        <Select value={status} onValueChange={setStatus}>
          <SelectTrigger className="w-full sm:w-40">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="expired">Expired</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Customer</TableHead>
              <TableHead>Product</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Seats</TableHead>
              <TableHead className="text-right">Started</TableHead>
              <TableHead className="text-right">Expires</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                  <TableCell className="text-right"><Skeleton className="ml-auto h-4 w-10" /></TableCell>
                  <TableCell className="text-right"><Skeleton className="ml-auto h-4 w-20" /></TableCell>
                  <TableCell className="text-right"><Skeleton className="ml-auto h-4 w-20" /></TableCell>
                  <TableCell />
                </TableRow>
              ))
            ) : rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} className="text-muted-foreground h-24 text-center">
                  No trials found
                </TableCell>
              </TableRow>
            ) : (
              rows.map((t) => {
                const expired = isExpired(t);
                const left = daysLeft(t.expiresAt);
                return (
                  <TableRow key={t.id}>
                    <TableCell className="font-medium">{t.customerEmail}</TableCell>
                    <TableCell>{t.productName}</TableCell>
                    <TableCell>
                      <Badge variant={expired ? "secondary" : "default"}>
                        {expired ? "Expired" : "Active"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {t.activationCount}/{t.maxActivations}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-right text-sm">
                      {fmtDate(t.createdAt)}
                    </TableCell>
                    <TableCell className="text-right text-sm">
                      <span className="text-muted-foreground">{fmtDate(t.expiresAt)}</span>
                      {!expired && left != null && left >= 0 && (
                        <span className="text-muted-foreground ml-1.5 text-xs">
                          ({left}d left)
                        </span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" asChild>
                        <Link to="/licenses/$licenseId" params={{ licenseId: t.id }}>
                          <Eye className="size-4" />
                        </Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </div>
    </AdminShell>
  );
}
