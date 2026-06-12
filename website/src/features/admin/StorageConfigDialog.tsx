import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import {
  getProductStorageConfig,
  saveProductStorageConfig,
} from "@/features/admin/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
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

export function StorageConfigDialog({
  productId,
  productName,
  open,
  onOpenChange,
  onSaved,
}: {
  productId: string;
  productName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const { data, isLoading } = useQuery({
    queryKey: ["productStorageConfig", productId],
    queryFn: () => getProductStorageConfig(productId),
    enabled: open,
  });

  const [backend, setBackend] = useState("local");
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [hydrated, setHydrated] = useState(false);

  if (open && data && !hydrated) {
    setBackend(data.backend);
    setConfig(data.config ?? {});
    setHydrated(true);
  }
  if (!open && hydrated) {
    setHydrated(false);
  }

  function field(key: string): string {
    const v = config[key];
    return typeof v === "string" ? v : "";
  }
  function setField(key: string, value: string) {
    setConfig((c) => ({ ...c, [key]: value }));
  }

  const mutation = useMutation({
    mutationFn: () => saveProductStorageConfig(productId, { backend, config }),
    onSuccess: () => {
      toast.success("Storage backend saved");
      onSaved();
      onOpenChange(false);
    },
    onError: (e) =>
      toast.error(e instanceof Error ? e.message : "Failed to save storage config"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Artifact storage</DialogTitle>
          <DialogDescription>
            Where uploaded release artifacts for <strong>{productName}</strong> are stored.
            Existing artifacts stay on their original backend.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <Skeleton className="h-40 w-full" />
        ) : (
          <form
            className="space-y-4"
            onSubmit={(e) => {
              e.preventDefault();
              mutation.mutate();
            }}
          >
            <div className="space-y-2">
              <Label>Backend</Label>
              <Select value={backend} onValueChange={setBackend}>
                <SelectTrigger className="w-full"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="local">Local filesystem</SelectItem>
                  <SelectItem value="s3">S3 / S3-compatible</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {backend === "local" && (
              <div className="space-y-2">
                <Label>Base path</Label>
                <Input
                  placeholder="Leave empty for server default"
                  value={field("base_path")}
                  onChange={(e) => setField("base_path", e.target.value)}
                />
              </div>
            )}

            {backend === "s3" && (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Bucket</Label>
                    <Input value={field("bucket")} onChange={(e) => setField("bucket", e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Region</Label>
                    <Input placeholder="us-east-1" value={field("region")} onChange={(e) => setField("region", e.target.value)} />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>Endpoint (optional)</Label>
                  <Input
                    placeholder="https://… for MinIO, R2, B2; empty for AWS"
                    value={field("endpoint")}
                    onChange={(e) => setField("endpoint", e.target.value)}
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>Access key ID</Label>
                    <Input value={field("access_key_id")} onChange={(e) => setField("access_key_id", e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label>Secret access key</Label>
                    <Input
                      type="password"
                      placeholder="Leave unchanged to keep current"
                      value={field("secret_access_key")}
                      onChange={(e) => setField("secret_access_key", e.target.value)}
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>Key prefix (optional)</Label>
                  <Input value={field("prefix")} onChange={(e) => setField("prefix", e.target.value)} />
                </div>
                <div className="flex items-center gap-3">
                  <Switch
                    checked={config["use_path_style"] === true}
                    onCheckedChange={(v) => setConfig((c) => ({ ...c, use_path_style: v }))}
                  />
                  <Label>Use path-style addressing (required for MinIO)</Label>
                </div>
              </div>
            )}

            <DialogFooter className="mt-2">
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? "Saving…" : "Save"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
