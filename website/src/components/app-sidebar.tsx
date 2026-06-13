import * as React from "react"
import { FlaskConical, GitBranch, Key, LayoutDashboard, Radio, Rocket, ScrollText, Settings, Users } from "lucide-react"
import { Link, useLocation, useNavigate } from "@tanstack/react-router"
import { toast } from "sonner"
import { useQuery } from "@tanstack/react-query"
import { getCurrentAdmin, logoutAdmin } from "@/features/admin/api"
import { fetchCsrfToken } from "@/lib/api"
import { NavUser } from "@/components/nav-user"
import { OrgSwitcher } from "@/components/org-switcher"
import { ProductSwitcher } from "@/components/product-switcher"
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

  function isActive(url: string) {
    return pathname === url || pathname.startsWith(`${url}/`)
  }

  return (
    <Sidebar variant="inset" {...props}>
      <SidebarHeader>
        <OrgSwitcher />
        <ProductSwitcher />
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Platform</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/dashboard")} tooltip="Dashboard">
                  <Link to="/dashboard"><LayoutDashboard /><span>Dashboard</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/licenses")} tooltip="Licenses">
                  <Link to="/licenses"><Key /><span>Licenses</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/trials")} tooltip="Trials">
                  <Link to="/trials"><FlaskConical /><span>Trials</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Updates</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/updates/releases")} tooltip="Releases">
                  <Link to="/updates/releases"><Rocket /><span>Releases</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/updates/sources")} tooltip="Sources">
                  <Link to="/updates/sources"><Radio /><span>Sources</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/updates/channels")} tooltip="Channels">
                  <Link to="/updates/channels"><GitBranch /><span>Channels</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/updates/changelogs")} tooltip="Changelogs">
                  <Link to="/updates/changelogs"><ScrollText /><span>Changelogs</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Administration</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/organization")} tooltip="Organization">
                  <Link to="/organization"><Users /><span>Organization</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/audit")} tooltip="Audit Log">
                  <Link to="/audit"><ScrollText /><span>Audit Log</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton asChild isActive={isActive("/settings")} tooltip="Settings">
                  <Link to="/settings"><Settings /><span>Settings</span></Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
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
