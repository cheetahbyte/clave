import { createFileRoute, redirect } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, useRef } from "react";
import { toast } from "sonner";
import {
  getCurrentAdmin,
  listAdminProducts,
  listReleases,
  createRelease,
  publishRelease,
  yankRelease,
  deleteRelease,
  uploadArtifact,
  type AdminProductItem,
} from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { Plus, Upload, Rocket, Ban, Trash2 } from "lucide-react";

export const Route = createFileRoute("/_dash/updates/releases")({
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
  component: ReleasesPage,
});

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function formatPlatform(platform: string): string {
  switch (platform) {
    case "macos": return "macOS";
    case "windows": return "Windows";
    case "linux": return "Linux";
    default: return platform;
  }
}

function statusLabel(status: string): string {
  if (!status) return "Draft";
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function ReleasesPage() {
  const queryClient = useQueryClient();
  const [newReleaseOpen, setNewReleaseOpen] = useState(false);
  const [uploadReleaseId, setUploadReleaseId] = useState<string | null>(null);

  const { data: products } = useQuery({
    queryKey: ["adminProducts"],
    queryFn: listAdminProducts,
  });

  const { data: releases, isLoading: releasesLoading } = useQuery({
    queryKey: ["updateReleases"],
    queryFn: () => listReleases({ limit: 50 }),
  });

  const publishMut = useMutation({
    mutationFn: publishRelease,
    onSuccess: () => {
      toast.success("Release published");
      queryClient.invalidateQueries({ queryKey: ["updateReleases"] });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Publish failed"),
  });

  const yankMut = useMutation({
    mutationFn: yankRelease,
    onSuccess: () => {
      toast.success("Release yanked");
      queryClient.invalidateQueries({ queryKey: ["updateReleases"] });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Yank failed"),
  });

  const deleteMut = useMutation({
    mutationFn: deleteRelease,
    onSuccess: () => {
      toast.success("Release deleted");
      queryClient.invalidateQueries({ queryKey: ["updateReleases"] });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Delete failed"),
  });

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["updateReleases"] });
  }

  return (
    <AdminShell
      title="Releases"
      actions={
        <Button onClick={() => setNewReleaseOpen(true)}>
          <Plus className="size-4" /> New release
        </Button>
      }
    >
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Releases</h1>
        <p className="text-muted-foreground text-sm">
          Manage releases across your products.
        </p>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Product</TableHead>
              <TableHead>Channel</TableHead>
              <TableHead>Platform</TableHead>
              <TableHead>Version</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="hidden md:table-cell">Artifacts</TableHead>
              <TableHead className="w-40" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {releasesLoading ? (
              [{ id: "sk1" }, { id: "sk2" }].map((s) => (
                <TableRow key={s.id}>
                  <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell className="hidden md:table-cell"><Skeleton className="h-4 w-12" /></TableCell>
                  <TableCell />
                </TableRow>
              ))
            ) : !releases?.length ? (
              <TableRow>
                <TableCell colSpan={7} className="text-muted-foreground h-16 text-center text-xs">
                  No releases yet. Create one to start delivering updates.
                </TableCell>
              </TableRow>
            ) : (
              releases.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-medium">{r.productName}</TableCell>
                  <TableCell className="text-muted-foreground">
                    <Badge variant="secondary" className="text-xs">{r.channel}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{formatPlatform(r.platform)}</TableCell>
                  <TableCell>
                    <span className="text-sm">{r.version}</span>
                    {r.buildNumber && (
                      <span className="text-muted-foreground text-xs ml-1">({r.buildNumber})</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        r.status === "published" ? "default" :
                        r.status === "yanked" ? "destructive" : "outline"
                      }
                      className="text-xs"
                    >
                      {statusLabel(r.status)}
                    </Badge>
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    {r.artifacts?.length ? (
                      <div className="space-y-1">
                        {r.artifacts.map((a, i) => (
                          <div key={i} className="text-xs text-muted-foreground">
                            <span className="font-mono">{a.filename || a.type}</span>
                            {a.sizeBytes != null && (
                              <span className="ml-1">({formatBytes(a.sizeBytes)})</span>
                            )}
                          </div>
                        ))}
                      </div>
                    ) : (
                      <span className="text-muted-foreground text-xs">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                      <div className="flex items-center gap-1">
                        {(r.status === "draft" || !r.status) && (
                          <>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="size-8"
                              onClick={() => setUploadReleaseId(r.id)}
                            >
                              <Upload className="size-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              className="size-8"
                              disabled={!r.artifacts?.length}
                              onClick={() => publishMut.mutate(r.id)}
                            >
                              <Rocket className="size-4" />
                            </Button>
                          </>
                        )}
                      {r.status === "published" && (
                        <Button
                          variant="ghost"
                          size="sm"
                          className="size-8 text-destructive"
                          onClick={() => yankMut.mutate(r.id)}
                        >
                          <Ban className="size-4" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        className="size-8 text-destructive"
                        onClick={() => deleteMut.mutate(r.id)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <NewReleaseDialog
        open={newReleaseOpen}
        products={products ?? []}
        onOpenChange={setNewReleaseOpen}
        onSaved={invalidate}
      />

      <UploadArtifactDialog
        releaseId={uploadReleaseId}
        open={uploadReleaseId !== null}
        onOpenChange={(open) => !open && setUploadReleaseId(null)}
        onSaved={invalidate}
      />
    </AdminShell>
  );
}

function UploadArtifactDialog({
  releaseId,
  open,
  onOpenChange,
  onSaved,
}: {
  releaseId: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const [artifactType, setArtifactType] = useState("dmg");
  const [os, setOs] = useState("macos");
  const [arch, setArch] = useState("universal");
  const fileRef = useRef<HTMLInputElement>(null);

  const mutation = useMutation({
    mutationFn: () => {
      const file = fileRef.current?.files?.[0];
      if (!file || !releaseId) throw new Error("No file selected");
      return uploadArtifact(releaseId, file, artifactType, os, arch);
    },
    onSuccess: () => {
      toast.success("Artifact uploaded");
      onOpenChange(false);
      onSaved();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Upload failed"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Upload artifact</DialogTitle>
          <DialogDescription>
            Upload a binary for this release.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="artifact-file">File</Label>
            <Input id="artifact-file" type="file" ref={fileRef} />
          </div>
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label>Type</Label>
              <Select value={artifactType} onValueChange={setArtifactType}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="dmg">DMG</SelectItem>
                  <SelectItem value="zip">ZIP</SelectItem>
                  <SelectItem value="pkg">PKG</SelectItem>
                  <SelectItem value="exe">EXE</SelectItem>
                  <SelectItem value="msi">MSI</SelectItem>
                  <SelectItem value="appimage">AppImage</SelectItem>
                  <SelectItem value="deb">DEB</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>OS</Label>
              <Select value={os} onValueChange={setOs}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="macos">macOS</SelectItem>
                  <SelectItem value="windows">Windows</SelectItem>
                  <SelectItem value="linux">Linux</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Arch</Label>
              <Select value={arch} onValueChange={setArch}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="universal">Universal</SelectItem>
                  <SelectItem value="arm64">ARM64</SelectItem>
                  <SelectItem value="x64">x64</SelectItem>
                  <SelectItem value="x86">x86</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </div>
        <DialogFooter className="mt-4">
          <Button
            type="button"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Uploading…" : "Upload"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function NewReleaseDialog({
  open,
  products,
  onOpenChange,
  onSaved,
}: {
  open: boolean;
  products: AdminProductItem[];
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const [productId, setProductId] = useState("");
  const [platform, setPlatform] = useState("macos");
  const [channel, setChannel] = useState("stable");
  const [version, setVersion] = useState("");
  const [buildNumber, setBuildNumber] = useState("");
  const [releaseNotes, setReleaseNotes] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      createRelease({
        productId,
        platform,
        channel,
        version: version.trim(),
        buildNumber: buildNumber.trim() || undefined,
        releaseNotes: releaseNotes.trim() || undefined,
      }),
    onSuccess: () => {
      toast.success("Release draft created");
      onOpenChange(false);
      onSaved();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to create release"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>New release</DialogTitle>
          <DialogDescription>
            Create a draft release. Upload artifacts and publish it after.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Product</Label>
            <Select value={productId} onValueChange={setProductId}>
              <SelectTrigger><SelectValue placeholder="Select a product" /></SelectTrigger>
              <SelectContent>
                {products.map((p) => (
                  <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
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
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="release-version">Version</Label>
              <Input id="release-version" value={version} placeholder="1.0.0" onChange={(e) => setVersion(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="release-build">Build number</Label>
              <Input id="release-build" value={buildNumber} placeholder="42" onChange={(e) => setBuildNumber(e.target.value)} />
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="release-notes">Release notes</Label>
            <Input id="release-notes" value={releaseNotes} placeholder="Bug fixes and improvements" onChange={(e) => setReleaseNotes(e.target.value)} />
          </div>
        </div>
        <DialogFooter className="mt-4">
          <Button
            type="button"
            disabled={mutation.isPending || !productId || !version.trim()}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Creating…" : "Create draft"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
