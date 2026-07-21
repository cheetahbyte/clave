import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { REGEXP_ONLY_DIGITS } from "input-otp";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { verify2FA, resend2FACode, getCurrentAdmin } from "@/features/admin/api";

import { Button } from "@/components/ui/button";
import {
  InputOTP,
  InputOTPGroup,
  InputOTPSeparator,
  InputOTPSlot,
} from "@/components/ui/input-otp";
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
import { useQuery, useQueryClient } from "@tanstack/react-query";

export const Route = createFileRoute("/2fa/")({
  beforeLoad: async () => {
    try {
      await getCurrentAdmin();
    } catch {
      throw new Error("not authenticated");
    }
  },
  component: TwoFactorPage,
});

function TwoFactorPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [resending, setResending] = useState(false);
  const [cooldown, setCooldown] = useState(0);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setTimeout(() => setCooldown((s) => s - 1), 1000);
    return () => clearTimeout(timer);
  }, [cooldown]);

  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await verify2FA(code);
      await queryClient.invalidateQueries({ queryKey: ["currentAdmin"] });
      navigate({ to: "/dashboard" });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Verification failed.";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  async function handleResend() {
    setError(null);
    setResending(true);
    try {
      await resend2FACode();
      setCode("");
      setCooldown(60);
      toast.success("New code sent.");
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Could not send a new code.";
      setError(message);
      toast.error(message);
    } finally {
      setResending(false);
    }
  }

  return (
    <div className="min-h-screen grid place-items-center p-6">
      <Card className="mx-auto w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-2xl">Two-factor authentication</CardTitle>
          <CardDescription>
            {admin
              ? `We sent a 6-digit code to ${admin.email}. It expires in 10 minutes.`
              : "Enter the 6-digit code we emailed you."}
          </CardDescription>
        </CardHeader>

        <form onSubmit={handleSubmit} noValidate>
          <CardContent className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="flex flex-col items-center gap-3">
              <Label htmlFor="code" className="self-start">
                Verification code
              </Label>
              <InputOTP
                id="code"
                maxLength={6}
                pattern={REGEXP_ONLY_DIGITS}
                autoComplete="one-time-code"
                autoFocus
                value={code}
                onChange={(value) => setCode(value)}
                containerClassName="justify-center"
              >
                <InputOTPGroup>
                  <InputOTPSlot index={0} />
                  <InputOTPSlot index={1} />
                  <InputOTPSlot index={2} />
                </InputOTPGroup>
                <InputOTPSeparator />
                <InputOTPGroup>
                  <InputOTPSlot index={3} />
                  <InputOTPSlot index={4} />
                  <InputOTPSlot index={5} />
                </InputOTPGroup>
              </InputOTP>
            </div>
          </CardContent>

          <CardFooter className="flex-col gap-2 pt-6">
            <Button className="w-full" type="submit" disabled={loading || code.length !== 6}>
              {loading ? "Verifying…" : "Verify"}
            </Button>
            <Button
              className="w-full"
              type="button"
              variant="ghost"
              onClick={handleResend}
              disabled={resending || cooldown > 0}
            >
              {cooldown > 0 ? `Resend code in ${cooldown}s` : resending ? "Sending…" : "Resend code"}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
