import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { getCurrentAdmin } from "@/features/admin/api";
import { getAdminOverview } from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
import { AdminShell } from "@/components/admin/AdminShell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Key,
  Users,
  Package,
  AlertCircle,
  CircleCheck,
  RefreshCw,
  Ban,
  Download,
  FlaskConical,
  type LucideIcon,
} from "lucide-react";

const UsageChart = React.lazy(() => import("@/features/admin/UsageChart"));

type Tone = "success" | "info" | "danger" | "muted";

const toneStyles: Record<Tone, string> = {
  success: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
  info: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  danger: "bg-destructive/10 text-destructive",
  muted: "bg-muted text-muted-foreground",
};

type ActivityEvent = {
  id: number;
  icon: LucideIcon;
  tone: Tone;
  title: string;
  meta: string[];
  time: string;
};

// Mock feed — wire to a real events endpoint when available.
const activity: ActivityEvent[] = [
  {
    id: 1,
    icon: CircleCheck,
    tone: "success",
    title: "License activated",
    meta: ["Kepler", "laura@mail.de", "MacBook Air"],
    time: "2 min ago",
  },
  {
    id: 2,
    icon: RefreshCw,
    tone: "info",
    title: "Update checked",
    meta: ["Kepler 0.5.1", "latest 0.5.2"],
    time: "5 min ago",
  },
  {
    id: 3,
    icon: Ban,
    tone: "danger",
    title: "License revoked",
    meta: ["Clave Suite", "admin"],
    time: "1h ago",
  },
  {
    id: 4,
    icon: Download,
    tone: "muted",
    title: "Download started",
    meta: ["Kepler 0.5.2", "arm64"],
    time: "2h ago",
  },
];

function RecentActivity() {
  return (
    <div className="space-y-3">
      <div className="space-y-0.5">
        <h2 className="text-lg font-semibold tracking-tight">
          Recent Activity
        </h2>
        <p className="text-muted-foreground text-sm">
          Latest events across licenses, updates, and downloads.
        </p>
      </div>
      <div className="divide-border divide-y rounded-md border">
        {activity.map((e) => (
          <div key={e.id} className="flex items-center gap-3 px-4 py-3">
            <span
              className={`flex size-8 shrink-0 items-center justify-center rounded-full ${toneStyles[e.tone]}`}
            >
              <e.icon className="size-4" />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">{e.title}</p>
              <p className="text-muted-foreground truncate text-xs">
                {e.meta.join(" · ")}
              </p>
            </div>
            <span className="text-muted-foreground shrink-0 text-xs whitespace-nowrap">
              {e.time}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

export const Route = createFileRoute("/_dash/dashboard")({
  component: DashboardPage,
});

function DashboardPage() {
  const { product } = useCurrentProduct();
  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
  });

  const { data: overview } = useQuery({
    queryKey: ["adminOverview", product?.id],
    queryFn: () => getAdminOverview(product?.id),
  });

  const isLoading = !overview;

  const stats = [
    { label: "Total Licenses", value: overview?.totalLicenses, icon: Key },
    { label: "Active", value: overview?.activeLicenses, icon: Package },
    {
      label: "Active Trials",
      value: overview?.activeTrials,
      icon: FlaskConical,
    },
    { label: "Expired", value: overview?.expiredLicenses, icon: AlertCircle },
    { label: "Products", value: overview?.totalProducts, icon: Users },
  ];

  return (
    <AdminShell title="Dashboard">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-muted-foreground text-sm">
          {admin ? (
            <>
              Signed in as{" "}
              <span className="text-foreground font-medium">{admin.email}</span>
            </>
          ) : (
            "Your license activity at a glance."
          )}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {stats.map((stat) => (
          <Card key={stat.label}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-muted-foreground text-sm font-medium">
                {stat.label}
              </CardTitle>
              <stat.icon className="text-muted-foreground size-4" />
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Skeleton className="h-8 w-16" />
              ) : (
                <div className="text-2xl font-bold tabular-nums">
                  {stat.value ?? 0}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <React.Suspense
        fallback={<Skeleton className="h-[330px] w-full rounded-xl" />}
      >
        <UsageChart />
      </React.Suspense>

      <RecentActivity />
    </AdminShell>
  );
}
