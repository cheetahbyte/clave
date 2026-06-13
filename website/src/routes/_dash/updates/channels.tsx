import { createFileRoute, redirect, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import {
  getCurrentAdmin,
  listChannels,
  createChannel,
  updateChannel,
  deleteChannel,
  type AdminProductItem,
  type ChannelDTO,
} from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
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
import { ExternalLink, Plus, Pencil, Trash2, Globe, Lock } from "lucide-react";

export const Route = createFileRoute("/_dash/updates/channels")({
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
  component: ChannelsPage,
});

type EditTarget = { productId: string; channel: ChannelDTO | null };

function ChannelsPage() {
  const queryClient = useQueryClient();
  const { product, isLoading } = useCurrentProduct();
  const [editTarget, setEditTarget] = useState<EditTarget | null>(null);
  const [deleting, setDeleting] = useState<ChannelDTO | null>(null);

  const deleteMutation = useMutation({
    mutationFn: (channelId: string) => deleteChannel(channelId),
    onSuccess: () => {
      toast.success("Channel deleted");
      setDeleting(null);
      queryClient.invalidateQueries({ queryKey: ["productChannels"] });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete channel"),
  });

  return (
    <AdminShell title="Channels">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Channels</h1>
        <p className="text-muted-foreground text-sm">
          Release tracks per product. Add required features to gate a channel — only licenses
          holding all of them can pull updates from it.
        </p>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          <Skeleton className="h-32 w-full" />
        </div>
      ) : !product ? (
        <div className="text-muted-foreground flex flex-col items-center gap-2 py-12 text-sm">
          <Globe className="size-6 opacity-40" />
          No product selected. Create or pick a product from the sidebar.
        </div>
      ) : (
        <ProductChannels
          product={product}
          onNew={() => setEditTarget({ productId: product.id, channel: null })}
          onEdit={(channel) => setEditTarget({ productId: product.id, channel })}
          onDelete={setDeleting}
        />
      )}

      <ChannelDialog
        target={editTarget}
        onOpenChange={(open) => {
          if (!open) setEditTarget(null);
        }}
        onSaved={() => {
          setEditTarget(null);
          queryClient.invalidateQueries({ queryKey: ["productChannels"] });
        }}
      />

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        title="Delete channel?"
        description={
          deleting ? (
            <>
              This deletes the <strong>{deleting.name}</strong> channel. Channels still referenced
              by releases or sources can't be deleted — remove those first.
            </>
          ) : null
        }
        pending={deleteMutation.isPending}
        onConfirm={() => {
          if (deleting) deleteMutation.mutate(deleting.id);
        }}
      />
    </AdminShell>
  );
}

function ProductChannels({
  product,
  onNew,
  onEdit,
  onDelete,
}: {
  product: AdminProductItem;
  onNew: () => void;
  onEdit: (channel: ChannelDTO) => void;
  onDelete: (channel: ChannelDTO) => void;
}) {
  const { data: channels, isLoading } = useQuery({
    queryKey: ["productChannels", product.id],
    queryFn: () => listChannels(product.id),
  });

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
        <Button variant="outline" size="sm" onClick={onNew}>
          <Plus className="size-4" /> New channel
        </Button>
      </div>

      {isLoading ? (
        <Skeleton className="h-20 w-full" />
      ) : !channels?.length ? (
        <p className="text-muted-foreground text-sm">No channels yet.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Channel</TableHead>
              <TableHead>Required features</TableHead>
              <TableHead className="w-20" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {channels.map((ch) => (
              <TableRow key={ch.id}>
                <TableCell>
                  <span className="inline-flex items-center gap-2 font-medium">
                    {ch.name}
                    {ch.isDefault && (
                      <Badge variant="secondary" className="text-xs font-normal">default</Badge>
                    )}
                  </span>
                </TableCell>
                <TableCell>
                  {ch.requiredFeatures.length === 0 ? (
                    <span className="text-muted-foreground text-sm">Public</span>
                  ) : (
                    <span className="inline-flex flex-wrap items-center gap-1">
                      <Lock className="text-muted-foreground size-3.5" />
                      {ch.requiredFeatures.map((f) => (
                        <Badge key={f} variant="outline" className="font-mono text-xs">{f}</Badge>
                      ))}
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex items-center justify-end gap-1">
                    <Button variant="ghost" size="sm" className="size-8" onClick={() => onEdit(ch)}>
                      <Pencil className="size-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-muted-foreground hover:text-destructive size-8"
                      onClick={() => onDelete(ch)}
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

function ChannelDialog({
  target,
  onOpenChange,
  onSaved,
}: {
  target: EditTarget | null;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const open = target !== null;
  const editing = target?.channel ?? null;

  const [name, setName] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [features, setFeatures] = useState<string[]>([]);
  const [featureInput, setFeatureInput] = useState("");
  const [hydratedFor, setHydratedFor] = useState<string | null>(null);

  // Re-seed state whenever the dialog target changes.
  const targetKey = target ? `${target.productId}:${editing?.id ?? "new"}` : null;
  if (open && hydratedFor !== targetKey) {
    setName(editing?.name ?? "");
    setIsDefault(editing?.isDefault ?? false);
    setFeatures(editing?.requiredFeatures ?? []);
    setFeatureInput("");
    setHydratedFor(targetKey);
  }
  if (!open && hydratedFor !== null) setHydratedFor(null);

  function addFeature() {
    const v = featureInput.trim();
    if (v && !features.includes(v)) setFeatures((f) => [...f, v]);
    setFeatureInput("");
  }

  const mutation = useMutation({
    mutationFn: () => {
      const payload = { name: name.trim(), isDefault, requiredFeatures: features };
      if (editing) return updateChannel(editing.id, payload);
      return createChannel(target!.productId, payload);
    },
    onSuccess: () => {
      toast.success(editing ? "Channel saved" : "Channel created");
      onSaved();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to save channel"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{editing ? "Edit channel" : "New channel"}</DialogTitle>
          <DialogDescription>
            Channel names are lowercased. Required features gate access (ALL must match).
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Name</Label>
            <Input
              value={name}
              autoFocus
              placeholder="insiders"
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label>Required features</Label>
            <div className="flex gap-2">
              <Input
                value={featureInput}
                placeholder="feature key, then Enter"
                onChange={(e) => setFeatureInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === ",") {
                    e.preventDefault();
                    addFeature();
                  }
                }}
              />
              <Button type="button" variant="outline" onClick={addFeature}>Add</Button>
            </div>
            {features.length > 0 && (
              <div className="flex flex-wrap gap-1.5 pt-1">
                {features.map((f) => (
                  <Badge key={f} variant="secondary" className="gap-1 font-mono text-xs">
                    {f}
                    <button
                      type="button"
                      className="hover:text-destructive"
                      onClick={() => setFeatures((arr) => arr.filter((x) => x !== f))}
                    >
                      <Trash2 className="size-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            )}
            <p className="text-muted-foreground text-xs">
              Leave empty for a public channel. Sparkle appcasts stay public regardless.
            </p>
          </div>

          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="space-y-0.5">
              <Label className="text-sm">Default channel</Label>
              <p className="text-muted-foreground text-xs">Used when a client omits the channel.</p>
            </div>
            <Switch checked={isDefault} onCheckedChange={setIsDefault} />
          </div>
        </div>
        <DialogFooter className="mt-2">
          <Button
            type="button"
            disabled={mutation.isPending || !name.trim()}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Saving…" : editing ? "Save" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
