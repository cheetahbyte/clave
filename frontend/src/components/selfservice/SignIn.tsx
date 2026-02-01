import { sendMagicSelfServiceLink } from "@/actions/selfservice"
import { Label } from "@radix-ui/react-label"
import { CardHeader, CardTitle, CardDescription, CardContent, CardFooter, Card } from "../ui/card"
import { Input } from "../ui/input"
import { Button } from "../ui/button"

const SelfServiceSignIn = async () => {
  return <main className="min-h-dvh w-full flex items-center justify-center bg-linear-to-b from-slate-50 to-white px-6 dark:from-slate-950 dark:to-slate-950">
    <div className="w-full max-w-xl">
      <Card className="mx-auto w-full max-w-sm">
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
      </div>
  </main>
}

export default SelfServiceSignIn;
