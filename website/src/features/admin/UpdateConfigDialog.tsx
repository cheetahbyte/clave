import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  listAdminProducts,
  saveProductUpdateConfig,
  type ProductUpdateConfigDTO,
} from "@/features/admin/api";
import { Button } from "@/components/ui/button";
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export function UpdateConfigDialog({
  productId,
  open,
  editingConfig,
  onOpenChange,
  onSaved,
}: {
  productId?: string;
  open: boolean;
  editingConfig: ProductUpdateConfigDTO | null;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const isEdit = editingConfig !== null;
  const isOpen = open || isEdit;

  const { data: products } = useQuery({
    queryKey: ["adminProducts"],
    queryFn: listAdminProducts,
    enabled: isOpen,
  });

  const [selectedProductId, setSelectedProductId] = useState(
    editingConfig?.productId ?? productId ?? "",
  );
  const [platform, setPlatform] = useState(editingConfig?.platform ?? "macos");
  const [channel, setChannel] = useState(editingConfig?.channel ?? "stable");
  const [providerKey, setProviderKey] = useState(editingConfig?.providerKey ?? "sparkle");
  const [enabled, setEnabled] = useState(editingConfig?.enabled ?? true);

  const effectiveProductId = productId ?? selectedProductId;

  const mutation = useMutation({
    mutationFn: () =>
      saveProductUpdateConfig(effectiveProductId, {
        platform,
        channel,
        providerKey,
        enabled,
        config: {},
      }),
    onSuccess: () => {
      toast.success(isEdit ? "Update source saved" : "Update source added");
      onSaved();
    },
    onError: (e) =>
      toast.error(e instanceof Error ? e.message : "Failed to save update config"),
  });

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit update source" : "Add update source"}</DialogTitle>
          <DialogDescription>
            Configure how clients on this platform and channel receive updates.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          {!productId && (
            <div className="space-y-2">
              <Label>Product</Label>
              <Select
                value={selectedProductId}
                onValueChange={setSelectedProductId}
                disabled={isEdit}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select a product" />
                </SelectTrigger>
                <SelectContent>
                  {(products ?? []).map((p) => (
                    <SelectItem key={p.id} value={p.id}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Platform</Label>
              <Select value={platform} onValueChange={setPlatform}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="macos">macOS</SelectItem>
                  <SelectItem value="windows">Windows</SelectItem>
                  <SelectItem value="linux">Linux</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Channel</Label>
              <Select value={channel} onValueChange={setChannel}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="stable">Stable</SelectItem>
                  <SelectItem value="beta">Beta</SelectItem>
                  <SelectItem value="nightly">Nightly</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="space-y-2">
            <Label>Delivery protocol</Label>
            <Select value={providerKey} onValueChange={setProviderKey}>
              <SelectTrigger><SelectValue placeholder="Select a protocol" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="sparkle">Sparkle (Appcast XML)</SelectItem>
                <SelectItem value="clave_native">Clave Native (JSON API)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center justify-between rounded-lg border p-3">
            <div className="space-y-0.5">
              <Label className="text-sm">Enabled</Label>
              <p className="text-muted-foreground text-xs">
                Disable to pause update checks for this config.
              </p>
            </div>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
          {providerKey === "sparkle" && (
            <div className="rounded-lg bg-muted/50 p-3 space-y-1">
              <p className="text-xs font-medium">Sparkle appcast</p>
              <p className="text-xs text-muted-foreground">
                The appcast XML is generated automatically by Clave. Point your Sparkle
                client to the URL shown on the Updates page.
              </p>
            </div>
          )}
          {providerKey === "clave_native" && (
            <div className="rounded-lg bg-muted/50 p-3 space-y-1">
              <p className="text-xs font-medium">Clave Native</p>
              <p className="text-xs text-muted-foreground">
                Clients poll the Clave update API directly with their license token. Staged
                rollouts, mandatory updates, and artifact selection are handled server-side.
              </p>
            </div>
          )}
        </div>
        <DialogFooter className="mt-4">
          <Button
            type="button"
            disabled={
              mutation.isPending ||
              !providerKey ||
              !platform ||
              !channel ||
              !effectiveProductId
            }
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Saving…" : isEdit ? "Save" : "Add source"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
