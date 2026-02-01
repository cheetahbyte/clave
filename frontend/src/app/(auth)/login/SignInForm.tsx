'use client';

import * as React from 'react';
import { z } from 'zod';
import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';

// IMPORTANT: import your Better Auth client
import { authClient } from '@/lib/auth-client';

const LoginSchema = z.object({
  email: z.email('Please enter a valid email.'),
});

type LoginValues = z.infer<typeof LoginSchema>;

export function LoginForm() {
  const [serverError, setServerError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);

  const form = useForm<LoginValues>({
    resolver: zodResolver(LoginSchema),
    defaultValues: { email: '' },
    mode: 'onSubmit',
  });

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = form;

  async function onSubmit(values: LoginValues) {
    setServerError(null);
    setSuccess(null);

    try {
      const { data, error } = await authClient.signIn.magicLink({
        email: values.email,
        callbackURL: '/dashboard',
        errorCallbackURL: '/error',
        newUserCallbackURL: '/welcome',
      });

      if (error) {
        const message =
          (typeof error === 'string' && error) ||
          (error as any)?.message ||
          (error as any)?.error?.message ||
          (error as any)?.data?.message ||
          'Could not send magic link. Please try again.';
        setServerError(message);
        return;
      }

      setSuccess('Check your inbox — we sent you a magic link.');
      // optional: reset email field after success
      // form.reset({ email: values.email });
    } catch (err: any) {
      const message =
        err?.message ||
        err?.error?.message ||
        err?.data?.message ||
        'Unexpected error. Please try again.';
      setServerError(message);
    }
  }

  return (
    <Card className="mx-auto w-full max-w-sm">
      <CardHeader>
        <CardTitle className="text-2xl">Sign in</CardTitle>
        <CardDescription>We’ll email you a magic link to sign in.</CardDescription>
      </CardHeader>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className='flex flex-col gap-2'>
        <CardContent className="space-y-4">
          {serverError && (
            <Alert variant="destructive">
              <AlertTitle>Something went wrong</AlertTitle>
              <AlertDescription>{serverError}</AlertDescription>
            </Alert>
          )}

          {success && (
            <Alert>
              <AlertTitle>Email sent</AlertTitle>
              <AlertDescription>{success}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              {...register('email')}
              aria-invalid={!!errors.email}
            />
            {errors.email && (
              <p className="text-sm text-destructive" role="alert">
                {errors.email.message}
              </p>
            )}
          </div>
        </CardContent>

        <CardFooter className="flex flex-col gap-3 mt-2">
          <Button className="w-full" type="submit" disabled={isSubmitting}>
            {isSubmitting ? 'Sending link…' : 'Send magic link'}
          </Button>

          <p className="text-sm text-muted-foreground">
            No account?{' '}
            <a href="/register" className="underline underline-offset-4 hover:text-foreground">
              Create one
            </a>
          </p>
        </CardFooter>
      </form>
    </Card>
  );
}
