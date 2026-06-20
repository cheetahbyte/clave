import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import {
  getAdminProduct,
  updateProduct,
  deleteProduct,
  listProductFeatures,
  createProductFeature,
  deleteProductFeature,
  listFeatureWindows,
  createFeatureWindow,
  deleteFeatureWindow,
  type AdminProductItem,
  type ProductFeature,
  type FeatureWindow,
} from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Pencil, Trash2, Image as ImageIcon, Plus, Clock } from "lucide-react";

export const Route = createFileRoute("/_dash/products/$productId")({
  component: ProductDetailPage,
});

function ProductDetailPage() {
  const { productId } = Route.useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const { data: product, isLoading } = useQuery({
    queryKey: ["adminProduct", productId],
    queryFn: () => getAdminProduct(productId),
  });

  const deleteMut = useMutation({
    mutationFn: () => deleteProduct(productId),
    onSuccess: () => {
      toast.success("Product deleted");
      queryClient.invalidateQueries({ queryKey: ["adminProducts"] });
      navigate({ to: "/products" });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete product"),
  });

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["adminProduct", productId] });
    queryClient.invalidateQueries({ queryKey: ["adminProducts"] });
  }

  return (
    <AdminShell
      title="Details"
      breadcrumbs={[{ label: "Products", to: "/products" }]}
    >
      <EditProductDialog
        product={product ?? null}
        open={editOpen}
        onOpenChange={setEditOpen}
        onSaved={invalidate}
      />
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete product?"
        description={
          <>
            This permanently deletes <strong>{product?.name}</strong>. Products with existing
            licenses can't be deleted — remove their licenses first.
          </>
        }
        pending={deleteMut.isPending}
        onConfirm={() => deleteMut.mutate()}
      />
      {isLoading ? (
        <div className="space-y-8">
          <Skeleton className="h-8 w-64" />
        </div>
      ) : !product ? (
        <p className="text-muted-foreground">Product not found</p>
      ) : (
        <div className="space-y-8">
          <div className="flex items-start gap-4">
            <div className="grid size-12 shrink-0 place-items-center overflow-hidden rounded-xl border bg-white">
              {product.logoUrl ? (
                <img src={product.logoUrl} alt="" className="h-full w-full object-contain" />
              ) : (
                <ImageIcon className="size-6 text-slate-300" />
              )}
            </div>
            <div className="min-w-0 flex-1 space-y-1.5">
              <h1 className="text-2xl font-semibold tracking-tight">{product.name}</h1>
              <div className="flex flex-wrap items-center gap-3 text-muted-foreground text-sm">
                {product.version && <span>v{product.version}</span>}
                <span className="font-mono text-xs">{product.id.slice(0, 8)}</span>
                <span>
                  Created {product.createdAt ? new Date(product.createdAt).toLocaleDateString() : "—"}
                </span>
              </div>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <Button variant="outline" onClick={() => setEditOpen(true)}>
                <Pencil className="size-4" /> Edit
              </Button>
              <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="size-4" /> Delete
              </Button>
            </div>
          </div>

          <ProductFeaturesSection productId={product.id} />
          <FeatureWindowsSection productId={product.id} />
        </div>
      )}
    </AdminShell>
  );
}

