import { createFileRoute, redirect } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts";
import { getCurrentAdmin } from "@/features/admin/api";
import { getAdminOverview, getAdminTimeseries } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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

const chartConfig = {
  activations: { label: "Activations", color: "var(--chart-2)" },
  trials: { label: "Trials started", color: "var(--chart-4)" },
} satisfies ChartConfig;

type Range = "90d" | "30d" | "7d";

const rangeDays: Record<Range, number> = { "90d": 90, "30d": 30, "7d": 7 };
const rangeLabel: Record<Range, string> = {
  "90d": "the last 3 months",
  "30d": "the last 30 days",
  "7d": "the last 7 days",
};

function UsageChart() {
  const [range, setRange] = React.useState<Range>("90d");
  const days = rangeDays[range];

  const { data: series, isLoading } = useQuery({
    queryKey: ["adminTimeseries", days],
    queryFn: () => getAdminTimeseries(days),
  });

  const data = series ?? [];

  return (
    <Card className="pt-0">
      <CardHeader className="flex flex-col items-stretch border-b !p-0 sm:flex-row">
        <div className="flex flex-1 flex-col justify-center gap-1 px-6 py-5">
          <CardTitle>License activity</CardTitle>
          <CardDescription>Activations and trials over {rangeLabel[range]}</CardDescription>
        </div>
        <div className="flex items-center px-6 pb-4 sm:pb-0 sm:pr-6">
          <Select value={range} onValueChange={(v) => setRange(v as Range)}>
            <SelectTrigger className="w-40" aria-label="Select range">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="90d">Last 3 months</SelectItem>
              <SelectItem value="30d">Last 30 days</SelectItem>
              <SelectItem value="7d">Last 7 days</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </CardHeader>
      <CardContent className="px-2 pt-4 sm:px-6 sm:pt-6">
        {isLoading ? (
          <Skeleton className="h-[250px] w-full" />
        ) : (
          <ChartContainer config={chartConfig} className="aspect-auto h-[250px] w-full">
            <AreaChart data={data} margin={{ left: 12, right: 12, top: 12 }}>
              <defs>
                <linearGradient id="fill-activations" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-activations)" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="var(--color-activations)" stopOpacity={0.1} />
                </linearGradient>
                <linearGradient id="fill-trials" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="var(--color-trials)" stopOpacity={0.8} />
                  <stop offset="95%" stopColor="var(--color-trials)" stopOpacity={0.1} />
                </linearGradient>
              </defs>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="date"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                minTickGap={32}
                tickFormatter={(value) =>
                  new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric" })
                }
              />
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    indicator="dot"
                    labelFormatter={(value) =>
                      new Date(value).toLocaleDateString("en-US", { month: "short", day: "numeric" })
                    }
                  />
                }
              />
              <Area
                dataKey="activations"
                type="monotone"
                fill="url(#fill-activations)"
                stroke="var(--color-activations)"
                strokeWidth={2}
                stackId="a"
              />
              <Area
                dataKey="trials"
                type="monotone"
                fill="url(#fill-trials)"
                stroke="var(--color-trials)"
                strokeWidth={2}
                stackId="a"
              />
              <ChartLegend content={<ChartLegendContent />} />
            </AreaChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  );
}

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
        <h2 className="text-lg font-semibold tracking-tight">Recent Activity</h2>
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
  component: DashboardPage,
});

function DashboardPage() {
  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
  });

  const { data: overview } = useQuery({
    queryKey: ["adminOverview"],
    queryFn: getAdminOverview,
  });

  const isLoading = !overview;

  const stats = [
    { label: "Total Licenses", value: overview?.totalLicenses, icon: Key },
    { label: "Active", value: overview?.activeLicenses, icon: Package },
    { label: "Active Trials", value: overview?.activeTrials, icon: FlaskConical },
    { label: "Expired", value: overview?.expiredLicenses, icon: AlertCircle },
    { label: "Products", value: overview?.totalProducts, icon: Users },
  ];

  return (
    <AdminShell title="Dashboard">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Overview</h1>
        <p className="text-muted-foreground text-sm">
          {admin ? (
            <>Signed in as <span className="text-foreground font-medium">{admin.email}</span></>
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
                <div className="text-2xl font-bold tabular-nums">{stat.value ?? 0}</div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <UsageChart />

      <RecentActivity />
    </AdminShell>
  );
}
