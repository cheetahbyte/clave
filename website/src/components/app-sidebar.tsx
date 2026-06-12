import * as React from "react"
import { Key, LayoutDashboard, Package, ScrollText, Settings, Users } from "lucide-react"
import { Link, useLocation, useNavigate } from "@tanstack/react-router"
import { toast } from "sonner"
import { useQuery } from "@tanstack/react-query"
import { getCurrentAdmin, logoutAdmin } from "@/features/admin/api"
import { fetchCsrfToken } from "@/lib/api"
import { NavUser } from "@/components/nav-user"
import { OrgSwitcher } from "@/components/org-switcher"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

const items = [
  { title: "Dashboard", url: "/dashboard", icon: LayoutDashboard },
  { title: "Licenses", url: "/licenses", icon: Key },
  { title: "Products", url: "/products", icon: Package },
  { title: "Organization", url: "/organization", icon: Users },
  { title: "Audit Log", url: "/audit", icon: ScrollText },
  { title: "Settings", url: "/settings", icon: Settings },
] as const

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const navigate = useNavigate()
  const { pathname } = useLocation()

  const { data: admin } = useQuery({
    queryKey: ["currentAdmin"],
    queryFn: getCurrentAdmin,
  })

  async function handleLogout() {
    try {
      await fetchCsrfToken()
      await logoutAdmin()
      navigate({ to: "/login" })
    } catch {
      toast.error("Logout failed")
    }
  }

  return (
    <Sidebar variant="inset" {...props}>
      <SidebarHeader>
        <OrgSwitcher />
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Platform</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => {
                const isActive =
                  pathname === item.url || pathname.startsWith(`${item.url}/`)
                return (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild isActive={isActive} tooltip={item.title}>
                      <Link to={item.url}>
                        <item.icon />
                        <span>{item.title}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        {admin ? (
          <NavUser email={admin.email} role={admin.role} onLogout={handleLogout} />
        ) : null}
      </SidebarFooter>
    </Sidebar>
  )
}
