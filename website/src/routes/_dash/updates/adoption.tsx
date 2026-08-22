import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  XAxis,
  YAxis,
} from "recharts";
import {
  ChartNoAxesColumnIncreasing,
  Monitor,
  RefreshCw,
  Tags,
} from "lucide-react";
import {
  getVersionAdoption,
  type VersionAdoptionResponse,
} from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
import { AdminShell } from "@/components/admin/AdminShell";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const DAYS = 30;
const CHART_COLORS = [
  "var(--chart-1)",
  "var(--chart-2)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
] as const;

const relativeTime = new Intl.RelativeTimeFormat(undefined, {
  numeric: "auto",
});

function formatRelativeTime(value: string) {
  const elapsedSeconds = (new Date(value).getTime() - Date.now()) / 1000;
  const ranges: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 60 * 60 * 24 * 365],
    ["month", 60 * 60 * 24 * 30],
    ["day", 60 * 60 * 24],
    ["hour", 60 * 60],
    ["minute", 60],
  ];
  for (const [unit, seconds] of ranges) {
    if (Math.abs(elapsedSeconds) >= seconds) {
      return relativeTime.format(Math.round(elapsedSeconds / seconds), unit);
    }
  }
  return relativeTime.format(Math.round(elapsedSeconds), "second");
}

type VersionSeries = {
  key: string;
  version: string;
  color: string;
};

function useChartData(data: VersionAdoptionResponse | undefined) {
  return React.useMemo(() => {
    const series: VersionSeries[] = (data?.distribution ?? []).map(
      (entry, index) => ({
        key: `version_${index}`,
        version: entry.version,
        color: CHART_COLORS[index % CHART_COLORS.length],
      }),
    );
    const keyByVersion = new Map(
      series.map((item) => [item.version, item.key]),
    );
    const trend = (data?.trend ?? []).map((point) => {
      const row: Record<string, string | number> = { date: point.date };
      for (const item of series) row[item.key] = 0;
      for (const value of point.versions) {
        const key = keyByVersion.get(value.version);
        if (key) row[key] = value.deviceCount;
      }
      return row;
    });
    const config = Object.fromEntries(
      series.map((item) => [
        item.key,
        { label: item.version, color: item.color },
      ]),
    ) satisfies ChartConfig;
    return { series, trend, config };
  }, [data]);
}

function LoadingState() {
  return (
    <div className="flex flex-col gap-6">
      <div className="grid gap-4 sm:grid-cols-2">
        <Skeleton className="h-32 rounded-xl" />
        <Skeleton className="h-32 rounded-xl" />
      </div>
      <div className="grid gap-6 xl:grid-cols-2">
        <Skeleton className="h-[360px] rounded-xl" />
        <Skeleton className="h-[360px] rounded-xl" />
      </div>
      <Skeleton className="h-80 rounded-xl" />
    </div>
  );
}

