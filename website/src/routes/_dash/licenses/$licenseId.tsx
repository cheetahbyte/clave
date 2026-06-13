import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";
import { getAdminLicense, updateLicense, deleteLicense, type AdminLicenseDetail } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Button } from "@/components/ui/button";
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
import { Pencil, Trash2, ShieldCheck, Laptop, Monitor, HardDrive } from "lucide-react";

export const Route = createFileRoute("/_dash/licenses/$licenseId")({
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
        <div className="space-y-8">
          <Skeleton className="h-8 w-64" />
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
          <Skeleton className="h-40" />
        </div>
      ) : !license ? (
        <p className="text-muted-foreground">License not found</p>
      ) : (
        <div className="space-y-8">
          {/* summary */}
          <div className="flex items-start gap-4">
            <div className="grid size-12 shrink-0 place-items-center rounded-xl bg-foreground text-background">
              <ShieldCheck className="size-6" />
            </div>
            <div className="min-w-0 flex-1 space-y-1.5">
              <div className="flex flex-wrap items-center gap-2.5">
                <h1 className="text-2xl font-semibold tracking-tight">{license.productName}</h1>
                <Badge variant={license.isActive ? "default" : "secondary"}>
                  {license.isActive ? "Active" : "Inactive"}
                </Badge>
                {license.isTrial && <Badge variant="outline">Trial</Badge>}
              </div>
              <p className="text-muted-foreground text-sm">{license.customerEmail}</p>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <Button variant="outline" onClick={() => setEditOpen(true)}>
                <Pencil className="size-4" /> Edit
              </Button>
              <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="size-4" /> Delete
              </Button>
            </div>
          </div>

          <div className="grid gap-x-12 gap-y-8 lg:grid-cols-5">
          {/* left column */}
          <div className="space-y-8 lg:col-span-2">
          {/* seat usage */}
          <div className="space-y-2">
            <div className="flex items-baseline justify-between text-sm">
              <span className="font-medium">Seat usage</span>
              <span className="text-muted-foreground tabular-nums">
                {license.activationCount} / {license.maxActivations} used
              </span>
            </div>
            <div className="bg-muted h-1.5 w-full overflow-hidden rounded-full">
              <div
                className="bg-foreground h-full rounded-full transition-all"
                style={{
                  width: `${Math.min(100, (license.activationCount / Math.max(1, license.maxActivations)) * 100)}%`,
                }}
              />
            </div>
          </div>

          {/* meta grid */}
          <dl className="grid grid-cols-2 gap-x-8 gap-y-6 border-y py-6">
            <Field label="Created">
              {license.createdAt ? new Date(license.createdAt).toLocaleDateString() : "—"}
            </Field>
            <Field label="Expires">
              {license.expiresAt ? new Date(license.expiresAt).toLocaleDateString() : "Never"}
            </Field>
            <Field label="Type">{license.isTrial ? "Trial" : "Standard"}</Field>
            <Field label="License ID">
              <span className="font-mono text-xs">{license.id.slice(0, 8)}</span>
            </Field>
          </dl>

          {/* features */}
          <section className="space-y-3">
            <h2 className="text-sm font-medium">Features</h2>
            {license.features?.length ? (
              <div className="flex flex-wrap gap-1.5">
                {license.features.map((f) => (
                  <Badge key={f} variant="secondary" className="rounded-full">{f}</Badge>
                ))}
              </div>
            ) : (
              <p className="text-muted-foreground text-sm">No features assigned.</p>
            )}
          </section>
          </div>

          {/* right column: activations */}
          <section className="space-y-3 lg:col-span-3">
            <div className="flex items-baseline justify-between">
              <h2 className="text-sm font-medium">Activations</h2>
              <span className="text-muted-foreground text-xs tabular-nums">
                {license.activations.length} device{license.activations.length === 1 ? "" : "s"}
              </span>
            </div>
            {!license.activations.length ? (
              <div className="rounded-xl border border-dashed py-12 text-center">
                <Monitor className="text-muted-foreground/50 mx-auto size-6" />
                <p className="text-muted-foreground mt-2 text-sm">No devices activated yet.</p>
              </div>
            ) : (
              <ul className="divide-border divide-y overflow-hidden rounded-xl border">
                {license.activations.map((a) => {
                  const name = a.hostname ?? "";
                  const Icon = /mac|book/i.test(name)
                    ? Laptop
                    : /linux|server|ubuntu/i.test(name)
                      ? HardDrive
                      : Monitor;
                  return (
                    <li key={a.id} className="flex items-center gap-3.5 px-4 py-3.5">
                      <div className="bg-muted grid size-9 shrink-0 place-items-center rounded-lg">
                        <Icon className="text-muted-foreground size-4" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium">{a.hostname ?? "Unknown device"}</div>
                        <div className="text-muted-foreground truncate font-mono text-xs">{a.deviceId}</div>
                      </div>
                      <div className="text-muted-foreground hidden shrink-0 text-right text-xs sm:block">
                        <div>{a.checkedInAt ? `Seen ${new Date(a.checkedInAt).toLocaleDateString()}` : "Never seen"}</div>
                        <div>{a.createdAt ? `Added ${new Date(a.createdAt).toLocaleDateString()}` : ""}</div>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
          </div>
        </div>
      )}
    </AdminShell>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-1">
      <dt className="text-muted-foreground text-xs font-medium uppercase tracking-wide">{label}</dt>
      <dd className="text-sm">{children}</dd>
    </div>
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
