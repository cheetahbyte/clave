"use client"
import { License } from "@/actions/selfservice";
import { Calendar, Laptop, Monitor, ShieldAlert } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Progress } from "../ui/progress";
import SelfServiceLicenseCard from "./Card";

type LicenseDetailsProps = {
  license: License;
};

type Device = {
  id: string;
  name: string;
  os?: string;
  ip?: string;
  last_seen?: Date | string | null;
};

function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}

function toInt(v: unknown) {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

function formatDate(input?: unknown) {
  if (!input) return "—";
  const d = input instanceof Date ? input : new Date(String(input));
  if (Number.isNaN(d.getTime())) return "—";

  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "long",
    year: "numeric",
  }).format(d);
}

export default function LicenseDetails({ license }: LicenseDetailsProps) {
  // TODO: Replace with real data from backend
  const devices: Device[] = [
    {
      id: "1",
      name: 'MacBook Pro 16"',
      os: "macOS 14.2",
      ip: "192.168.1.101",
      last_seen: new Date(),
    },
    {
      id: "2",
      name: "Windows Workstation",
      os: "Windows 11 Pro",
      ip: "192.168.1.102",
      last_seen: new Date(),
    },
  ];

  const maxActivations = clamp(toInt(license.max_activations), 1, 999);
  const currentActivations = clamp(devices.length, 0, maxActivations);
  const pct = (currentActivations / maxActivations) * 100;

  return (
    <div className="space-y-6">
      {/* Title row */}
      <div className="min-w-0">
        <div className="text-2xl font-extrabold tracking-tight text-slate-900 dark:text-slate-50">
          {license.name}
        </div>
        <div className="text-sm text-slate-500 dark:text-slate-400">
          License details
        </div>
      </div>

      {/* License info */}
      {/*<Card className="rounded-2xl shadow-sm">
        <CardHeader className="flex flex-row items-start justify-between gap-3">
          <div>
            <CardTitle className="text-xl">License information</CardTitle>
            <div className="mt-1 text-sm text-slate-600 dark:text-slate-300">
              Details about your product license
            </div>
          </div>

          <Badge
            className={[
              "rounded-full px-3 py-1",
              license.is_active
                ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-200"
                : "bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-200",
            ].join(" ")}
          >
            {license.is_active ? "Active" : "Inactive"}
          </Badge>
        </CardHeader>

        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-[1.2fr_0.8fr]">
            <div className="space-y-2">
              <div className="text-sm font-medium text-slate-700 dark:text-slate-200">
                License key
              </div>
              <div className="flex items-center gap-2 rounded-xl border bg-slate-50 px-3 py-2 text-sm text-slate-900 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-50">
                <code className="truncate">••••-••••-••••-••••</code>
              </div>
            </div>

            <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-1">
              <div className="flex items-start gap-2 rounded-xl border bg-white p-3 dark:border-slate-800 dark:bg-slate-900">
                <Calendar className="mt-0.5 size-4 text-slate-400" />
                <div>
                  <div className="text-xs text-slate-500 dark:text-slate-400">
                    Created
                  </div>
                  <div className="text-sm font-semibold text-slate-900 dark:text-slate-50">
                    —
                  </div>
                </div>
              </div>

              <div className="flex items-start gap-2 rounded-xl border bg-white p-3 dark:border-slate-800 dark:bg-slate-900">
                <Calendar className="mt-0.5 size-4 text-slate-400" />
                <div>
                  <div className="text-xs text-slate-500 dark:text-slate-400">
                    Valid until
                  </div>
                  <div className="text-sm font-semibold text-slate-900 dark:text-slate-50">
                    {formatDate(license.expires_at)}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>*/}
      <SelfServiceLicenseCard
        expiry={new Date()}
        product="Hondicard Suite Pro"
        status="revoked"
        image="https://www.hondicard.com/img/logos/hondicard-logo.png"
      />

      {/* Devices */}
      <Card className="rounded-2xl shadow-sm">
        <CardHeader className="flex flex-row items-start justify-between gap-3">
          <div>
            <CardTitle className="text-xl">Registered devices</CardTitle>
            <div className="mt-1 text-sm text-slate-600 dark:text-slate-300">
              {currentActivations} of {maxActivations} seats used
            </div>
          </div>

          <div className="rounded-full border bg-white px-3 py-1 text-sm text-slate-700 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200">
            {currentActivations}/{maxActivations}
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          <Progress value={pct} />

          <div className="space-y-3">
            {devices.map((d) => (
              <div
                key={d.id}
                className="flex items-center gap-3 rounded-2xl border bg-white p-4 dark:border-slate-800 dark:bg-slate-900"
              >
                <div className="flex size-11 items-center justify-center rounded-2xl bg-slate-50 dark:bg-slate-950">
                  {d.name.toLowerCase().includes("mac") ? (
                    <Laptop className="size-5 text-slate-700 dark:text-slate-200" />
                  ) : (
                    <Monitor className="size-5 text-slate-700 dark:text-slate-200" />
                  )}
                </div>

                <div className="min-w-0">
                  <div className="truncate font-semibold text-slate-900 dark:text-slate-50">
                    {d.name}
                  </div>
                  <div className="truncate text-sm text-slate-500 dark:text-slate-400">
                    {(d.os ?? "—") + " • " + (d.ip ?? "—")}
                  </div>
                  <div className="text-xs text-slate-500 dark:text-slate-400">
                    Last seen: {formatDate(d.last_seen)}
                  </div>
                </div>

                <div className="ml-auto">
                  <Button variant="outline" className="rounded-xl">
                    Remove
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Danger zone */}
      <Card className="rounded-2xl border-red-200 bg-red-50/60 shadow-sm dark:border-red-900/40 dark:bg-red-950/20">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-xl text-slate-900 dark:text-slate-50">
            <span className="inline-flex size-10 items-center justify-center rounded-2xl bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-200">
              <ShieldAlert className="size-5" />
            </span>
            Danger zone
          </CardTitle>
        </CardHeader>

        <CardContent className="space-y-4">
          <p className="text-sm text-slate-700 dark:text-slate-200">
            Revoking a license permanently disables it on all devices and issues a new license key.
          </p>

          <hr className="border-red-200/60 dark:border-red-900/40" />

          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <Button className="rounded-xl bg-red-600 hover:bg-red-700">
              Revoke license
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
