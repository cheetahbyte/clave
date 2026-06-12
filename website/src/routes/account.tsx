import { createFileRoute, redirect } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { getCurrentAdmin } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

export const Route = createFileRoute("/account")({
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
  component: AccountPage,
});

function AccountPage() {
  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
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
    </AdminShell>
  );
}
