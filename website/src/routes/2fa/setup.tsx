import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { REGEXP_ONLY_DIGITS } from "input-otp";
import { useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { toast } from "sonner";
import { start2FASetup, verify2FASetup, getCurrentAdmin } from "@/features/admin/api";
import { fetchCsrfToken } from "@/lib/api";
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

export const Route = createFileRoute("/2fa/setup")({
  beforeLoad: async () => {
    try {
      await getCurrentAdmin();
    } catch {
      throw new Error("not authenticated");
    }
  },
  component: TwoFactorSetupPage,
});

function TwoFactorSetupPage() {
  const navigate = useNavigate();
  const [otpauthUrl, setOtpauthUrl] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleStart() {
    setError(null);
    setLoading(true);
    try {
      await fetchCsrfToken();
      const resp = await start2FASetup();
      setOtpauthUrl(resp.otpauthUrl);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Setup failed.";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  async function handleVerify(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await fetchCsrfToken();
      await verify2FASetup(code);
      navigate({ to: "/dashboard" });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Invalid code.";
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  }

  if (!otpauthUrl) {
    return (
      <div className="min-h-screen grid place-items-center p-6">
        <Card className="mx-auto w-full max-w-sm">
          <CardHeader>
            <CardTitle className="text-2xl">Setup two-factor authentication</CardTitle>
            <CardDescription>
              You must enable 2FA before accessing the dashboard. Scan the QR code with your authenticator app.
            </CardDescription>
          </CardHeader>

          <CardContent className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </CardContent>

          <CardFooter>
            <Button className="w-full" onClick={handleStart} disabled={loading}>
              {loading ? "Generating…" : "Generate QR code"}
            </Button>
          </CardFooter>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen grid place-items-center p-6">
      <Card className="mx-auto w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-2xl">Setup two-factor authentication</CardTitle>
          <CardDescription>
            Scan this QR code with your authenticator app, then enter the 6-digit code to confirm.
          </CardDescription>
        </CardHeader>

        <form onSubmit={handleVerify} noValidate>
          <CardContent className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="flex justify-center p-4 bg-white rounded-xl">
              <QRCodeSVG value={otpauthUrl} size={192} />
            </div>

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

          <CardFooter className="pt-6">
            <Button className="w-full" type="submit" disabled={loading || code.length !== 6}>
              {loading ? "Verifying…" : "Confirm setup"}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
