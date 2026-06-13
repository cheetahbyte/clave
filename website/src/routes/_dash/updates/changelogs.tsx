import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, useEffect } from "react";
import { toast } from "sonner";
import {
  listChangelogs,
  createChangelog,
  updateChangelog,
  deleteChangelog,
  type ChangelogDTO,
} from "@/features/admin/api";
import { useCurrentProduct } from "@/features/admin/product-context";
import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
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
import { Plus, Pencil, Trash2, Globe } from "lucide-react";

export const Route = createFileRoute("/_dash/updates/changelogs")({
  component: ChangelogsPage,
});

function formatDate(s?: string): string {
  if (!s) return "";
  return new Date(s).toLocaleDateString("en-US", { year: "numeric", month: "short", day: "numeric" });
}

function ChangelogsPage() {
  const queryClient = useQueryClient();
  const { product, isLoading } = useCurrentProduct();
  const [editTarget, setEditTarget] = useState<ChangelogDTO | null>(null);
  const [newOpen, setNewOpen] = useState(false);
  const [deleting, setDeleting] = useState<ChangelogDTO | null>(null);

  const { data: changelogs, isLoading: changelogsLoading } = useQuery({
    queryKey: ["productChangelogs", product?.id],
    queryFn: () => listChangelogs(product!.id),
    enabled: !!product,
  });

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ["productChangelogs", product?.id] });
    queryClient.invalidateQueries({ queryKey: ["updateReleases"] });
  }

  const deleteMutation = useMutation({
    mutationFn: (changelogId: string) => deleteChangelog(changelogId),
    onSuccess: () => {
      toast.success("Changelog deleted");
      setDeleting(null);
      invalidate();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete changelog"),
  });

  return (
    <AdminShell title="Changelogs">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Changelogs</h1>
          <p className="text-muted-foreground text-sm">
            Reusable changelog entries that can be attached to releases. Content is
            written in Markdown and rendered for Sparkle and Clave Native clients.
          </p>
        </div>
        {product ? (
          <Button onClick={() => setNewOpen(true)}>
            <Plus className="size-4" /> New changelog
          </Button>
        ) : null}
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
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead className="hidden md:table-cell">Preview</TableHead>
                <TableHead className="hidden md:table-cell">Updated</TableHead>
                <TableHead className="w-20" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {changelogsLoading ? (
                [{ id: "sk1" }, { id: "sk2" }].map((s) => (
                  <TableRow key={s.id}>
                    <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                    <TableCell className="hidden md:table-cell"><Skeleton className="h-4 w-64" /></TableCell>
                    <TableCell className="hidden md:table-cell"><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell />
                  </TableRow>
                ))
              ) : !changelogs?.length ? (
                <TableRow>
                  <TableCell colSpan={4} className="text-muted-foreground h-16 text-center text-xs">
                    No changelogs yet. Create one to attach it to a release.
                  </TableCell>
                </TableRow>
              ) : (
                changelogs.map((cl) => (
                  <TableRow key={cl.id}>
                    <TableCell className="font-medium">{cl.title}</TableCell>
                    <TableCell className="text-muted-foreground hidden md:table-cell text-xs max-w-xs truncate">
                      {cl.body.slice(0, 80)}{cl.body.length > 80 ? "…" : ""}
                    </TableCell>
                    <TableCell className="text-muted-foreground hidden md:table-cell text-xs">
                      {formatDate(cl.updatedAt)}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="size-8"
                          onClick={() => setEditTarget(cl)}
                        >
                          <Pencil className="size-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-muted-foreground hover:text-destructive size-8"
                          onClick={() => setDeleting(cl)}
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
      )}

      <ChangelogDialog
        productId={product?.id ?? ""}
        open={newOpen}
        editing={editTarget}
        onOpenChange={(open) => {
          if (!open) {
            setNewOpen(false);
            setEditTarget(null);
          }
        }}
        onSaved={() => {
          setNewOpen(false);
          setEditTarget(null);
          invalidate();
        }}
      />

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        title="Delete changelog?"
        description={
          deleting ? (
            <>
              This permanently deletes <strong>{deleting.title}</strong>. This
              cannot be undone. The changelog will be detached from any releases.
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

function ChangelogDialog({
  productId,
  open,
  editing,
  onOpenChange,
  onSaved,
}: {
  productId: string;
  open: boolean;
  editing: ChangelogDTO | null;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");

  // Re-seed state whenever the dialog target changes.
  useEffect(() => {
    setTitle(editing?.title ?? "");
    setBody(editing?.body ?? "");
  }, [editing]);

  const mutation = useMutation({
    mutationFn: () => {
      if (editing) {
        return updateChangelog(editing.id, { title: title.trim(), body });
      }
      return createChangelog(productId, { title: title.trim(), body });
    },
    onSuccess: () => {
      toast.success(editing ? "Changelog updated" : "Changelog created");
      onSaved();
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to save changelog"),
  });

  const isOpen = open || editing !== null;

  return (
    <Dialog open={isOpen} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{editing ? "Edit changelog" : "New changelog"}</DialogTitle>
          <DialogDescription>
            {editing
              ? "Releases that use this changelog will see the updated content."
              : "Create a reusable changelog entry. Attach it to a release later."}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="cl-title">Title</Label>
            <Input
              id="cl-title"
              value={title}
              placeholder="Version 1.0"
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="cl-body">Body (Markdown)</Label>
            <Textarea
              id="cl-body"
              value={body}
              rows={10}
              placeholder={"## What's new\n- Added X\n- Fixed Y"}
              onChange={(e) => setBody(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter className="mt-4">
          <Button
            type="button"
            disabled={mutation.isPending || !title.trim()}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? "Saving…" : editing ? "Save" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
