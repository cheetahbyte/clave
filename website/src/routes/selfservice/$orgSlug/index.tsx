import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { checkSelfServiceSession, listSelfServiceLicenses, logoutSelfService } from "@/features/selfservice/api";
import { SignIn } from "@/components/selfservice/SignIn";
import { LicenseSelect } from "@/components/selfservice/LicenseSelect";
import { LicenseDetails } from "@/components/selfservice/LicenseDetails";
import type { License } from "@/features/selfservice/api";
import { ArrowLeft, LogOut, Loader2, AlertCircle } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

export const Route = createFileRoute("/selfservice/$orgSlug/")({
  validateSearch: (search: Record<string, unknown>) => ({
    licenseId: typeof search.licenseId === "string" ? search.licenseId : undefined,
  }),
  component: SelfServicePage,
});

function SelfServicePage() {
  const { orgSlug } = Route.useParams();
  const { licenseId } = Route.useSearch();

  const {
    data: hasSession,
    isLoading: sessionLoading,
    isError: sessionError,
  } = useQuery({
    queryKey: ["selfserviceSession"],
    queryFn: checkSelfServiceSession,
    retry: false,
  });

  if (sessionLoading) {
    return (
      <div className="grid min-h-dvh place-items-center">
        <div className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
          <Loader2 className="size-4 animate-spin" />
          Loading...
        </div>
      </div>
    );
  }

  if (sessionError) {
    return (
      <div className="grid min-h-dvh place-items-center px-6">
        <Alert variant="destructive" className="max-w-sm">
          <AlertCircle className="size-4" />
          <AlertDescription>
            Unable to verify your session. Please try again.
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!hasSession) {
    return <SignIn orgSlug={orgSlug} />;
  }

  return <SelfServiceContent orgSlug={orgSlug} licenseId={licenseId} />;
}

function SelfServiceContent({
  orgSlug,
  licenseId,
}: {
  orgSlug: string;
  licenseId?: string;
}) {
  const {
    data: licenses = [],
    isLoading,
    isError,
  } = useQuery({
    queryKey: ["selfserviceLicenses"],
    queryFn: listSelfServiceLicenses,
  });

  if (isLoading) {
    return (
      <div className="grid min-h-dvh place-items-center">
        <div className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
          <Loader2 className="size-4 animate-spin" />
          Loading licenses&hellip;
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="grid min-h-dvh place-items-center px-6">
        <Alert variant="destructive" className="max-w-sm">
          <AlertCircle className="size-4" />
          <AlertDescription>
            Failed to load licenses. Please try again.
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  // Single license: skip the selection list, jump straight to its details.
  const single = licenses.length === 1;

  const selected = licenseId
    ? licenses.find((l: License) => l.id === licenseId) ?? null
    : single
      ? licenses[0]
      : null;

  if (selected) {
    return <DetailsView license={selected} orgSlug={orgSlug} showBack={!single} />;
  }

  return <LicenseSelect licenses={licenses} orgSlug={orgSlug} />;
}

function DetailsView({
  license,
  orgSlug,
  showBack,
}: {
  license: License;
  orgSlug: string;
  showBack: boolean;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const logout = useMutation({
    mutationFn: logoutSelfService,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["selfserviceSession"] });
      queryClient.removeQueries({ queryKey: ["selfserviceLicenses"] });
      navigate({ to: "/selfservice/$orgSlug", params: { orgSlug }, search: { licenseId: undefined }, replace: true });
    },
  });

  return (
    <main className="min-h-dvh bg-linear-to-b from-slate-50 to-white px-6 py-12 dark:from-slate-950 dark:to-slate-950">
      <div className="mx-auto mb-8 flex max-w-2xl items-center justify-between">
        {showBack ? (
          <Link
            to="/selfservice/$orgSlug"
            params={{ orgSlug }}
            search={{ licenseId: undefined }}
            className="inline-flex size-9 items-center justify-center rounded-xl border bg-white text-slate-500 shadow-xs hover:bg-slate-50 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-400 dark:hover:bg-slate-800"
            aria-label="Back to license list"
          >
            <ArrowLeft className="size-4" />
          </Link>
        ) : (
          <span />
        )}
        <Button
          variant="ghost"
          size="sm"
          className="gap-1.5"
          onClick={() => logout.mutate()}
          disabled={logout.isPending}
        >
          <LogOut className="size-4" />
          {logout.isPending ? "Signing out..." : "Sign out"}
        </Button>
      </div>
      <LicenseDetails license={license} orgSlug={orgSlug} />
    </main>
  );
}
