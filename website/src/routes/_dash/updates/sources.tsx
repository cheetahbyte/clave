import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import {
  listProductUpdateConfigs,
  getProductStorageConfig,
  deleteProductUpdateConfig,
  type AdminProductItem,
  type ProductUpdateConfigDTO,
} from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
import { UpdateConfigDialog } from "@/features/admin/UpdateConfigDialog";
import { StorageConfigDialog } from "@/features/admin/StorageConfigDialog";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { AdminShell } from "@/components/admin/AdminShell";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ExternalLink, Plus, Pencil, Trash2, Globe, HardDrive, Cloud, Terminal } from "lucide-react";
import { cn } from "@/lib/utils";

function AppleIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden>
      <path d="M16.365 1.43c0 1.14-.493 2.27-1.177 3.08-.744.9-1.99 1.57-2.987 1.57-.12 0-.23-.02-.3-.03-.01-.06-.04-.22-.04-.39 0-1.15.572-2.27 1.206-2.98.804-.94 2.142-1.64 3.248-1.68.03.13.05.28.05.43zm4.565 15.71c-.03.07-.463 1.58-1.518 3.12-.945 1.34-1.94 2.71-3.43 2.71-1.517 0-1.9-.88-3.63-.88-1.698 0-2.302.91-3.67.91-1.377 0-2.332-1.26-3.428-2.8-1.287-1.82-2.323-4.63-2.323-7.28 0-4.28 2.797-6.55 5.552-6.55 1.448 0 2.675.95 3.6.95.865 0 2.222-1.01 3.902-1.01.613 0 2.886.06 4.374 2.19-.13.09-2.383 1.37-2.383 4.19 0 3.26 2.854 4.42 2.886 4.43z" />
    </svg>
  );
}

function WindowsIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} aria-hidden>
      <path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801" />
    </svg>
  );
}

function platformLabel(platform: string): string {
  const p = platform.toLowerCase();
  if (p === "macos") return "macOS";
  if (p === "windows") return "Windows";
  if (p === "linux") return "Linux";
  return platform;
}

function PlatformIcon({ platform }: { platform: string }) {
  const p = platform.toLowerCase();
  if (p === "macos") return <AppleIcon className="size-4" />;
  if (p === "windows") return <WindowsIcon className="size-4" />;
  return <Terminal className="size-4" />;
}

function StatusBadge({ enabled }: { enabled: boolean }) {
  return (
    <Badge variant="secondary" className="gap-1.5 font-normal">
      <span
        className={cn(
          "size-1.5 rounded-full",
          enabled ? "bg-emerald-500" : "bg-muted-foreground/40",
        )}
      />
      {enabled ? "Active" : "Disabled"}
    </Badge>
  );
}

export const Route = createFileRoute("/_dash/updates/sources")({
  component: SourcesPage,
});

type StorageTarget = { id: string; name: string };

function SourcesPage() {
  const queryClient = useQueryClient();
  const { product, isLoading: productsLoading } = useCurrentProduct();
  const [newConfigOpen, setNewConfigOpen] = useState(false);
  const [editingConfig, setEditingConfig] = useState<ProductUpdateConfigDTO | null>(null);
  const [deletingConfig, setDeletingConfig] = useState<ProductUpdateConfigDTO | null>(null);
  const [storageTarget, setStorageTarget] = useState<StorageTarget | null>(null);

  function invalidateSources() {
    queryClient.invalidateQueries({ queryKey: ["adminProducts"] });
    queryClient.invalidateQueries({ queryKey: ["productUpdateConfigs"] });
  }

  const deleteMutation = useMutation({
    mutationFn: (configId: string) => deleteProductUpdateConfig(configId),
    onSuccess: () => {
      toast.success("Update source deleted");
      setDeletingConfig(null);
      invalidateSources();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete update source"),
  });

  return (
    <AdminShell
      title="Sources"
      actions={
        product ? (
          <Button onClick={() => setNewConfigOpen(true)}>
            <Plus className="size-4" /> Add source
          </Button>
        ) : null
      }
    >
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Sources</h1>
        <p className="text-muted-foreground text-sm">
          Configure update sources per platform and channel — and where this product's release
          artifacts are stored.
        </p>
      </div>

      {productsLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-40 w-full" />
        </div>
      ) : !product ? (
        <Card>
          <CardContent className="text-muted-foreground flex flex-col items-center gap-2 py-12 text-sm">
            <Globe className="size-6 opacity-40" />
            No product selected. Create or pick a product from the sidebar.
          </CardContent>
        </Card>
      ) : (
        <ProductCard
          product={product}
          onAddSource={() => setNewConfigOpen(true)}
          onEditConfig={setEditingConfig}
          onDeleteConfig={setDeletingConfig}
          onEditStorage={() => setStorageTarget({ id: product.id, name: product.name })}
        />
      )}

      <UpdateConfigDialog
        productId={product?.id}
        open={newConfigOpen}
        editingConfig={editingConfig}
        onOpenChange={(open) => {
          if (!open) {
            setNewConfigOpen(false);
            setEditingConfig(null);
          }
        }}
        onSaved={() => {
          setNewConfigOpen(false);
          setEditingConfig(null);
          invalidateSources();
        }}
      />

      <ConfirmDialog
        open={deletingConfig !== null}
        onOpenChange={(open) => {
          if (!open) setDeletingConfig(null);
        }}
        title="Delete update source?"
        description={
          deletingConfig ? (
            <>
              This removes the <strong>{platformLabel(deletingConfig.platform)} · {deletingConfig.channel}</strong>{" "}
              source. Existing releases and artifacts are not deleted.
            </>
          ) : null
        }
        pending={deleteMutation.isPending}
        onConfirm={() => {
          if (deletingConfig) deleteMutation.mutate(deletingConfig.id);
        }}
      />

      <StorageConfigDialog
        productId={storageTarget?.id ?? ""}
        productName={storageTarget?.name ?? ""}
        open={storageTarget !== null}
        onOpenChange={(open) => {
          if (!open) setStorageTarget(null);
        }}
        onSaved={() => {
          if (storageTarget) {
            queryClient.invalidateQueries({ queryKey: ["productStorageConfig", storageTarget.id] });
          }
        }}
      />
    </AdminShell>
  );
}

