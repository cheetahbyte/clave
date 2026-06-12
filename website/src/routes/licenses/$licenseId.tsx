import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { getCurrentAdmin } from "@/features/admin/api";
import { getAdminLicense, updateLicense, deleteLicense, type AdminLicenseDetail } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
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
import { Pencil, Trash2 } from "lucide-react";

export const Route = createFileRoute("/licenses/$licenseId")({
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
  component: LicenseDetailPage,
});

function LicenseDetailPage() {
  const { licenseId } = Route.useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const { data: license, isLoading } = useQuery({
    queryKey: ["adminLicense", licenseId],
    queryFn: () => getAdminLicense(licenseId),
  });

  const deleteMut = useMutation({
    mutationFn: () => deleteLicense(licenseId),
    onSuccess: () => {
      toast.success("License deleted");
      queryClient.invalidateQueries({ queryKey: ["adminLicenses"] });
      queryClient.invalidateQueries({ queryKey: ["adminOverview"] });
      navigate({ to: "/licenses" });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to delete license"),
  });

  return (
    <AdminShell
      title="Details"
      breadcrumbs={[{ label: "Licenses", to: "/licenses" }]}
      actions={
        license ? (
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
      {license ? (
        <EditLicenseDialog
          key={`${license.id}-${editOpen}`}
          license={license}
          open={editOpen}
          onOpenChange={setEditOpen}
        />
      ) : null}
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete license?"
        description="This permanently deletes the license and all its activations. This can't be undone."
        pending={deleteMut.isPending}
        onConfirm={() => deleteMut.mutate()}
      />
      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2">
          <Skeleton className="h-48" />
          <Skeleton className="h-48" />
        </div>
      ) : !license ? (
        <p className="text-muted-foreground">License not found</p>
      ) : (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">License Info</CardTitle>
              </CardHeader>
              <CardContent className="space-y-1 text-sm">
                <p><span className="text-muted-foreground">Product:</span> {license.productName}</p>
                <p><span className="text-muted-foreground">Customer:</span> {license.customerEmail}</p>
                <p><span className="text-muted-foreground">Status:</span>{" "}
                  <Badge variant={license.isActive ? "default" : "secondary"}>
                    {license.isActive ? "Active" : "Inactive"}
                  </Badge>
                  {license.isTrial && <Badge variant="outline" className="ml-1.5">Trial</Badge>}
                </p>
                <p><span className="text-muted-foreground">Activations:</span> {license.activationCount}/{license.maxActivations}</p>
                <p><span className="text-muted-foreground">Created:</span> {license.createdAt ? new Date(license.createdAt).toLocaleString() : "—"}</p>
                <p><span className="text-muted-foreground">Expires:</span> {license.expiresAt ? new Date(license.expiresAt).toLocaleString() : "—"}</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm font-medium">Features</CardTitle>
              </CardHeader>
              <CardContent>
                {license.features?.length ? (
                  <div className="flex flex-wrap gap-1">
                    {license.features.map((f) => (
                      <Badge key={f} variant="outline">{f}</Badge>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">No features</p>
                )}
              </CardContent>
            </Card>
          </div>

          <div className="space-y-3">
            <h3 className="text-sm font-medium">Activations</h3>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Device ID</TableHead>
                    <TableHead>Hostname</TableHead>
                    <TableHead>Checked In</TableHead>
                    <TableHead className="text-right">Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {!license.activations.length ? (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-muted-foreground">No activations</TableCell>
                    </TableRow>
                  ) : (
                    license.activations.map((a) => (
                      <TableRow key={a.id}>
                        <TableCell className="font-mono text-xs">{a.deviceId}</TableCell>
                        <TableCell>{a.hostname ?? "—"}</TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {a.checkedInAt ? new Date(a.checkedInAt).toLocaleString() : "Never"}
                        </TableCell>
                        <TableCell className="text-right text-muted-foreground text-sm">
                          {a.createdAt ? new Date(a.createdAt).toLocaleDateString() : "—"}
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </div>
        </div>
      )}
    </AdminShell>
  );
}

function toDateInput(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toISOString().slice(0, 10);
}

function EditLicenseDialog({
  license,
  open,
  onOpenChange,
}: {
  license: AdminLicenseDetail;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [email, setEmail] = useState(license.customerEmail);
  const [maxActs, setMaxActs] = useState(String(license.maxActivations));
  const [active, setActive] = useState(license.isActive ? "active" : "inactive");
  const [expiresAt, setExpiresAt] = useState(toDateInput(license.expiresAt));
  const [features, setFeatures] = useState(license.features.join(", "));

  const mutation = useMutation({
    mutationFn: () =>
      updateLicense(license.id, {
        customerEmail: email.trim(),
        maxActivations: parseInt(maxActs, 10) || 1,
        isActive: active === "active",
        expiresAt: expiresAt ? new Date(expiresAt).toISOString() : null,
        features: features
          .split(",")
          .map((f) => f.trim())
          .filter(Boolean),
      }),
    onSuccess: () => {
      toast.success("License updated");
      queryClient.invalidateQueries({ queryKey: ["adminLicense", license.id] });
      queryClient.invalidateQueries({ queryKey: ["adminLicenses"] });
      queryClient.invalidateQueries({ queryKey: ["adminOverview"] });
      onOpenChange(false);
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to update license"),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Edit license</DialogTitle>
          <DialogDescription>Update license details for {license.productName}.</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            if (email.trim()) mutation.mutate();
          }}
        >
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-email">Customer email</Label>
              <Input id="edit-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="edit-max">Max activations</Label>
                <Input
                  id="edit-max"
                  type="number"
                  min={1}
                  value={maxActs}
                  onChange={(e) => setMaxActs(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-status">Status</Label>
                <Select value={active} onValueChange={setActive}>
                  <SelectTrigger id="edit-status" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="inactive">Inactive</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-expires">Expires</Label>
              <Input
                id="edit-expires"
                type="date"
                value={expiresAt}
                onChange={(e) => setExpiresAt(e.target.value)}
              />
              <p className="text-muted-foreground text-xs">Leave empty for no expiry.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-features">Features</Label>
              <Input
                id="edit-features"
                value={features}
                placeholder="pro, beta, sso"
                onChange={(e) => setFeatures(e.target.value)}
              />
              <p className="text-muted-foreground text-xs">Comma-separated.</p>
            </div>
          </div>
          <DialogFooter className="mt-4">
            <Button type="submit" disabled={mutation.isPending || !email.trim()}>
              {mutation.isPending ? "Saving…" : "Save changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
