import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { ProductProvider } from "@/features/admin/product-context";
import { getCurrentAdmin } from "@/features/admin/api";

export const Route = createFileRoute("/_dash")({
  beforeLoad: async ({ context }) => {
    let admin;
    try {
      admin = await context.queryClient.ensureQueryData({
        queryKey: ["currentAdmin"],
        queryFn: getCurrentAdmin,
      });
    } catch {
      throw redirect({ to: "/login" });
    }
    if (!admin.mfaVerified) {
      throw redirect({ to: "/2fa" });
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
