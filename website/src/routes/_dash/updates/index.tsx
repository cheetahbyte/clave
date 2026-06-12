import { createFileRoute, redirect } from "@tanstack/react-router";
import { getCurrentAdmin } from "@/features/admin/api";

export const Route = createFileRoute("/_dash/updates/")({
  beforeLoad: async () => {
    try {
      const admin = await getCurrentAdmin();
      if (!admin.mfaVerified) {
        if (admin.mfaEnabled) throw redirect({ to: "/2fa" });
        throw redirect({ to: "/2fa/setup" });
      }
    } catch (err) {
      if (err instanceof Error && "redirect" in (err as unknown as Record<string, unknown>)) throw err;
      throw redirect({ to: "/login" });
    }
  },
  loader: () => {
    throw redirect({ to: "/updates/releases" });
  },
});