function StorageBadge({ productId }: { productId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ["productStorageConfig", productId],
    queryFn: () => getProductStorageConfig(productId),
  });

  if (isLoading) return <Skeleton className="h-5 w-28" />;

  const backend = data?.backend ?? "local";
  const isS3 = backend === "s3";
  const Icon = isS3 ? Cloud : HardDrive;
  const label = isS3 ? "S3 / compatible" : "Local filesystem";

  return (
    <span className="text-muted-foreground inline-flex items-center gap-1.5 text-sm">
      <Icon className="size-4" />
      {label}
    </span>
  );
}

function ProductCard({
  product,
  onAddSource,
  onEditConfig,
  onDeleteConfig,
  onEditStorage,
}: {
  product: AdminProductItem;
  onAddSource: (productId: string) => void;
  onEditConfig: (cfg: ProductUpdateConfigDTO) => void;
  onDeleteConfig: (cfg: ProductUpdateConfigDTO) => void;
  onEditStorage: () => void;
}) {
  const { data: configs, isLoading } = useQuery({
    queryKey: ["productUpdateConfigs", product.id],
    queryFn: () => listProductUpdateConfigs(product.id),
  });

  // Hide products without any update sources.
  if (!isLoading && !configs?.length) return null;

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between gap-4">
        <Link
          to="/products/$productId"
          params={{ productId: product.id }}
          className="text-primary hover:underline inline-flex items-center gap-1.5 text-lg font-semibold"
        >
          {product.name}
          <ExternalLink className="size-3.5" />
        </Link>
        <div className="flex items-center gap-3">
          <StorageBadge productId={product.id} />
          <Button variant="outline" size="sm" onClick={onEditStorage}>
            <Pencil className="size-3.5" /> Edit
          </Button>
          <Button variant="outline" size="sm" onClick={() => onAddSource(product.id)}>
            <Plus className="size-4" /> Add source
          </Button>
        </div>
      </div>
      {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
          </div>
        ) : (
          <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Platform</TableHead>
                  <TableHead>Channel</TableHead>
                  <TableHead>Protocol</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden md:table-cell">Feed URL</TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {configs?.map((cfg) => (
                  <TableRow key={cfg.id}>
                    <TableCell>
                      <span className="inline-flex items-center gap-2">
                        <PlatformIcon platform={cfg.platform} />
                        {platformLabel(cfg.platform)}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="text-xs">{cfg.channel}</Badge>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm">
                        {cfg.providerKey === "sparkle" ? "Sparkle" : "Clave Native"}
                      </span>
                    </TableCell>
                    <TableCell>
                      <StatusBadge enabled={cfg.enabled} />
                    </TableCell>
                    <TableCell className="hidden md:table-cell">
                      {cfg.feedUrl ? (
                        <code className="text-muted-foreground bg-muted rounded px-1.5 py-0.5 text-xs truncate block max-w-[320px]">
                          {cfg.feedUrl}
                        </code>
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="size-8"
                          onClick={() => onEditConfig(cfg)}
                        >
                          <Pencil className="size-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-muted-foreground hover:text-destructive size-8"
                          onClick={() => onDeleteConfig(cfg)}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
        )}
    </section>
  );
}
