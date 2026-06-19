import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { disable2FA, getCurrentAdmin } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ShieldOff } from "lucide-react";
import { toast } from "sonner";

export const Route = createFileRoute("/_dash/account")({
  component: AccountPage,
});

function AccountPage() {
  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = useState(false);

  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
  });

  const disableMut = useMutation({
    mutationFn: disable2FA,
    onSuccess: () => {
      toast.success("2FA disabled");
      queryClient.invalidateQueries({ queryKey: ["currentAdmin"] });
      setConfirmOpen(false);
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to disable 2FA");
    },
  });

  const rows = [
    { label: "Email", value: admin?.email },
    { label: "Role", value: admin?.role, className: "capitalize" },
    {
      label: "Two-factor auth",
      value: (
        <Badge variant={admin?.mfaEnabled ? "default" : "secondary"}>
          {admin?.mfaEnabled ? "Enabled" : "Not enabled"}
        </Badge>
      ),
    },
    ...(admin?.last_login_at
      ? [{ label: "Last login", value: new Date(admin.last_login_at).toLocaleString() }]
      : []),
    ...(admin?.created_at
      ? [{ label: "Joined", value: new Date(admin.created_at).toLocaleDateString() }]
      : []),
  ];

  return (
    <AdminShell title="Account">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Account</h1>
        <p className="text-muted-foreground text-sm">Your admin account details.</p>
      </div>

      <Card className="max-w-xl">
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>Details for your admin account.</CardDescription>
        </CardHeader>
        <CardContent className="text-sm">
          <dl className="divide-border divide-y">
            {rows.map((row) => (
              <div key={row.label} className="flex items-center justify-between py-3 first:pt-0 last:pb-0">
                <dt className="text-muted-foreground">{row.label}</dt>
                <dd className={`font-medium ${row.className ?? ""}`}>{row.value ?? "\u2014"}</dd>
              </div>
            ))}
          </dl>
        </CardContent>
      </Card>

      {admin?.mfaEnabled ? (
        <Card className="max-w-xl">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ShieldOff className="size-5" />
              Disable 2FA
            </CardTitle>
            <CardDescription>
              Dev-only convenience for local testing. Not available in production.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button variant="destructive" onClick={() => setConfirmOpen(true)} disabled={disableMut.isPending}>
              {disableMut.isPending ? "Working…" : "Disable 2FA"}
            </Button>
          </CardContent>
        </Card>
      ) : null}

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Disable 2FA?"
        description="Two-factor authentication will be turned off for your account. Recovery codes will be invalidated. This endpoint only works in dev mode."
        confirmLabel="Disable"
        destructive
        pending={disableMut.isPending}
        onConfirm={() => disableMut.mutate()}
      />
    </AdminShell>
  );
}
