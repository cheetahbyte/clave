import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import {
  listAdminDevices,
  deleteAdminDevice,
  listAdminProducts,
  type AdminDeviceItem,
} from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
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
import { Search, Trash2, Monitor, Laptop, HardDrive } from "lucide-react";

function deviceIcon(hostname: string | null) {
  const name = hostname ?? "";
  if (/mac|book/i.test(name)) return Laptop;
  if (/linux|server|ubuntu/i.test(name)) return HardDrive;
  return Monitor;
}

export const Route = createFileRoute("/_dash/devices/")({
  component: DevicesPage,
});

function DevicesPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const pageSize = 30;
  const [search, setSearch] = useState("");
  const [productFilter, setProductFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [deleting, setDeleting] = useState<AdminDeviceItem | null>(null);
  const { product } = useCurrentProduct();

  // Follow the sidebar product selection; the local dropdown can still override.
  useEffect(() => {
    if (product?.id) {
      setProductFilter(product.id);
      setPage(1);
    }
  }, [product?.id]);

  const { data: products } = useQuery({
    queryKey: ["adminProducts"],
    queryFn: listAdminProducts,
  });

  const params = {
    page,
    pageSize,
    q: search.trim() || undefined,
    productId: productFilter !== "all" ? productFilter : undefined,
    status: statusFilter !== "all" ? statusFilter : undefined,
  };

  const { data, isLoading } = useQuery({
    queryKey: ["adminDevices", params],
    queryFn: () => listAdminDevices(params),
    refetchOnMount: "always",
    staleTime: 0,
  });

  const deleteMut = useMutation({
    mutationFn: (deviceId: string) => deleteAdminDevice(deviceId),
    onSuccess: () => {
      toast.success("Device removed");
      queryClient.invalidateQueries({ queryKey: ["adminDevices"] });
      queryClient.invalidateQueries({ queryKey: ["adminLicenses"] });
      queryClient.invalidateQueries({ queryKey: ["adminOverview"] });
      setDeleting(null);
    },
    onError: (e) =>
      toast.error(
        e instanceof Error ? e.message : "Failed to deactivate device",
      ),
  });

  const totalPages = data ? Math.ceil(data.total / data.pageSize) : 0;

  return (
    <AdminShell title="Devices">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Devices</h1>
        <p className="text-muted-foreground text-sm">
          All activated devices across your organization.
        </p>
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <div className="flex-1 min-w-[200px] max-w-sm">
          <Label htmlFor="device-search" className="sr-only">
            Search
          </Label>
          <div className="relative">
            <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
            <Input
              id="device-search"
              placeholder="Search hostname, customer, product…"
              className="pl-8"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
            />
          </div>
        </div>
        <div className="w-40">
          <Label htmlFor="device-product" className="sr-only">
            Product
          </Label>
          <Select
            value={productFilter}
            onValueChange={(v) => {
              setProductFilter(v);
              setPage(1);
            }}
          >
            <SelectTrigger id="device-product">
              <SelectValue placeholder="All products" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All products</SelectItem>
              {products?.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="w-36">
          <Label htmlFor="device-status" className="sr-only">
            Status
          </Label>
          <Select
            value={statusFilter}
            onValueChange={(v) => {
              setStatusFilter(v);
              setPage(1);
            }}
          >
            <SelectTrigger id="device-status">
              <SelectValue placeholder="All" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All devices</SelectItem>
              <SelectItem value="seen">Recently seen</SelectItem>
              <SelectItem value="never_seen">Never seen</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Device</TableHead>
              <TableHead>Customer</TableHead>
              <TableHead>Product</TableHead>
              <TableHead>License</TableHead>
              <TableHead>Activated</TableHead>
              <TableHead>Last seen</TableHead>
              <TableHead className="w-0" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 8 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-36" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-40" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-20" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-8" />
                  </TableCell>
                </TableRow>
              ))
            ) : !data?.items.length ? (
              <TableRow>
                <TableCell
                  colSpan={7}
                  className="text-muted-foreground h-24 text-center"
                >
                  No devices found
                </TableCell>
              </TableRow>
            ) : (
              data.items.map((d) => {
                const Icon = deviceIcon(d.hostname);
                return (
                  <TableRow key={d.deviceId}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="bg-muted grid size-7 shrink-0 place-items-center rounded-md">
                          <Icon className="text-muted-foreground size-3.5" />
                        </div>
                        <span className="text-sm font-medium">
                          {d.hostname ?? "Unknown device"}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm font-medium">
                        {d.customerName || d.customerEmail || "—"}
                      </div>
                      {d.customerName && (
                        <div className="text-muted-foreground text-xs">
                          {d.customerEmail}
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="text-sm">{d.productName}</TableCell>
                    <TableCell className="font-mono text-muted-foreground text-xs">
                      <Link
                        to="/licenses/$licenseId"
                        params={{ licenseId: d.licenseId }}
                        className="hover:underline"
                      >
                        {d.licenseId.slice(0, 8)}
                      </Link>
                      {!d.licenseActive && (
                        <Badge
                          variant="secondary"
                          className="ml-1.5 text-[10px]"
                        >
                          Inactive
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {d.activatedAt
                        ? new Date(d.activatedAt).toLocaleDateString()
                        : "—"}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm whitespace-nowrap">
                      {d.checkedInAt
                        ? new Date(d.checkedInAt).toLocaleDateString()
                        : "Never"}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setDeleting(d)}
                        title="Deactivate device"
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })
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
            <Button
              variant="outline"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
            >
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

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={() => setDeleting(null)}
        title="Deactivate device?"
        description={
          deleting
            ? `This will deactivate "${deleting.hostname ?? "Unknown device"}" and free its activation seat.`
            : ""
        }
        pending={deleteMut.isPending}
        onConfirm={() => deleting && deleteMut.mutate(deleting.deviceId)}
      />
    </AdminShell>
  );
}
