import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { z } from "zod";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { loginAdmin } from "@/features/admin/api";

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
import { Lock, ShieldCheck, LayoutDashboard, ArrowRight } from "lucide-react";

const LoginSchema = z.object({
  email: z.string().email("Please enter a valid email."),
  password: z.string().min(1, "Password is required."),
});

type LoginValues = z.infer<typeof LoginSchema>;

export const Route = createFileRoute("/login")({
  component: LoginPage,
});

function LoginPage() {
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const form = useForm<LoginValues>({
    resolver: zodResolver(LoginSchema),
    defaultValues: { email: "", password: "" },
  });

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = form;

  async function onSubmit(values: LoginValues) {
    setError(null);
    setLoading(true);
    try {
      const resp = await loginAdmin(values);

      if (resp.mfaSetupRequired) {
        navigate({ to: "/2fa/setup" });
      } else if (resp.mfaVerificationRequired) {
        navigate({ to: "/2fa" });
      } else {
        navigate({ to: "/dashboard" });
      }
    } catch (err: unknown) {
      const message =
        err instanceof Error ? err.message : "Login failed. Please try again.";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="grid min-h-dvh lg:grid-cols-2">
      {/* hero panel */}
      <div className="relative hidden flex-col justify-between bg-linear-to-br from-slate-950 to-slate-900 p-12 text-slate-50 lg:flex">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,hsl(var(--primary)/.08),transparent_60%)]" />
        <div>
          <div className="inline-flex size-10 items-center justify-center rounded-xl border border-slate-700 bg-slate-800/50">
            <Lock className="size-5" />
          </div>
        </div>
        <div className="relative space-y-4">
          <h1 className="text-3xl font-bold tracking-tight">Clave admin</h1>
          <p className="max-w-md text-pretty text-base leading-relaxed text-slate-400">
            Manage licenses, products, devices, and organizations from one dashboard.
          </p>
          <div className="flex flex-col gap-3 pt-2">
            <div className="flex items-center gap-3 text-sm text-slate-400">
              <span className="inline-flex size-6 shrink-0 items-center justify-center rounded-md border border-slate-700 bg-slate-800/50 text-slate-300">
                <LayoutDashboard className="size-3" />
              </span>
              Full license &amp; product management
            </div>
            <div className="flex items-center gap-3 text-sm text-slate-400">
              <span className="inline-flex size-6 shrink-0 items-center justify-center rounded-md border border-slate-700 bg-slate-800/50 text-slate-300">
                <ShieldCheck className="size-3" />
              </span>
              Two-factor protected access
            </div>
          </div>
        </div>
        <div className="relative text-xs text-slate-600">Clave &middot; license management</div>
      </div>

      {/* form */}
      <div className="flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-sm">
          <div className="mb-8 lg:hidden">
            <h1 className="text-2xl font-bold tracking-tight text-slate-900 dark:text-slate-50">
              Clave admin
            </h1>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
              Sign in to your dashboard
            </p>
          </div>

          <Card className="border-0 shadow-none lg:border lg:shadow-sm">
            <CardHeader className="px-0 lg:px-6">
              <CardTitle className="text-xl">Admin sign in</CardTitle>
              <CardDescription>Enter your credentials to access the dashboard.</CardDescription>
            </CardHeader>

            <form onSubmit={handleSubmit(onSubmit)} noValidate>
              <CardContent className="space-y-4 px-0 lg:px-6">
                {error && (
                  <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}

                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    placeholder="you@example.com"
                    {...register("email")}
                  />
                  {errors.email && (
                    <p className="text-sm text-destructive">{errors.email.message}</p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="password">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    autoComplete="current-password"
                    {...register("password")}
                  />
                  {errors.password && (
                    <p className="text-sm text-destructive">{errors.password.message}</p>
                  )}
                </div>
              </CardContent>

              <CardFooter className="mt-2 flex-col px-0 lg:px-6">
                <Button className="w-full gap-1.5" type="submit" disabled={loading}>
                  {loading ? "Signing in…" : "Sign in"}
                  {!loading && <ArrowRight className="size-4" />}
                </Button>
              </CardFooter>
            </form>
          </Card>

          <div className="mt-6 hidden text-center text-xs text-slate-400 lg:block">
            Secure access &middot; two-factor authentication
          </div>
        </div>
      </div>
    </div>
  );
}
