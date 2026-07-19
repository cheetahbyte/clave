import * as React from "react";
import { useQuery } from "@tanstack/react-query";
import { Area, AreaChart, CartesianGrid, XAxis } from "recharts";
import { getAdminTimeseries } from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
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

export default function UsageChart() {
  const [range, setRange] = React.useState<Range>("90d");
  const days = rangeDays[range];
  const { product } = useCurrentProduct();
  const { data: series, isLoading } = useQuery({
    queryKey: ["adminTimeseries", days, product?.id],
    queryFn: () => getAdminTimeseries(days, product?.id),
  });
  return (
    <Card className="pt-0">
      <CardHeader className="flex flex-col items-stretch border-b !p-0 sm:flex-row">
        <div className="flex flex-1 flex-col justify-center gap-1 px-6 py-5">
          <CardTitle>License activity</CardTitle>
          <CardDescription>
            Activations and trials over {rangeLabel[range]}
          </CardDescription>
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
          <ChartContainer
            config={chartConfig}
            className="aspect-auto h-[250px] w-full"
          >
            <AreaChart
              data={series ?? []}
              margin={{ left: 12, right: 12, top: 12 }}
            >
              <defs>
                <linearGradient
                  id="fill-activations"
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop
                    offset="5%"
                    stopColor="var(--color-activations)"
                    stopOpacity={0.8}
                  />
                  <stop
                    offset="95%"
                    stopColor="var(--color-activations)"
                    stopOpacity={0.1}
                  />
                </linearGradient>
                <linearGradient id="fill-trials" x1="0" y1="0" x2="0" y2="1">
                  <stop
                    offset="5%"
                    stopColor="var(--color-trials)"
                    stopOpacity={0.8}
                  />
                  <stop
                    offset="95%"
                    stopColor="var(--color-trials)"
                    stopOpacity={0.1}
                  />
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
                  new Date(value).toLocaleDateString("en-US", {
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
                      new Date(value).toLocaleDateString("en-US", {
                        month: "short",
                        day: "numeric",
                      })
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
