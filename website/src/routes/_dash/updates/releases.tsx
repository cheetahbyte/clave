import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, useRef } from "react";
import { toast } from "sonner";
import { Link } from "@tanstack/react-router";
import {
  listReleases,
  listChannels,
  listChangelogs,
  createRelease,
  publishRelease,
  yankRelease,
  deleteRelease,
  uploadArtifact,
  attachReleaseChangelog,
  type ReleaseDTO,
} from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
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
import { Plus, Upload, Rocket, Ban, Trash2, Pencil } from "lucide-react";

export const Route = createFileRoute("/_dash/updates/releases")({
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
  const { product } = useCurrentProduct();
  const [newReleaseOpen, setNewReleaseOpen] = useState(false);
  const [uploadReleaseId, setUploadReleaseId] = useState<string | null>(null);
  const [editRelease, setEditRelease] = useState<ReleaseDTO | null>(null);

  const { data: releases, isLoading: releasesLoading } = useQuery({
    queryKey: ["updateReleases", product?.id],
    queryFn: () => listReleases({ limit: 50, productId: product?.id }),
    enabled: !!product,
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
    <AdminShell title="Releases">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Releases</h1>
          <p className="text-muted-foreground text-sm">
            Manage releases for {product ? product.name : "the selected product"}.
          </p>
        </div>
        {product ? (
          <Button onClick={() => setNewReleaseOpen(true)}>
            <Plus className="size-4" /> New release
          </Button>
        ) : null}
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Channel</TableHead>
              <TableHead>Platform</TableHead>
              <TableHead>Version</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="hidden md:table-cell">Changelog</TableHead>
              <TableHead className="hidden md:table-cell">Artifacts</TableHead>
              <TableHead className="w-40" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {releasesLoading ? (
              [{ id: "sk1" }, { id: "sk2" }].map((s) => (
                <TableRow key={s.id}>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                  <TableCell className="hidden md:table-cell"><Skeleton className="h-4 w-20" /></TableCell>
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
                    {r.changelogId ? (
                      <Badge variant="secondary" className="text-xs">Attached</Badge>
                    ) : (
                      <span className="text-muted-foreground text-xs">—</span>
                    )}
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
                        <Button
                          variant="ghost"
                          size="sm"
                          className="size-8"
                          title="Edit release"
                          onClick={() => setEditRelease(r)}
                        >
                          <Pencil className="size-4" />
                        </Button>
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
        productId={product?.id ?? ""}
        onOpenChange={setNewReleaseOpen}
        onSaved={invalidate}
      />

      <UploadArtifactDialog
        releaseId={uploadReleaseId}
        open={uploadReleaseId !== null}
        onOpenChange={(open) => !open && setUploadReleaseId(null)}
        onSaved={invalidate}
      />

      <EditReleaseDialog
        release={editRelease}
        productId={product?.id ?? ""}
        open={editRelease !== null}
        onOpenChange={(open) => !open && setEditRelease(null)}
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
  const [metadata, setMetadata] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const mutation = useMutation({
    mutationFn: () => {
      const file = fileRef.current?.files?.[0];
      if (!file || !releaseId) throw new Error("No file selected");
      return uploadArtifact(releaseId, file, artifactType, os, arch, metadata || undefined);
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
                  <SelectItem value="delta">Delta</SelectItem>
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
          {artifactType === "delta" && (
            <div className="space-y-2">
              <Label htmlFor="artifact-metadata">Metadata (JSON)</Label>
              <textarea
                id="artifact-metadata"
                className="flex min-h-20 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                placeholder={`{\n  "fromVersion": "1.0.0",\n  "fromBuild": "42",\n  "fromManifestSha256": "...",\n  "fromArtifactSha256": "...",\n  "toVersion": "1.1.0",\n  "toBuild": "43",\n  "toManifestSha256": "...",\n  "toArtifactSha256": "...",\n  "format": "clave.delta/v1"\n}`}
                value={metadata}
                onChange={(e) => setMetadata(e.target.value)}
              />
            </div>
          )}
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

function EditReleaseDialog({
  release,
  productId,
  open,
  onOpenChange,
  onSaved,
}: {
  release: ReleaseDTO | null;
  productId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const [changelogId, setChangelogId] = useState("__none__");

  // Reset the selection whenever a different release is opened.
  const lastReleaseId = useRef<string | null>(null);
  if (release && release.id !== lastReleaseId.current) {
    lastReleaseId.current = release.id;
    setChangelogId(release.changelogId ?? "__none__");
  }

  const { data: changelogs } = useQuery({
    queryKey: ["productChangelogs", productId],
    queryFn: () => listChangelogs(productId),
    enabled: open && !!productId,
  });

  const mutation = useMutation({
    mutationFn: () => {
      if (!release) throw new Error("No release selected");
      return attachReleaseChangelog(
        release.id,
        changelogId === "__none__" ? null : changelogId,
      );
    },
    onSuccess: () => {
      toast.success("Release updated");
      onOpenChange(false);
      onSaved();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to update release"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Edit release</DialogTitle>
          <DialogDescription>
            {release
              ? `${release.version}${release.buildNumber ? ` (${release.buildNumber})` : ""} · ${formatPlatform(release.platform)} · ${release.channel}`
              : ""}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Changelog</Label>
            <Select value={changelogId} onValueChange={setChangelogId}>
              <SelectTrigger><SelectValue placeholder="No changelog" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">None</SelectItem>
                {changelogs?.map((cl) => (
                  <SelectItem key={cl.id} value={cl.id}>{cl.title}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-muted-foreground text-xs">
              <Link to="/updates/changelogs" className="underline hover:text-foreground">
                Manage changelogs
              </Link>
            </p>
          </div>
        </div>
        <DialogFooter className="mt-4">
          <Button
            type="button"
            disabled={mutation.isPending || !release}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Saving…" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function NewReleaseDialog({
  open,
  productId,
  onOpenChange,
  onSaved,
}: {
  open: boolean;
  productId: string;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const [platform, setPlatform] = useState("macos");
  const [channel, setChannel] = useState("stable");
  const [version, setVersion] = useState("");
  const [buildNumber, setBuildNumber] = useState("");
  const [releaseNotes, setReleaseNotes] = useState("");
  const [changelogId, setChangelogId] = useState("__none__");

  const { data: channels } = useQuery({
    queryKey: ["productChannels", productId],
    queryFn: () => listChannels(productId),
    enabled: open && !!productId,
  });

  const { data: changelogs } = useQuery({
    queryKey: ["productChangelogs", productId],
    queryFn: () => listChangelogs(productId),
    enabled: open && !!productId,
  });

  const mutation = useMutation({
    mutationFn: () =>
      createRelease({
        productId,
        platform,
        channel,
        version: version.trim(),
        buildNumber: buildNumber.trim() || undefined,
        releaseNotes: releaseNotes.trim() || undefined,
        changelogId: changelogId === "__none__" ? undefined : changelogId,
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
                <SelectTrigger><SelectValue placeholder="Select a channel" /></SelectTrigger>
                <SelectContent>
                  {(channels?.length
                    ? channels.map((c) => c.name)
                    : ["stable", "beta", "nightly"]
                  ).map((name) => (
                    <SelectItem key={name} value={name} className="capitalize">{name}</SelectItem>
                  ))}
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
            <p className="text-muted-foreground text-xs">Short one-line summary.</p>
          </div>
          <div className="space-y-2">
            <Label>Changelog</Label>
            <Select value={changelogId} onValueChange={setChangelogId}>
              <SelectTrigger><SelectValue placeholder="No changelog" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="__none__">None</SelectItem>
                {changelogs?.map((cl) => (
                  <SelectItem key={cl.id} value={cl.id}>{cl.title}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-muted-foreground text-xs">
              <Link to="/updates/changelogs" className="underline hover:text-foreground">
                Manage changelogs
              </Link>
            </p>
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
