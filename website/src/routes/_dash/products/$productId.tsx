import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import {
  getCurrentAdmin,
  getAdminProduct,
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Pencil, Trash2, Image as ImageIcon } from "lucide-react";

export const Route = createFileRoute("/_dash/products/$productId")({
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
      actions={
        product ? (
          <>
            <Button variant="outline" onClick={() => setEditOpen(true)}>
              <Pencil className="size-4" /> Edit
            </Button>
            <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
              <Trash2 className="size-4" /> Delete
            </Button>
          </>
        ) : null
      }
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
          </div>
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
