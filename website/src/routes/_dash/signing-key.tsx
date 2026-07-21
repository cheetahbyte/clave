import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { getSigningKey } from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Copy, ShieldAlert } from "lucide-react";
import { toast } from "sonner";

export const Route = createFileRoute("/_dash/signing-key")({
  component: SigningKeyPage,
});

function SigningKeyPage() {
  const { data: signingKey } = useQuery({
    queryKey: ["signingKey"],
    queryFn: getSigningKey,
  });

  return (
    <AdminShell title="Client signing key">
      <div className="max-w-3xl space-y-8">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Client signing key</h1>
          <p className="text-muted-foreground text-sm">
            Public half of the {signingKey?.algorithm ?? "Ed25519"} keypair this server signs
            license tokens and delta update contracts with.
          </p>
        </div>

        <Alert>
          <ShieldAlert className="size-4" />
          <AlertTitle>Embed this key at build time</AlertTitle>
          <AlertDescription>
            Ship it inside the signed application bundle. Clients must never fetch a verification
            key over the network — an attacker who can impersonate this server would otherwise
            supply both a forged update and the key that validates it.
          </AlertDescription>
        </Alert>

        {signingKey ? (
          <div className="space-y-8">
            <div className="space-y-2">
              <h2 className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
                Public key (raw 32 bytes, Base64)
              </h2>
              <div className="flex items-center gap-2">
                <code className="bg-muted flex-1 truncate rounded-md px-3 py-2 font-mono text-xs">
                  {signingKey.publicKey}
                </code>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8 shrink-0"
                  onClick={() => {
                    navigator.clipboard.writeText(signingKey.publicKey);
                    toast.success("Copied");
                  }}
                >
                  <Copy className="size-4" />
                </Button>
              </div>
            </div>

            <Separator />

            <div className="space-y-2">
              <h2 className="text-muted-foreground text-xs font-medium uppercase tracking-wide">
                SHA-256 fingerprint
              </h2>
              <p className="font-mono text-base font-medium">
                {signingKey.fingerprint.split(":").slice(0, 4).join(":")}
              </p>
              <p className="text-muted-foreground font-mono text-xs break-all">
                {signingKey.fingerprint}
              </p>
              <p className="text-muted-foreground text-xs">
                Compare the short form against the fingerprint a client logs for its embedded key to
                confirm the two still match.
              </p>
            </div>
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">Loading signing key…</p>
        )}
      </div>
    </AdminShell>
  );
}
