import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { useCurrentProduct } from "@/features/admin/product-context";
import {
  listAdminLicenses,
  listAdminProducts,
  createLicense,
  deleteLicense,
  type AdminLicenseItem,
  type ListLicensesParams,
} from "@/features/admin/api";

import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Eye, Plus, Search, Trash2 } from "lucide-react";

export const Route = createFileRoute("/_dash/licenses/")({
  component: LicensesPage,
});

function LicensesPage() {
  const queryClient = useQueryClient();
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("all");
  const [type, setType] = useState("all");
  const [productId, setProductId] = useState("");
  const { product: currentProduct } = useCurrentProduct();
  const [page, setPage] = useState(1);
  // Follow the sidebar product selection; the local dropdown can still override.
  useEffect(() => {
    if (currentProduct?.id) {
      setProductId(currentProduct.id);
      setPage(1);
    }
  }, [currentProduct?.id]);
  const pageSize = 20;
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newEmail, setNewEmail] = useState("");
  const [newName, setNewName] = useState("");
  const [newProductId, setNewProductId] = useState("");
  const [newMaxActs, setNewMaxActs] = useState("3");
  const [newType, setNewType] = useState("standard");
  const [newTrialDays, setNewTrialDays] = useState("14");
  const [newSendEmail, setNewSendEmail] = useState(true);
  const [newKey, setNewKey] = useState("");
  const [deleting, setDeleting] = useState<AdminLicenseItem | null>(null);

  const params: ListLicensesParams = {
    q,
    status,
    type,
    productId: productId || undefined,
    page,
    pageSize,
  };

  const { data: licenseData, isLoading } = useQuery({
    queryKey: ["adminLicenses", params],
    queryFn: () => listAdminLicenses(params),
  });

  const { data: products } = useQuery({
    queryKey: ["adminProducts"],
    queryFn: listAdminProducts,
  });

  const createMutation = useMutation({
    mutationFn: async () => {
      const isTrial = newType === "trial";
      return createLicense({
        productId: newProductId,
        maxActivations: parseInt(newMaxActs, 10) || 3,
        customerEmail: newEmail,
        customerName: newName.trim() || undefined,
        isTrial,
        trialDays: isTrial ? parseInt(newTrialDays, 10) || 14 : undefined,
        sendEmail: newSendEmail,
      });
    },
    onSuccess: (data) => {
      setNewKey(data.licenseKey);
      toast.success(data.isTrial ? "Trial created" : "License created");
      queryClient.invalidateQueries({ queryKey: ["adminLicenses"] });
      queryClient.invalidateQueries({ queryKey: ["adminOverview"] });
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : "Failed to create license",
      ),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteLicense(id),
    onSuccess: () => {
      toast.success("License deleted");
      queryClient.invalidateQueries({ queryKey: ["adminLicenses"] });
      queryClient.invalidateQueries({ queryKey: ["adminOverview"] });
      setDeleting(null);
    },
    onError: (err) =>
      toast.error(
        err instanceof Error ? err.message : "Failed to delete license",
      ),
  });

  const totalPages = licenseData
    ? Math.ceil(licenseData.total / licenseData.pageSize)
    : 0;

  const createDialog = (
    <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="size-4" /> Create License
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create License</DialogTitle>
          <DialogDescription>
            Generate a new license key for a customer.
          </DialogDescription>
        </DialogHeader>
        {newKey ? (
          <div className="space-y-3">
            <p className="text-sm font-medium">
              License key (copy it now — it won't be shown again):
            </p>
            <div className="rounded-md bg-muted p-3 font-mono text-sm break-all">
              {newKey}
            </div>
            <Button
              variant="outline"
              onClick={() => navigator.clipboard.writeText(newKey)}
              className="w-full"
            >
              Copy
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="new-name">Customer name</Label>
              <Input
                id="new-name"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Optional"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-email">Customer email</Label>
              <Input
                id="new-email"
                type="email"
                value={newEmail}
                onChange={(e) => setNewEmail(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-product">Product</Label>
              <Select value={newProductId} onValueChange={setNewProductId}>
                <SelectTrigger id="new-product" className="w-full">
                  <SelectValue placeholder="Select product…" />
                </SelectTrigger>
                <SelectContent>
                  {products?.map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="new-max">Max activations</Label>
                <Input
                  id="new-max"
                  type="number"
                  min={1}
                  value={newMaxActs}
                  onChange={(e) => setNewMaxActs(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-type">Type</Label>
                <Select value={newType} onValueChange={setNewType}>
                  <SelectTrigger id="new-type" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="standard">Standard</SelectItem>
                    <SelectItem value="trial">Trial</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            {newType === "trial" && (
              <div className="space-y-2">
                <Label htmlFor="new-trial-days">Trial length (days)</Label>
                <Input
                  id="new-trial-days"
                  type="number"
                  min={1}
                  max={365}
                  value={newTrialDays}
                  onChange={(e) => setNewTrialDays(e.target.value)}
                />
              </div>
            )}
            <div className="flex items-center justify-between rounded-md border p-3">
              <div className="space-y-0.5">
                <Label htmlFor="new-send-email">Email the license</Label>
                <p className="text-muted-foreground text-xs">
                  Send the key and portal link to the customer.
                </p>
              </div>
              <Switch
                id="new-send-email"
                checked={newSendEmail}
                onCheckedChange={setNewSendEmail}
              />
            </div>
          </div>
        )}
        <DialogFooter>
          {newKey ? (
            <Button
              onClick={() => {
                setDialogOpen(false);
                setNewKey("");
                setNewName("");
                setNewEmail("");
                setNewProductId("");
                setNewMaxActs("3");
                setNewType("standard");
                setNewTrialDays("14");
                setNewSendEmail(true);
              }}
            >
              Close
            </Button>
          ) : (
            <Button
              onClick={() => createMutation.mutate()}
              disabled={createMutation.isPending || !newEmail || !newProductId}
            >
              {createMutation.isPending ? "Creating…" : "Create"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  return (
    <AdminShell title="Licenses">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Licenses</h1>
          <p className="text-muted-foreground text-sm">
            Search, filter, and issue license keys.
          </p>
        </div>
        {createDialog}
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1 sm:max-w-sm">
          <Search className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
          <Input
            placeholder="Search name, email, or product…"
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
            className="pl-8"
          />
        </div>
        <Select
          value={status}
          onValueChange={(v) => {
            setStatus(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="sm:w-36">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="inactive">Inactive</SelectItem>
            <SelectItem value="expired">Expired</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={type}
          onValueChange={(v) => {
            setType(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="sm:w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All types</SelectItem>
            <SelectItem value="standard">Standard</SelectItem>
            <SelectItem value="trial">Trial</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={productId || "all"}
          onValueChange={(v) => {
            setProductId(v === "all" ? "" : v);
            setPage(1);
          }}
        >
          <SelectTrigger className="sm:w-44">
            <SelectValue />
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

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Customer</TableHead>
              <TableHead>Product</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Activations</TableHead>
              <TableHead className="text-right">Expires</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell>
                    <Skeleton className="h-4 w-40" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-4 w-24" />
                  </TableCell>
                  <TableCell>
                    <Skeleton className="h-5 w-16 rounded-full" />
                  </TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="ml-auto h-4 w-10" />
                  </TableCell>
                  <TableCell className="text-right">
                    <Skeleton className="ml-auto h-4 w-20" />
                  </TableCell>
                  <TableCell />
                </TableRow>
              ))
            ) : !licenseData?.items.length ? (
              <TableRow>
                <TableCell
                  colSpan={6}
                  className="text-muted-foreground h-24 text-center"
                >
                  No licenses found
                </TableCell>
              </TableRow>
            ) : (
              licenseData.items.map((l) => (
                <TableRow key={l.id}>
                  <TableCell>
                    <div className="font-medium">
                      {l.customerName || l.customerEmail}
                    </div>
                    {l.customerName && (
                      <div className="text-muted-foreground text-xs">
                        {l.customerEmail}
                      </div>
                    )}
                  </TableCell>
                  <TableCell>{l.productName}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5">
                      <Badge variant={l.isActive ? "default" : "secondary"}>
                        {l.isActive ? "Active" : "Inactive"}
                      </Badge>
                      {l.isTrial && <Badge variant="outline">Trial</Badge>}
                    </div>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">
                    {l.activationCount}/{l.maxActivations}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-right text-sm">
                    {l.expiresAt
                      ? new Date(l.expiresAt).toLocaleDateString()
                      : "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" asChild>
                      <Link
                        to="/licenses/$licenseId"
                        params={{ licenseId: l.id }}
                      >
                        <Eye className="size-4" />
                      </Link>
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setDeleting(l)}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title="Delete license?"
        description={
          <>
            This permanently deletes the license for{" "}
            <strong>{deleting?.customerName || deleting?.customerEmail}</strong>{" "}
            and all its activations. This can't be undone.
          </>
        }
        pending={deleteMutation.isPending}
        onConfirm={() => deleting && deleteMutation.mutate(deleting.id)}
      />

      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Page {page} of {totalPages} ({licenseData?.total} total)
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
    </AdminShell>
  );
}
