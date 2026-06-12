import { useState } from "react";
import { requestSelfServiceLink } from "@/features/selfservice/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ShieldCheck, ArrowRight, MailCheck } from "lucide-react";

export function SignIn({ orgSlug }: { orgSlug: string }) {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!email) return;

    setError(null);
    setSuccess(false);
    setLoading(true);

    try {
      await requestSelfServiceLink(email, orgSlug);
      setSuccess(true);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="grid min-h-dvh place-items-center bg-white px-6 dark:bg-slate-950">
      <div className="w-full max-w-sm">
        {success ? (
          <div className="text-center">
            <div className="mx-auto grid size-12 place-items-center rounded-2xl bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400">
              <MailCheck className="size-6" />
            </div>
            <h1 className="mt-6 text-xl font-semibold tracking-tight text-slate-900 dark:text-slate-50">
              Check your inbox
            </h1>
            <p className="mx-auto mt-2 max-w-xs text-sm leading-relaxed text-slate-500 dark:text-slate-400">
              We sent a sign-in link to{" "}
              <span className="font-medium text-slate-700 dark:text-slate-300">{email}</span>. It
              expires in 15 minutes.
            </p>
            <button
              type="button"
              onClick={() => setSuccess(false)}
              className="mt-6 text-sm font-medium text-slate-500 underline-offset-4 hover:text-slate-900 hover:underline dark:text-slate-400 dark:hover:text-slate-50"
            >
              Use a different email
            </button>
          </div>
        ) : (
          <>
            <div className="mb-8 text-center">
              <div className="mx-auto grid size-12 place-items-center rounded-2xl bg-slate-900 text-white dark:bg-white dark:text-slate-900">
                <ShieldCheck className="size-6" />
              </div>
              <h1 className="mt-6 text-xl font-semibold tracking-tight text-slate-900 dark:text-slate-50">
                License portal
              </h1>
              <p className="mt-1.5 text-sm text-slate-500 dark:text-slate-400">
                Sign in with the email on your license.
              </p>
            </div>

            <form onSubmit={handleSubmit} noValidate className="space-y-4">
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}

              <div className="space-y-2">
                <Label htmlFor="email" className="sr-only">
                  Email
                </Label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  placeholder="you@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="h-11"
                />
              </div>

              <Button className="h-11 w-full gap-1.5" type="submit" disabled={loading}>
                {loading ? "Sending link…" : "Send magic link"}
                {!loading && <ArrowRight className="size-4" />}
              </Button>
            </form>

            <p className="mt-6 text-center text-xs text-slate-400 dark:text-slate-500">
              No password required &middot; link expires in 15 minutes
            </p>
          </>
        )}
      </div>
    </main>
  );
}
