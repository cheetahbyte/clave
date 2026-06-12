import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { validateSelfServiceToken } from "@/features/selfservice/api";

export const Route = createFileRoute("/selfservice/$orgSlug/auth")({
  validateSearch: (search: Record<string, unknown>) => ({
    token: typeof search.token === "string" ? search.token : "",
  }),
  component: SelfServiceAuthPage,
});

function SelfServiceAuthPage() {
  const { orgSlug } = Route.useParams();
  const { token } = Route.useSearch();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!token) {
      setError("Missing token.");
      return;
    }

    validateSelfServiceToken(token, orgSlug)
      .then(() => {
        navigate({ to: "/selfservice/$orgSlug", params: { orgSlug }, search: { licenseId: undefined } });
      })
      .catch(() => {
        setError("Invalid or expired token.");
      });
  }, [token, orgSlug, navigate]);

  if (error) {
    return (
      <div className="grid min-h-screen place-items-center p-6">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-red-600">Authentication failed</h1>
          <p className="text-muted-foreground mt-2">{error}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="grid min-h-screen place-items-center p-6">
      <p className="text-muted-foreground">Validating your token...</p>
    </div>
  );
}
