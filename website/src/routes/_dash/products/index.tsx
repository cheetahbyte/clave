import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import {
  listAdminProducts,
  createProduct,
  updateProduct,
  deleteProduct,
  type AdminProductItem,
} from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Pencil, Plus, Trash2, Image as ImageIcon, ExternalLink } from "lucide-react";

export const Route = createFileRoute("/_dash/products/")({
  component: ProductsPage,
});

function ProductsPage() {
  const queryClient = useQueryClient();
  const { data: products, isLoading } = useQuery({
    queryKey: ["adminProducts"],
    queryFn: listAdminProducts,
  });

  const [editing, setEditing] = useState<AdminProductItem | "new" | null>(null);
  const [deleting, setDeleting] = useState<AdminProductItem | null>(null);

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["adminProducts"] });
  }

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteProduct(id),
    onSuccess: () => {
      toast.success("Product deleted");
      invalidate();
      setDeleting(null);
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete product"),
  });

  return (
    <AdminShell title="Products">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Products</h1>
          <p className="text-muted-foreground text-sm">Products you can issue licenses for.</p>
        </div>
        <Button onClick={() => setEditing("new")}>
          <Plus className="size-4" /> New product
        </Button>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Version</TableHead>
              <TableHead>ID</TableHead>
              <TableHead>Created</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              [{ id: "sk1" }, { id: "sk2" }, { id: "sk3" }].map((s) => (
                <TableRow key={s.id}>
                  <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-48" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                  <TableCell />
                </TableRow>
              ))
            ) : !products?.length ? (
              <TableRow>
                <TableCell colSpan={5} className="text-muted-foreground h-24 text-center">
                  No products
                </TableCell>
              </TableRow>
            ) : (
              products.map((p) => (
                <TableRow key={p.id}>
                  <TableCell className="font-medium">
                    <Link
                      to="/products/$productId"
                      params={{ productId: p.id }}
                      className="text-primary hover:underline inline-flex items-center gap-1"
                    >
                      {p.name}
                      <ExternalLink className="size-3" />
                    </Link>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">{p.version ?? "—"}</TableCell>
                  <TableCell className="text-muted-foreground font-mono text-xs">{p.id}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {p.createdAt ? new Date(p.createdAt).toLocaleDateString() : "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => setEditing(p)}>
                      <Pencil className="size-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setDeleting(p)}
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

      <ProductDialog
        key={editing === "new" ? "new" : editing?.id ?? "closed"}
        target={editing}
        onOpenChange={(open) => !open && setEditing(null)}
        onSaved={() => {
          invalidate();
          setEditing(null);
        }}
      />

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => !open && setDeleting(null)}
        title="Delete product?"
        description={
          <>
            This permanently deletes <strong>{deleting?.name}</strong>. Products with existing
            licenses can't be deleted — remove their licenses first.
          </>
        }
        pending={deleteMut.isPending}
        onConfirm={() => deleting && deleteMut.mutate(deleting.id)}
      />
    </AdminShell>
  );
}

function ProductDialog({
  target,
  onOpenChange,
  onSaved,
}: {
  target: AdminProductItem | "new" | null;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const isNew = target === "new";
  const product = target && target !== "new" ? target : null;
  const [name, setName] = useState(product?.name ?? "");
  const [version, setVersion] = useState(product?.version ?? "");
  const [logoUrl, setLogoUrl] = useState(product?.logoUrl ?? "");
  const [logoError, setLogoError] = useState(false);

  const mutation = useMutation({
    mutationFn: () => {
      const v = version.trim() || null;
      const l = logoUrl.trim() || null;
      return isNew ? createProduct(name.trim(), v, l) : updateProduct(product!.id, name.trim(), v, l);
    },
    onSuccess: () => {
      toast.success(isNew ? "Product created" : "Product updated");
      onSaved();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to save product"),
  });

  return (
    <Dialog open={target !== null} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isNew ? "New product" : "Edit product"}</DialogTitle>
          <DialogDescription>
            {isNew ? "Add a product you can issue licenses for." : "Update product details."}
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (name.trim()) mutation.mutate();
          }}
        >
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="product-name">Name</Label>
              <Input
                id="product-name"
                value={name}
                autoFocus
                placeholder="Kepler"
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="product-version">Version (optional)</Label>
              <Input
                id="product-version"
                value={version}
                placeholder="1.0.0"
                onChange={(e) => setVersion(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="product-logo-url">Logo image URL (optional)</Label>
              <Input
                id="product-logo-url"
                value={logoUrl}
                placeholder="https://example.com/logo.png"
                onChange={(e) => setLogoUrl(e.target.value)}
              />
              {logoUrl.trim() && (
                <div className="mt-2 flex size-12 items-center justify-center overflow-hidden rounded-xl border bg-white">
                  {logoError ? (
                    <ImageIcon className="size-6 text-slate-300" />
                  ) : (
                    <img
                      src={logoUrl.trim()}
                      alt="Logo preview"
                      className="h-full w-full object-contain"
                      onError={() => setLogoError(true)}
                    />
                  )}
                </div>
              )}
            </div>
          </div>
          <DialogFooter className="mt-4">
            <Button type="submit" disabled={mutation.isPending || !name.trim()}>
              {mutation.isPending ? "Saving…" : isNew ? "Create" : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
