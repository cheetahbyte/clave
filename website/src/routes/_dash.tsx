import { createFileRoute, Outlet } from "@tanstack/react-router";
import { SidebarProvider } from "@/components/ui/sidebar";
import { AppSidebar } from "@/components/app-sidebar";
import { ProductProvider } from "@/features/admin/product-context";

export const Route = createFileRoute("/_dash")({
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