function SummaryCards({ data }: { data: VersionAdoptionResponse }) {
  const cards = [
    {
      title: "Active devices",
      description: `Reported during the last ${DAYS} days`,
      value: data.activeDevices,
      icon: Monitor,
    },
    {
      title: "Versions in use",
      description: "Exact versions reported by active devices",
      value: data.versionCount,
      icon: Tags,
    },
  ];
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {cards.map((card) => (
        <Card key={card.title}>
          <CardHeader>
            <div className="flex items-center justify-between gap-4">
              <CardTitle>{card.title}</CardTitle>
              <card.icon
                className="text-muted-foreground size-4"
                aria-hidden="true"
              />
            </div>
            <CardDescription>{card.description}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-semibold tabular-nums">
              {card.value}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function DistributionChart({
  data,
  series,
}: {
  data: VersionAdoptionResponse;
  series: VersionSeries[];
}) {
  const config = {
    deviceCount: { label: "Devices", color: "var(--chart-1)" },
  } satisfies ChartConfig;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Current distribution</CardTitle>
        <CardDescription>
          Latest reported version per active device
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ChartContainer config={config} className="h-[260px] w-full">
          <BarChart
            accessibilityLayer
            data={data.distribution}
            layout="vertical"
            margin={{ left: 12, right: 24 }}
          >
            <CartesianGrid horizontal={false} />
            <XAxis
              type="number"
              allowDecimals={false}
              axisLine={false}
              tickLine={false}
            />
            <YAxis
              type="category"
              dataKey="version"
              width={88}
              axisLine={false}
              tickLine={false}
              tickMargin={8}
            />
            <ChartTooltip
              cursor={false}
              content={
                <ChartTooltipContent
                  hideLabel
                  formatter={(value, _name, item) => (
                    <div className="flex min-w-36 items-center justify-between gap-4">
                      <span>{item.payload.version}</span>
                      <span className="font-mono font-medium tabular-nums">
                        {String(value)} · {item.payload.percentage}%
                      </span>
                    </div>
                  )}
                />
              }
            />
            <Bar dataKey="deviceCount" radius={4}>
              {data.distribution.map((entry, index) => (
                <Cell
                  key={entry.version}
                  fill={series[index]?.color ?? CHART_COLORS[0]}
                />
              ))}
            </Bar>
          </BarChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

function TrendChart({
  trend,
  config,
  series,
}: {
  trend: Record<string, string | number>[];
  config: ChartConfig;
  series: VersionSeries[];
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Daily adoption</CardTitle>
        <CardDescription>
          Daily active devices grouped by reported version
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ChartContainer config={config} className="h-[260px] w-full">
          <AreaChart
            accessibilityLayer
            data={trend}
            margin={{ left: 8, right: 8, top: 8 }}
          >
            <CartesianGrid vertical={false} />
            <XAxis
              dataKey="date"
              axisLine={false}
              tickLine={false}
              tickMargin={8}
              minTickGap={24}
              tickFormatter={(value) =>
                new Date(`${value}T00:00:00`).toLocaleDateString(undefined, {
                  month: "short",
                  day: "numeric",
                })
              }
            />
            <ChartTooltip
              cursor={false}
              content={
                <ChartTooltipContent
                  indicator="dot"
                  labelFormatter={(value) =>
                    new Date(`${value}T00:00:00`).toLocaleDateString(
                      undefined,
                      {
                        month: "short",
                        day: "numeric",
                        year: "numeric",
                      },
                    )
                  }
                />
              }
            />
            {series.map((item) => (
              <Area
                key={item.key}
                dataKey={item.key}
                type="monotone"
                stackId="versions"
                fill={`var(--color-${item.key})`}
                fillOpacity={0.35}
                stroke={`var(--color-${item.key})`}
                strokeWidth={2}
              />
            ))}
            <ChartLegend content={<ChartLegendContent />} />
          </AreaChart>
        </ChartContainer>
      </CardContent>
    </Card>
  );
}

function DeviceTable({ data }: { data: VersionAdoptionResponse }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Reporting devices</CardTitle>
        <CardDescription>
          Current state for devices seen during the last {DAYS} days
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Device</TableHead>
                <TableHead>Version</TableHead>
                <TableHead>Build</TableHead>
                <TableHead>Platform</TableHead>
                <TableHead>Architecture</TableHead>
                <TableHead>OS version</TableHead>
                <TableHead className="text-right">Last check-in</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.devices.map((device) => (
                <TableRow key={device.activationId}>
                  <TableCell className="font-medium">
                    {device.hostname || "Unknown"}
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {device.version}
                  </TableCell>
                  <TableCell>{device.build || "Unknown"}</TableCell>
                  <TableCell>{device.platform || "Unknown"}</TableCell>
                  <TableCell>{device.arch || "Unknown"}</TableCell>
                  <TableCell>{device.osVersion || "Unknown"}</TableCell>
                  <TableCell className="text-muted-foreground text-right whitespace-nowrap">
                    <time
                      dateTime={device.lastCheckin}
                      title={new Date(device.lastCheckin).toLocaleString()}
                    >
                      {formatRelativeTime(device.lastCheckin)}
                    </time>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
}

export const Route = createFileRoute("/_dash/updates/adoption")({
  component: VersionAdoptionPage,
});

function VersionAdoptionPage() {
  const { product } = useCurrentProduct();
  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ["versionAdoption", product?.id, DAYS],
    queryFn: () => getVersionAdoption(product?.id, DAYS),
  });
  const { series, trend, config } = useChartData(data);

  return (
    <AdminShell
      title="Version Adoption"
      breadcrumbs={[{ label: "Updates", to: "/updates" }]}
    >
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">
          Version Adoption
        </h1>
        <p className="text-muted-foreground text-sm">
          {product
            ? `${product.name} installations active during the last ${DAYS} days.`
            : "Client versions active during the last 30 days."}
        </p>
      </div>

      {isLoading ? <LoadingState /> : null}

      {isError ? (
        <Alert variant="destructive">
          <AlertTitle>Version adoption is unavailable</AlertTitle>
          <AlertDescription className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center">
            The diagnostics could not be loaded. Try the request again.
            <Button
              variant="outline"
              size="sm"
              disabled={isFetching}
              onClick={() => refetch()}
            >
              <RefreshCw data-icon="inline-start" />
              Retry
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      {data && data.activeDevices === 0 ? (
        <Card>
          <CardContent>
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <ChartNoAxesColumnIncreasing />
                </EmptyMedia>
                <EmptyTitle>No version diagnostics yet</EmptyTitle>
                <EmptyDescription>
                  Clients appear here after an authorized{" "}
                  <code>/client/sync</code> request includes the{" "}
                  <code>version</code> field.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          </CardContent>
        </Card>
      ) : null}

      {data && data.activeDevices > 0 ? (
        <div className="flex flex-col gap-6">
          <SummaryCards data={data} />
          <div className="grid gap-6 xl:grid-cols-2">
            <DistributionChart data={data} series={series} />
            <TrendChart trend={trend} config={config} series={series} />
          </div>
          <DeviceTable data={data} />
        </div>
      ) : null}
    </AdminShell>
  );
}
