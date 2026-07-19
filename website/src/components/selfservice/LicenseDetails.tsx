import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import {
  Download,
  Laptop,
  Monitor,
  HardDrive,
  ShieldCheck,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";
import type { Device, License } from "@/features/selfservice/api";
import {
  listSelfServiceDevices,
  removeSelfServiceDevice,
  revokeSelfServiceLicense,
} from "@/features/selfservice/api";
import {
  browserDownloadPlatform,
  isLicenseDownloadable,
  latestDownloadURL,
} from "@/features/selfservice/download";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";

interface LicenseDetailsProps {
  license: License;
  orgSlug: string;
}

function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}

function formatDate(input?: string | null) {
  if (!input) return "—";
  const d = new Date(input);
  if (Number.isNaN(d.getTime())) return "—";
  return new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  }).format(d);
}

function licenseStatus(license: License): "active" | "expired" {
  return isLicenseDownloadable(license) ? "active" : "expired";
}

function deviceIcon(name: string) {
  const n = name.toLowerCase();
  if (n.includes("mac") || n.includes("book")) return Laptop;
  if (n.includes("linux") || n.includes("ubuntu") || n.includes("server"))
    return HardDrive;
  return Monitor;
}

export function LicenseDetails({ license, orgSlug }: LicenseDetailsProps) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const [deviceToRemove, setDeviceToRemove] = useState<Device | null>(null);
  const [revokeOpen, setRevokeOpen] = useState(false);
  const [logoError, setLogoError] = useState(false);

  const {
    data: devices = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["selfserviceDevices", license.id],
    queryFn: () => listSelfServiceDevices(license.id),
  });

  const removeDevice = useMutation({
    mutationFn: (deviceId: string) =>
      removeSelfServiceDevice(license.id, deviceId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["selfserviceDevices", license.id],
      });
      setDeviceToRemove(null);
    },
  });

  const revoke = useMutation({
    mutationFn: () => revokeSelfServiceLicense(license.id),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["selfserviceLicenses"] });
      queryClient.invalidateQueries({
        queryKey: ["selfserviceDevices", license.id],
      });
      setRevokeOpen(false);
      toast.success("License revoked. A new key has been emailed to you.");
      if (data.newLicenseId) {
        navigate({
          to: "/selfservice/$orgSlug",
          params: { orgSlug },
          search: { licenseId: data.newLicenseId },
          replace: true,
        });
      }
    },
    onError: (e) =>
      toast.error(e instanceof Error ? e.message : "Failed to revoke license"),
  });

  const status = licenseStatus(license);
  const platform = browserDownloadPlatform();
  const maxActivations = clamp(license.max_activations || 1, 1, 999);
  const used = devices.length;
  const overLimit = used > maxActivations;
  const pct = Math.min((used / maxActivations) * 100, 100);

  return (
    <div className="mx-auto w-full max-w-2xl space-y-8">
      {/* header */}
      <header className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3.5">
          <div className="grid size-11 shrink-0 place-items-center overflow-hidden rounded-xl bg-slate-900 text-white dark:bg-white dark:text-slate-900">
            {license.logo_url && !logoError ? (
              <img
                src={license.logo_url}
                alt=""
                className="h-full w-full object-contain"
                onError={() => setLogoError(true)}
              />
            ) : (
              <ShieldCheck className="size-5" />
            )}
          </div>
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-slate-900 dark:text-slate-50">
              {license.name}
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {license.expires_at
                ? `Valid until ${formatDate(license.expires_at)}`
                : "No expiry"}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {status === "active" &&
            (platform ? (
              <Button asChild variant="outline" size="sm">
                <a href={latestDownloadURL(license.download_url, platform)}>
                  <Download className="size-4" />
                  Download latest
                </a>
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                disabled
                title="A compatible macOS, Windows, or Linux download could not be selected."
              >
                <Download className="size-4" />
                Download unavailable
              </Button>
            ))}
          <span
            className={[
              "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium",
              status === "active"
                ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400"
                : "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
            ].join(" ")}
          >
            {status === "active" ? "Active" : "Expired"}
          </span>
        </div>
      </header>

      {/* devices */}
      <section className="space-y-4">
        <div className="flex items-baseline justify-between">
          <h2 className="text-sm font-medium text-slate-900 dark:text-slate-50">
            Registered devices
          </h2>
          <span className="text-sm tabular-nums text-slate-500 dark:text-slate-400">
            {used} / {maxActivations}
          </span>
        </div>

        <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
          <div
            className="h-full rounded-full bg-slate-900 transition-all dark:bg-slate-100"
            style={{ width: `${pct}%` }}
          />
        </div>

        {overLimit && (
          <div className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
            {used - maxActivations} device
            {used - maxActivations !== 1 ? "s" : ""} above the limit of{" "}
            {maxActivations}. Existing devices continue to work, but no new
            devices can be added.
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center gap-2 py-10 text-sm text-slate-400">
            <Loader2 className="size-4 animate-spin" />
            Loading devices…
          </div>
        ) : isError ? (
          <div className="py-10 text-center text-sm text-slate-400">
            Couldn't load devices.
          </div>
        ) : devices.length === 0 ? (
          <div className="rounded-xl border border-dashed py-10 text-center text-sm text-slate-400 dark:border-slate-800">
            No devices registered yet.
          </div>
        ) : (
          <ul className="divide-y divide-slate-100 overflow-hidden rounded-xl border dark:divide-slate-800 dark:border-slate-800">
            {devices.map((d: Device) => {
              const Icon = deviceIcon(d.name);
              return (
                <li
                  key={d.id}
                  className="flex items-center gap-3.5 bg-white px-4 py-3.5 dark:bg-slate-900"
                >
                  <div className="grid size-9 shrink-0 place-items-center rounded-lg bg-slate-100 dark:bg-slate-800">
                    <Icon className="size-4.5 text-slate-500 dark:text-slate-400" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium text-slate-900 dark:text-slate-50">
                      {d.name || "Unknown device"}
                    </div>
                    <div className="text-xs text-slate-400 dark:text-slate-500">
                      Last seen {formatDate(d.last_seen)}
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="shrink-0 text-slate-400 hover:text-slate-900 dark:hover:text-slate-50"
                    onClick={() => setDeviceToRemove(d)}
                  >
                    Remove
                  </Button>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      {/* revoke */}
      {status === "active" && (
        <section className="flex flex-col gap-3 rounded-xl border border-slate-200 p-4 dark:border-slate-800 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h3 className="text-sm font-medium text-slate-900 dark:text-slate-50">
              Revoke license
            </h3>
            <p className="mt-0.5 text-xs text-slate-500 dark:text-slate-400">
              Permanently disables this license on all devices. A new key will
              be issued.
            </p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="shrink-0 self-start text-red-600 hover:bg-red-50 hover:text-red-700 dark:text-red-400 dark:hover:bg-red-500/10"
            onClick={() => setRevokeOpen(true)}
          >
            Revoke
          </Button>
        </section>
      )}

      <ConfirmDialog
        open={deviceToRemove !== null}
        onOpenChange={(open) => !open && setDeviceToRemove(null)}
        title="Deactivate device"
        description={
          <>
            This frees up the seat used by{" "}
            <span className="font-medium text-slate-900 dark:text-slate-50">
              {deviceToRemove?.name || "this device"}
            </span>
            . This frees up its seat. It will need to reactivate to use the
            license again.
          </>
        }
        confirmLabel="Deactivate device"
        pending={removeDevice.isPending}
        onConfirm={() =>
          deviceToRemove && removeDevice.mutate(deviceToRemove.id)
        }
      />

      <ConfirmDialog
        open={revokeOpen}
        onOpenChange={(open) => !open && setRevokeOpen(false)}
        title="Revoke license"
        description="Revoking permanently disables this license on all devices. This cannot be undone."
        confirmLabel="Revoke license"
        pending={revoke.isPending}
        onConfirm={() => revoke.mutate()}
      />
    </div>
  );
}
