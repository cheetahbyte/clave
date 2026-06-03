"use client";

import * as React from "react";
import { ShieldCheck, Calendar } from "lucide-react";

import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";

type LicenseStatus = "active" | "expired" | "revoked";

interface SelfServiceLicenseCardProps {
  status: LicenseStatus;
  product: string;      // license name
  expiry: Date;
  image?: string;
}

function statusMeta(status: LicenseStatus) {
  switch (status) {
    case "active":
      return {
        label: "active",
        badge:
          "border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-900/20 dark:text-emerald-200",
        dot: "bg-emerald-500",
        accent: "from-emerald-500 dark:from-emerald-400",
      };
    case "expired":
      return {
        label: "expired",
        badge:
          "border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200",
        dot: "bg-amber-500",
        accent: "from-amber-500 dark:from-amber-400",
      };
    case "revoked":
      return {
        label: "revoked",
        badge:
          "border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-900/40 dark:bg-rose-900/20 dark:text-rose-200",
        dot: "bg-rose-500",
        accent: "from-rose-500 dark:from-rose-400",
      };
  }
}

export default function SelfServiceLicenseCard({
  status,
  product,
  expiry,
  image,
}: SelfServiceLicenseCardProps) {
  const meta = statusMeta(status);

  return (
    <Card className="relative w-full overflow-hidden rounded-2xl border bg-white shadow-sm dark:bg-slate-900">
      {/* subtle bg accent */}
      <div
        className={[
          "pointer-events-none absolute inset-y-0 right-0 w-1/2 opacity-[0.07]",
          "bg-linear-to-l to-transparent",
          meta.accent,
        ].join(" ")}
      />

      <div className="relative z-10 flex items-start justify-between gap-4 p-5 sm:p-6">
        {/* left */}
        <div className="flex min-w-0 items-center gap-3">
          {image ? (
            <div className="h-10 w-10 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-950">
              <img
                src={image}
                alt={`${product} logo`}
                className="h-full w-full object-cover"
                loading="lazy"
                decoding="async"
                referrerPolicy="no-referrer"
              />
            </div>
          ) : (
            <div className="grid h-10 w-10 place-items-center rounded-xl bg-slate-900/90 shadow-sm dark:bg-slate-100/10">
              <ShieldCheck className="h-5 w-5 text-white dark:text-slate-100" />
            </div>
          )}

          <div className="min-w-0">
            <h3 className="truncate text-xl font-extrabold tracking-tight text-slate-900 dark:text-slate-50">
              {product}
            </h3>

            <div className="mt-1 flex items-center gap-1.5 text-sm text-slate-600 dark:text-slate-400">
              <Calendar className="h-3.5 w-3.5" />
              <span>
                Valid until{" "}
                <span className="font-medium">
                  {expiry.toLocaleDateString("de-DE", {
                    day: "2-digit",
                    month: "2-digit",
                    year: "numeric",
                  })}
                </span>
              </span>
            </div>
          </div>
        </div>

        {/* right */}
        <Badge
          className={`shrink-0 gap-2 rounded-full border px-3 py-1 text-[10px] font-bold uppercase tracking-wider ${meta.badge}`}
        >
          <span className={`h-1.5 w-1.5 rounded-full ${meta.dot}`} />
          {meta.label}
        </Badge>
      </div>
    </Card>
  );
}
