import { sendMagicSelfServiceLink } from "@/actions/selfservice";
import { Alert, AlertTitle, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export async function SignInForm() {
  return <Card className="mx-auto w-full max-w-sm">
    <CardHeader>
      <CardTitle className="text-2xl">Sign in</CardTitle>
      <CardDescription>Use the license associated with your account.</CardDescription>
    </CardHeader>

    <form noValidate className='flex flex-col gap-2' action={sendMagicSelfServiceLink}>
      <CardContent className="space-y-4">
        {/*{serverError && (
          <Alert variant="destructive">
            <AlertTitle>Something went wrong</AlertTitle>
            <AlertDescription>{serverError}</AlertDescription>
          </Alert>
        )}*/}

        {/*{success && (
          <Alert>
            <AlertTitle>Email sent</AlertTitle>
            <AlertDescription>{success}</AlertDescription>
          </Alert>
        )}*/}

        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            name="email"
            type="email"
            autoComplete="email"
            placeholder="you@example.com"
          />
          {/*{errors.email && (
            <p className="text-sm text-destructive" role="alert">
              {errors.email.message}
            </p>
          )}*/}
        </div>
      </CardContent>

      <CardFooter className="flex flex-col gap-3 mt-2">
        <Button className="w-full" type="submit" >
          send magic link
          {/*{isSubmitting ? 'Sending link…' : 'Send magic link'}*/}
        </Button>
      </CardFooter>
    </form>
  </Card>
}
