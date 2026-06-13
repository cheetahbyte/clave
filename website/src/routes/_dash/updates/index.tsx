import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_dash/updates/")({
  loader: () => {
    throw redirect({ to: "/updates/releases" });
  },
});
