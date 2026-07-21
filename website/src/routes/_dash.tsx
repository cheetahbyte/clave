import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { ProductProvider } from "@/features/admin/product-context";
import { getCurrentAdmin } from "@/features/admin/api";

export const Route = createFileRoute("/_dash")({
  beforeLoad: async ({ context }) => {
    try {
      const admin = await context.queryClient.ensureQueryData({
        queryKey: ["currentAdmin"],
        queryFn: getCurrentAdmin,
      });
      if (!admin.mfaVerified) {
        throw redirect({ to: "/2fa" });
      }
    } catch (err) {
      if (
        err instanceof Error &&
        "redirect" in (err as unknown as Record<string, unknown>)
      )
        throw err;
      throw redirect({ to: "/login" });
    }
  },
  component: DashLayout,
});

function DashLayout() {
  return (
    <ProductProvider>
      <SidebarProvider>
        <AppSidebar />
        <Outlet />
      </SidebarProvider>
    </ProductProvider>
  );
}
