
import { Boxes } from "lucide-react";
import { CardHeader, CardTitle, CardContent, Card } from "../ui/card";
import { Button } from "../ui/button";
import { License } from "@/actions/selfservice";

type LicenseSelectProps = {
  licenses: License[]
}

const LicenseSelect = async ({licenses}: LicenseSelectProps) => {
  return <main className="min-h-dvh w-full flex items-center justify-center bg-linear-to-b from-slate-50 to-white px-6 dark:from-slate-950 dark:to-slate-950">
      <div className="w-full max-w-xl">
      {/* Header */}
      <div className="mb-10 text-center">

        <h1 className="mt-4 text-4xl font-extrabold tracking-tight text-slate-900 dark:text-slate-50">
          Manage your licenses
        </h1>
        <p className="mx-auto mt-3 max-w-2xl text-base text-slate-600 dark:text-slate-300">
          Select a product to remove devices or revoke a license.
        </p>
      </div>

      {/* Simple selection card */}
      <Card className="rounded-2xl shadow-sm">
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Select a product</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="flex flex-col gap-3 sm:flex-row sm:items-center">
            <div className="relative flex-1">
              <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">
                <Boxes className="size-4" />
              </div>

              {/* simpler: no disabled placeholder locking you out; first option is placeholder */}
              <select
                name="licenseId"
                defaultValue=""
                className="h-11 w-full rounded-xl border bg-white pl-10 pr-3 text-sm text-slate-900 shadow-xs outline-none ring-violet-500/30 focus:ring-4 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-50"
              >
                <option value="">Choose a product…</option>
                {licenses.map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
              </select>
            </div>

            <Button type="submit" className="h-11 rounded-xl">
              Manage
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  </main>
}

export default LicenseSelect;
