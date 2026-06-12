import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { previewInvite, acceptInvite } from "@/features/admin/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";

export const Route = createFileRoute("/invite/$token")({
  component: InvitePage,
});

function InvitePage() {
  const { token } = Route.useParams();
  const navigate = useNavigate();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const { data: preview, isLoading } = useQuery({
    queryKey: ["invitePreview", token],
    queryFn: () => previewInvite(token),
  });

  async function handleAccept() {
    setError(null);
    setSubmitting(true);
    try {
      await acceptInvite(token, preview?.requiresPassword ? password : undefined);
      toast.success("Invite accepted — you can now sign in.");
      navigate({ to: "/login" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to accept invite.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="grid min-h-screen place-items-center p-6">
      <Card className="mx-auto w-full max-w-sm">
        {isLoading ? (
          <CardHeader>
            <CardTitle>Loading invite…</CardTitle>
          </CardHeader>
        ) : !preview?.valid ? (
          <>
            <CardHeader>
              <CardTitle>Invite not valid</CardTitle>
              <CardDescription>
                This invite link is invalid, expired, or has already been used.
              </CardDescription>
            </CardHeader>
            <CardFooter>
              <Button variant="outline" className="w-full" onClick={() => navigate({ to: "/login" })}>
                Go to sign in
              </Button>
            </CardFooter>
          </>
        ) : (
          <>
            <CardHeader>
              <CardTitle>Join {preview.organizationName}</CardTitle>
              <CardDescription>
                You've been invited as <strong>{preview.email}</strong>.
              </CardDescription>
            </CardHeader>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleAccept();
              }}
            >
              <CardContent className="space-y-4">
                {error && (
                  <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                {preview.requiresPassword ? (
                  <div className="space-y-2">
                    <Label htmlFor="password">Choose a password</Label>
                    <Input
                      id="password"
                      type="password"
                      autoComplete="new-password"
                      placeholder="At least 8 characters"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                    />
                    <p className="text-muted-foreground text-xs">
                      No account exists for this email — set a password to create one.
                    </p>
                  </div>
                ) : (
                  <p className="text-muted-foreground text-sm">
                    You already have a Clave account. Accept to join this organization, then sign in.
                  </p>
                )}
              </CardContent>
              <CardFooter>
                <Button
                  type="submit"
                  className="w-full"
                  disabled={submitting || (preview.requiresPassword && password.length < 8)}
                >
                  {submitting ? "Accepting…" : "Accept invite"}
                </Button>
              </CardFooter>
            </form>
          </>
        )}
      </Card>
    </div>
  );
}
