import { createFileRoute, redirect, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { getCurrentAdmin, listOrganizations } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Copy, ExternalLink, Users } from "lucide-react";
import { toast } from "sonner";

export const Route = createFileRoute("/settings/")({
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
  component: SettingsPage,
});

function SettingsPage() {
  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
  });

  const { data: orgs } = useQuery({
    queryKey: ["adminOrganizations"],
    queryFn: listOrganizations,
  });

  const org = orgs?.find((o) => o.id === admin?.organizationId);
  const portalUrl = org ? `${window.location.origin}/selfservice/${org.slug}` : null;

  return (
    <AdminShell title="Settings">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Organization settings</h1>
        <p className="text-muted-foreground text-sm">
          Manage your organization profile and access.
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="max-w-xl lg:max-w-none">
          <CardHeader>
            <CardTitle>Organization</CardTitle>
            <CardDescription>Details about your current organization.</CardDescription>
          </CardHeader>
          <CardContent className="text-sm">
            <dl className="divide-border divide-y">
              <div className="flex items-center justify-between py-3 first:pt-0">
                <dt className="text-muted-foreground">Name</dt>
                <dd className="font-medium">{admin?.organizationName ?? "\u2014"}</dd>
              </div>
              <div className="flex items-center justify-between py-3">
                <dt className="text-muted-foreground">Slug</dt>
                <dd className="font-medium font-mono text-xs">{org?.slug ?? "\u2014"}</dd>
              </div>
              <div className="flex items-center justify-between py-3">
                <dt className="text-muted-foreground">ID</dt>
                <dd className="font-medium font-mono text-xs">{admin?.organizationId ?? "\u2014"}</dd>
              </div>
              <div className="flex items-center justify-between py-3">
                <dt className="text-muted-foreground">Your role</dt>
                <dd className="font-medium capitalize">{org?.role ?? admin?.role ?? "\u2014"}</dd>
              </div>
              {org?.createdAt ? (
                <div className="flex items-center justify-between py-3">
                  <dt className="text-muted-foreground">Created</dt>
                  <dd className="font-medium">{new Date(org.createdAt).toLocaleDateString()}</dd>
                </div>
              ) : null}
            </dl>
          </CardContent>
        </Card>

        <Card className="max-w-xl lg:max-w-none">
          <CardHeader>
            <CardTitle>Self-service portal</CardTitle>
            <CardDescription>
              Customers can manage their licenses at this URL.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 text-sm">
            {portalUrl ? (
              <>
                <div className="flex items-center gap-2">
                  <code className="bg-muted flex-1 truncate rounded-md px-3 py-2 font-mono text-xs">
                    {portalUrl}
                  </code>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-8 shrink-0"
                    onClick={() => {
                      navigator.clipboard.writeText(portalUrl);
                      toast.success("Copied");
                    }}
                  >
                    <Copy className="size-4" />
                  </Button>
                </div>
                <a
                  href={portalUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-sm text-primary underline-offset-4 hover:underline"
                >
                  Open portal
                  <ExternalLink className="size-3.5" />
                </a>
              </>
            ) : (
              <p className="text-muted-foreground">No organization selected.</p>
            )}
          </CardContent>
        </Card>

        <Card className="max-w-xl lg:max-w-none lg:col-span-2">
          <CardHeader>
            <CardTitle>Members &amp; access</CardTitle>
            <CardDescription>
              Invite team members and manage roles.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex items-center gap-3">
            <Button variant="outline" asChild>
              <Link to="/organization">
                <Users className="size-4" />
                Manage members
              </Link>
            </Button>
            {org ? (
              <Badge variant="secondary" className="capitalize">
                {org.role}
              </Badge>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </AdminShell>
  );
}
