import { useNavigate } from "@tanstack/react-router";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  Package,
  ShieldCheck,
  Calendar,
  ChevronRight,
  Download,
  LogOut,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { logoutSelfService } from "@/features/selfservice/api";
import type { License } from "@/features/selfservice/api";
import {
  browserDownloadPlatform,
  isLicenseDownloadable,
  latestDownloadURL,
} from "@/features/selfservice/download";

interface LicenseSelectProps {
  licenses: License[];
  orgSlug: string;
}

function formatExpiry(expiresAt?: string) {
  if (!expiresAt) return null;
  const d = new Date(expiresAt);
  if (isNaN(d.getTime())) return null;
  return new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(d);
}

export function LicenseSelect({ licenses, orgSlug }: LicenseSelectProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const logout = useMutation({
    mutationFn: logoutSelfService,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["selfserviceSession"] });
      queryClient.removeQueries({ queryKey: ["selfserviceLicenses"] });
      navigate({
        to: "/selfservice/$orgSlug",
        params: { orgSlug },
        search: { licenseId: undefined },
        replace: true,
      });
    },
  });

  const [logoErrors, setLogoErrors] = useState<Set<string>>(new Set());
  const platform = browserDownloadPlatform();

  return (
    <main className="min-h-dvh bg-white px-6 py-12 dark:bg-slate-950">
      <div className="mx-auto w-full max-w-2xl space-y-8">
        {/* header */}
        <header className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-50">
              License portal
            </h1>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
              Select a product to view details or manage devices.
            </p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="shrink-0 gap-1.5 text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-50"
            onClick={() => logout.mutate()}
            disabled={logout.isPending}
          >
            <LogOut className="size-4" />
            {logout.isPending ? "Signing out…" : "Sign out"}
          </Button>
        </header>

        {licenses.length === 0 ? (
          <div className="flex flex-col items-center rounded-xl border border-dashed py-16 text-center dark:border-slate-800">
            <div className="grid size-11 place-items-center rounded-xl bg-slate-100 dark:bg-slate-800">
              <Package className="size-5 text-slate-400" />
            </div>
            <h2 className="mt-4 text-sm font-medium text-slate-900 dark:text-slate-50">
              No licenses found
            </h2>
            <p className="mt-1 max-w-xs text-sm text-slate-500 dark:text-slate-400">
              No licenses are linked to your account for this organization.
            </p>
          </div>
        ) : (
          <ul className="divide-y divide-slate-100 overflow-hidden rounded-xl border dark:divide-slate-800 dark:border-slate-800">
            {licenses.map((license) => {
              const expiry = formatExpiry(license.expires_at);
              const downloadable = isLicenseDownloadable(license);
              return (
                <li
                  key={license.id}
                  className="flex items-center bg-white transition-colors hover:bg-slate-50 dark:bg-slate-900 dark:hover:bg-slate-800/50"
                >
                  <button
                    type="button"
                    className="group flex min-w-0 flex-1 items-center gap-3.5 px-4 py-4 text-left"
                    onClick={() =>
                      navigate({
                        to: "/selfservice/$orgSlug",
                        params: { orgSlug },
                        search: { licenseId: license.id },
                      })
                    }
                  >
                    <div className="grid size-11 shrink-0 place-items-center overflow-hidden rounded-xl bg-slate-900 text-white dark:bg-white dark:text-slate-900">
                      {license.logo_url && !logoErrors.has(license.id) ? (
                        <img
                          src={license.logo_url}
                          alt=""
                          className="h-full w-full object-contain"
                          onError={() =>
                            setLogoErrors((prev) =>
                              new Set(prev).add(license.id),
                            )
                          }
                        />
                      ) : (
                        <ShieldCheck className="size-5" />
                      )}
                    </div>

                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <h3 className="truncate text-sm font-medium text-slate-900 dark:text-slate-50">
                          {license.name}
                        </h3>
                        <span
                          className={[
                            "shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide",
                            license.is_active
                              ? "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400"
                              : "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400",
                          ].join(" ")}
                        >
                          {license.is_active ? "Active" : "Inactive"}
                        </span>
                      </div>

                      {expiry && (
                        <div className="mt-0.5 flex items-center gap-1.5 text-xs text-slate-400 dark:text-slate-500">
                          <Calendar className="size-3.5" />
                          <span>Valid until {expiry}</span>
                        </div>
                      )}
                    </div>

                    <ChevronRight className="size-4 shrink-0 text-slate-300 transition-colors group-hover:text-slate-500 dark:text-slate-600 dark:group-hover:text-slate-400" />
                  </button>
                  {downloadable && (
                    <div className="shrink-0 pr-4">
                      {platform ? (
                        <Button asChild variant="outline" size="sm">
                          <a
                            href={latestDownloadURL(
                              license.download_url,
                              platform,
                            )}
                          >
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
                      )}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </main>
  );
}
