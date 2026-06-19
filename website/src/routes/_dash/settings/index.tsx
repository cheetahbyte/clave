import { createFileRoute, Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  getCurrentAdmin,
  getMCPToken,
  listOrganizations,
  regenerateMCPToken,
} from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Copy, ExternalLink, KeyRound, RefreshCw, ShieldAlert, Users } from "lucide-react";
import { toast } from "sonner";

export const Route = createFileRoute("/_dash/settings/")({
  component: SettingsPage,
});

function SettingsPage() {
  const queryClient = useQueryClient();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [newMCPToken, setNewMCPToken] = useState<string | null>(null);

  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
  });

  const { data: orgs } = useQuery({
    queryKey: ["adminOrganizations"],
    queryFn: listOrganizations,
  });

  const { data: mcpToken } = useQuery({
    queryKey: ["mcpToken"],
    queryFn: getMCPToken,
  });

  const regenerateMut = useMutation({
    mutationFn: regenerateMCPToken,
    onSuccess: (data) => {
      setNewMCPToken(data.token);
      toast.success("MCP token regenerated");
      queryClient.invalidateQueries({ queryKey: ["mcpToken"] });
      setConfirmOpen(false);
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : "Failed to regenerate MCP token");
    },
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

        <Card className="max-w-xl lg:col-span-2 lg:max-w-none">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <KeyRound className="size-5" />
              MCP access token
            </CardTitle>
            <CardDescription>
              Use this organization token for MCP clients calling <code>/api/v1/mcp</code>.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 text-sm">
            <div className="grid gap-3 md:grid-cols-3">
              <div>
                <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
                  Status
                </p>
                <p className="mt-1 font-medium">{mcpToken?.exists ? "Configured" : "Not created"}</p>
              </div>
              <div>
                <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
                  Prefix
                </p>
                <p className="mt-1 font-mono text-xs">{mcpToken?.prefix ?? "—"}</p>
              </div>
              <div>
                <p className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
                  Last used
                </p>
                <p className="mt-1 font-medium">
                  {mcpToken?.lastUsedAt ? new Date(mcpToken.lastUsedAt).toLocaleString() : "Never"}
                </p>
              </div>
            </div>

            {newMCPToken ? (
              <Alert>
                <ShieldAlert className="size-4" />
                <AlertTitle>Copy token now</AlertTitle>
                <AlertDescription>
                  <div className="flex w-full items-center gap-2 pt-2">
                    <code className="bg-muted flex-1 truncate rounded-md px-3 py-2 font-mono text-xs">
                      {newMCPToken}
                    </code>
                    <Button
                      variant="outline"
                      size="icon"
                      className="size-8 shrink-0"
                      onClick={() => {
                        navigator.clipboard.writeText(newMCPToken);
                        toast.success("Copied");
                      }}
                    >
                      <Copy className="size-4" />
                    </Button>
                  </div>
                  <p className="pt-2">This raw token is shown once. Store it in your MCP client.</p>
                </AlertDescription>
              </Alert>
            ) : null}

            <div className="flex flex-wrap items-center gap-3">
              <Button onClick={() => setConfirmOpen(true)}>
                <RefreshCw className="size-4" />
                {mcpToken?.exists ? "Regenerate token" : "Create token"}
              </Button>
              {mcpToken?.regeneratedAt ? (
                <span className="text-muted-foreground text-xs">
                  Last regenerated {new Date(mcpToken.regeneratedAt).toLocaleString()}
                </span>
              ) : null}
            </div>
          </CardContent>
        </Card>
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={mcpToken?.exists ? "Regenerate MCP token?" : "Create MCP token?"}
        description={
          mcpToken?.exists
            ? "Existing MCP clients using the old token will stop working immediately."
            : "A new bearer token will be created for this organization. It will be shown once."
        }
        confirmLabel={mcpToken?.exists ? "Regenerate" : "Create token"}
        destructive={Boolean(mcpToken?.exists)}
        pending={regenerateMut.isPending}
        onConfirm={() => regenerateMut.mutate()}
      />
    </AdminShell>
  );
}