function EditProductDialog({
  product,
  open,
  onOpenChange,
  onSaved,
}: {
  product: AdminProductItem | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(product?.name ?? "");
  const [version, setVersion] = useState(product?.version ?? "");
  const [logoUrl, setLogoUrl] = useState(product?.logoUrl ?? "");
  const [logoError, setLogoError] = useState(false);

  const mutation = useMutation({
    mutationFn: () => {
      const v = version.trim() || null;
      const l = logoUrl.trim() || null;
      return updateProduct(product!.id, name.trim(), v, l);
    },
    onSuccess: () => {
      toast.success("Product updated");
      onSaved();
      onOpenChange(false);
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to save product"),
  });

  if (!product) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit product</DialogTitle>
          <DialogDescription>Update product details.</DialogDescription>
        </DialogHeader>
        <form onSubmit={(e) => { e.preventDefault(); if (name.trim()) mutation.mutate(); }}>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input id="edit-name" value={name} autoFocus placeholder="Kepler" onChange={(e) => setName(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-version">Version</Label>
              <Input id="edit-version" value={version} placeholder="1.0.0" onChange={(e) => setVersion(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-logo-url">Logo URL</Label>
              <Input id="edit-logo-url" value={logoUrl} placeholder="https://example.com/logo.png" onChange={(e) => setLogoUrl(e.target.value)} />
              {logoUrl.trim() && (
                <div className="mt-2 flex size-12 items-center justify-center overflow-hidden rounded-xl border bg-white">
                  {logoError ? (
                    <ImageIcon className="size-6 text-slate-300" />
                  ) : (
                    <img src={logoUrl.trim()} alt="Logo" className="h-full w-full object-contain" onError={() => setLogoError(true)} />
                  )}
                </div>
              )}
            </div>
          </div>
          <DialogFooter className="mt-4">
            <Button type="submit" disabled={mutation.isPending || !name.trim()}>
              {mutation.isPending ? "Saving…" : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

// ============ Product Features Section ============

function ProductFeaturesSection({ productId }: { productId: string }) {
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [newName, setNewName] = useState("");
  const [newDesc, setNewDesc] = useState("");
  const [deleting, setDeleting] = useState<ProductFeature | null>(null);

  const { data: features = [], isLoading } = useQuery({
    queryKey: ["productFeatures", productId],
    queryFn: () => listProductFeatures(productId),
  });

  const createMut = useMutation({
    mutationFn: () =>
      createProductFeature(productId, {
        key: newKey.trim(),
        name: newName.trim() || undefined,
        description: newDesc.trim() || undefined,
      }),
    onSuccess: () => {
      toast.success("Feature created");
      queryClient.invalidateQueries({ queryKey: ["productFeatures", productId] });
      setDialogOpen(false);
      setNewKey(""); setNewName(""); setNewDesc("");
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to create feature"),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteProductFeature(productId, id),
    onSuccess: () => {
      toast.success("Feature deleted");
      queryClient.invalidateQueries({ queryKey: ["productFeatures", productId] });
      setDeleting(null);
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete feature"),
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Feature Catalog</h2>
          <p className="text-muted-foreground text-sm">
            Features that can be assigned to licenses and required by update channels.
          </p>
        </div>
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="size-4" /> Add Feature</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add feature</DialogTitle>
              <DialogDescription>Create a new feature key for this product.</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="feat-key">Key</Label>
                <Input id="feat-key" value={newKey} placeholder="pro, beta, early-bird" onChange={(e) => setNewKey(e.target.value)} />
                <p className="text-muted-foreground text-xs">Lowercase, no spaces. This goes into JWTs and API responses.</p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="feat-name">Display name (optional)</Label>
                <Input id="feat-name" value={newName} placeholder="Pro Tier" onChange={(e) => setNewName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="feat-desc">Description (optional)</Label>
                <Input id="feat-desc" value={newDesc} placeholder="Full access to all features" onChange={(e) => setNewDesc(e.target.value)} />
              </div>
            </div>
            <DialogFooter>
              <Button onClick={() => createMut.mutate()} disabled={createMut.isPending || !newKey.trim()}>
                {createMut.isPending ? "Creating…" : "Create"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : features.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center border rounded-lg">
          No features yet. Add one to start gating channels and scheduling campaigns.
        </p>
      ) : (
        <div className="border rounded-lg">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Key</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Description</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {features.map((f) => (
                <TableRow key={f.id}>
                  <TableCell className="font-mono text-sm">{f.key}</TableCell>
                  <TableCell>{f.name || "—"}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">{f.description || "—"}</TableCell>
                  <TableCell>
                    {f.archived ? (
                      <Badge variant="secondary">Archived</Badge>
                    ) : (
                      <Badge variant="default">Active</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setDeleting(f)}
                    >
                      <Trash2 className="size-4 text-destructive" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(o) => !o && setDeleting(null)}
        title="Delete feature?"
        description={
          <>
            This removes <strong className="font-mono">{deleting?.key}</strong> from the catalog.
            Licenses and channels that reference it will lose this feature.
          </>
        }
        pending={deleteMut.isPending}
        onConfirm={() => deleting && deleteMut.mutate(deleting.id)}
      />
    </div>
  );
}

// ============ Feature Windows Section ============

function toDateTimeLocal(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  const off = d.getTimezoneOffset();
  const local = new Date(d.getTime() - off * 60_000);
  return local.toISOString().slice(0, 16);
}

function toRFC3339(localDateTime: string): string {
  if (!localDateTime) return "";
  return new Date(localDateTime).toISOString();
}

function FeatureWindowsSection({ productId }: { productId: string }) {
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<FeatureWindow | null>(null);
  const [featureKey, setFeatureKey] = useState("");
  const [startsAt, setStartsAt] = useState("");
  const [endsAt, setEndsAt] = useState("");
  const [appliesTo, setAppliesTo] = useState("standard");
  const [isActive, setIsActive] = useState(true);
  const [deleting, setDeleting] = useState<FeatureWindow | null>(null);

  const { data: windows = [], isLoading } = useQuery({
    queryKey: ["featureWindows", productId],
    queryFn: () => listFeatureWindows(productId),
  });

  const { data: features = [] } = useQuery({
    queryKey: ["productFeatures", productId],
    queryFn: () => listProductFeatures(productId),
  });

  function resetForm() {
    setFeatureKey(""); setStartsAt(""); setEndsAt("");
    setAppliesTo("standard"); setIsActive(true); setEditing(null);
  }

  function openCreate() {
    resetForm();
    setDialogOpen(true);
  }

  function openEdit(w: FeatureWindow) {
    setEditing(w);
    setFeatureKey(w.featureKey);
    setStartsAt(toDateTimeLocal(w.startsAt));
    setEndsAt(toDateTimeLocal(w.endsAt));
    setAppliesTo(w.appliesTo);
    setIsActive(w.isActive);
    setDialogOpen(true);
  }

  const saveMut = useMutation({
    mutationFn: () => {
      const payload = {
        featureKey: featureKey.trim(),
        startsAt: toRFC3339(startsAt),
        endsAt: toRFC3339(endsAt),
        appliesTo,
        isActive,
      };
      if (editing) return updateFeatureWindow(productId, editing.id, payload);
      return createFeatureWindow(productId, payload);
    },
    onSuccess: () => {
      toast.success(editing ? "Window updated" : "Window created");
      queryClient.invalidateQueries({ queryKey: ["featureWindows", productId] });
      setDialogOpen(false);
      resetForm();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to save window"),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteFeatureWindow(productId, id),
    onSuccess: () => {
      toast.success("Window deleted");
      queryClient.invalidateQueries({ queryKey: ["featureWindows", productId] });
      setDeleting(null);
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete window"),
  });

  const now = new Date();

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Scheduled Features</h2>
          <p className="text-muted-foreground text-sm">
            Automatically assign features to licenses created during a time window.
          </p>
        </div>
        <Button onClick={openCreate}>
          <Clock className="size-4" /> New Window
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : windows.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center border rounded-lg">
          No scheduled feature windows. Create one to run a campaign (e.g. "all licenses
          bought during launch week get early-support").
        </p>
      ) : (
        <div className="border rounded-lg">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Feature</TableHead>
                <TableHead>Starts</TableHead>
                <TableHead>Ends</TableHead>
                <TableHead>Applies to</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="w-20" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {windows.map((w) => {
                const start = w.startsAt ? new Date(w.startsAt) : null;
                const end = w.endsAt ? new Date(w.endsAt) : null;
                const isCurrent = w.isActive && start && end && now >= start && now < end;
                const isFuture = w.isActive && start && now < start;
                return (
                  <TableRow key={w.id}>
                    <TableCell className="font-mono text-sm">{w.featureKey}</TableCell>
                    <TableCell className="text-sm">{start ? start.toLocaleDateString() : "—"}</TableCell>
                    <TableCell className="text-sm">{end ? end.toLocaleDateString() : "—"}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="capitalize">{w.appliesTo}</Badge>
                    </TableCell>
                    <TableCell>
                      {isCurrent ? (
                        <Badge variant="default">Active now</Badge>
                      ) : isFuture ? (
                        <Badge variant="secondary">Upcoming</Badge>
                      ) : !w.isActive ? (
                        <Badge variant="secondary">Disabled</Badge>
                      ) : (
                        <Badge variant="secondary">Ended</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button variant="ghost" size="icon" onClick={() => openEdit(w)}>
                          <Pencil className="size-4" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => setDeleting(w)}>
                          <Trash2 className="size-4 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      <Dialog open={dialogOpen} onOpenChange={(o) => { setDialogOpen(o); if (!o) resetForm(); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editing ? "Edit feature window" : "New feature window"}</DialogTitle>
            <DialogDescription>
              Licenses created during this window will automatically receive the feature.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="win-feature">Feature</Label>
              {features.length > 0 ? (
                <Select value={featureKey} onValueChange={setFeatureKey}>
                  <SelectTrigger id="win-feature">
                    <SelectValue placeholder="Select feature…" />
                  </SelectTrigger>
                  <SelectContent>
                    {features.filter((f) => !f.archived).map((f) => (
                      <SelectItem key={f.id} value={f.key}>{f.key}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  id="win-feature"
                  value={featureKey}
                  placeholder="feature key"
                  onChange={(e) => setFeatureKey(e.target.value)}
                />
              )}
              {features.length === 0 && (
                <p className="text-muted-foreground text-xs">
                  Add features to the catalog first, or type a key directly.
                </p>
              )}
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="win-start">Starts</Label>
                <Input id="win-start" type="datetime-local" value={startsAt} onChange={(e) => setStartsAt(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="win-end">Ends</Label>
                <Input id="win-end" type="datetime-local" value={endsAt} onChange={(e) => setEndsAt(e.target.value)} />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="win-applies">Applies to</Label>
              <Select value={appliesTo} onValueChange={setAppliesTo}>
                <SelectTrigger id="win-applies">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="standard">Standard licenses only</SelectItem>
                  <SelectItem value="trial">Trials only</SelectItem>
                  <SelectItem value="all">All licenses</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between rounded-md border p-3">
              <div className="space-y-0.5">
                <Label htmlFor="win-active">Active</Label>
                <p className="text-muted-foreground text-xs">Inactive windows don't apply to new licenses.</p>
              </div>
              <Switch id="win-active" checked={isActive} onCheckedChange={setIsActive} />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={() => saveMut.mutate()}
              disabled={saveMut.isPending || !featureKey.trim() || !startsAt || !endsAt}
            >
              {saveMut.isPending ? "Saving…" : editing ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(o) => !o && setDeleting(null)}
        title="Delete feature window?"
        description={
          <>
            This stops <strong className="font-mono">{deleting?.featureKey}</strong> from being
            assigned to new licenses. Existing licenses keep their features.
          </>
        }
        pending={deleteMut.isPending}
        onConfirm={() => deleting && deleteMut.mutate(deleting.id)}
      />
    </div>
  );
}
