import Link from "next/link";
import { checkTokenValidity, getLicenses } from "@/actions/selfservice";

import { ArrowLeft, Boxes, Calendar, Laptop, Monitor, ShieldAlert } from "lucide-react";
import LicenseSelect from "@/components/selfservice/LicenseSelect";
import LicenseDetails from "@/components/selfservice/LicenseDetails";
import SelfServiceSignIn from "@/components/selfservice/SignIn";

export type License = {
  is_active: boolean;
  id: number;
  max_activations: number;
  name: string;
  expires_at?: Date;
};

type Device = {
  id: string;
  name: string;
  os?: string;
  ip?: string;
  last_seen?: Date;
};

function formatDate(d?: Date) {
  if (!d) return "—";
  const dd = typeof d === "string" ? new Date(d) : d;
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(dd);
}

function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}

function toInt(v: unknown) {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

export default async function SelfServicePage({
  searchParams,
}: {
  searchParams?: Promise<{ licenseId?: string }>;
}) {
  const tokenValid = await checkTokenValidity();
  if (!tokenValid) return <SelfServiceSignIn/>;

  const licenses = (await getLicenses()) as License[];

  const sp = await searchParams;
  const selectedId = sp?.licenseId ? Number(sp.licenseId) : null;
  const selected =
    selectedId != null ? licenses.find((l) => l.id === selectedId) ?? null : null;

  if (selected) {
    return (
      <main className="min-h-screen w-full bg-linear-to-b from-slate-50 to-white px-6 py-12 dark:from-slate-950 dark:to-slate-950">
        <div className="mx-auto w-full max-w-5xl">
          {/* Header: only back arrow */}
          <div className="mb-8">
            <Link
              href="/selfservice"
              className="inline-flex size-10 items-center justify-center rounded-xl border bg-white shadow-xs hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900 dark:hover:bg-slate-800"
              aria-label="Back"
            >
              <ArrowLeft className="size-4" />
            </Link>
          </div>

          <LicenseDetails license={selected} />
        </div>
      </main>
    );
  }
  return (
    <LicenseSelect licenses={licenses} />
  );
}
