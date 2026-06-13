import * as React from "react"
import { Building2, Check, ChevronsUpDown, Loader2, Plus } from "lucide-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useRouter } from "@tanstack/react-router"
import { toast } from "sonner"

import {
  getCurrentAdmin,
  listOrganizations,
  createOrganization,
  switchOrganization,
} from "@/features/admin/api"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

export function OrgSwitcher() {
  const { isMobile } = useSidebar()
  const queryClient = useQueryClient()
  const router = useRouter()
  const [createOpen, setCreateOpen] = React.useState(false)
  const [name, setName] = React.useState("")

  const { data: admin } = useQuery({ queryKey: ["currentAdmin"], queryFn: getCurrentAdmin })
  const { data: orgs } = useQuery({ queryKey: ["organizations"], queryFn: listOrganizations })

  const activeId = admin?.organizationId
  const activeName =
    orgs?.find((o) => o.id === activeId)?.name ?? admin?.organizationName ?? "Organization"
  const activeRole = orgs?.find((o) => o.id === activeId)?.role

  async function refreshAll() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["currentAdmin"] }),
      queryClient.invalidateQueries({ queryKey: ["organizations"] }),
      queryClient.invalidateQueries({ queryKey: ["adminOverview"] }),
      queryClient.invalidateQueries({ queryKey: ["adminLicenses"] }),
      queryClient.invalidateQueries({ queryKey: ["adminProducts"] }),
      queryClient.invalidateQueries({ queryKey: ["adminTrials"] }),
      queryClient.invalidateQueries({ queryKey: ["adminTimeseries"] }),
      queryClient.invalidateQueries({ queryKey: ["adminAudit"] }),
    ])
    await router.invalidate()
  }

  const switchMut = useMutation({
    mutationFn: switchOrganization,
    onSuccess: refreshAll,
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to switch organization"),
  })

  const createMut = useMutation({
    mutationFn: () => createOrganization(name.trim()),
    onSuccess: async () => {
      setCreateOpen(false)
      setName("")
      toast.success("Organization created")
      await refreshAll()
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to create organization"),
  })

  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <SidebarMenuButton
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              >
                <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
                  {switchMut.isPending ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <Building2 className="size-4" />
                  )}
                </div>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">{activeName}</span>
                  <span className="text-muted-foreground truncate text-xs capitalize">
                    {activeRole ?? "Clave"}
                  </span>
                </div>
                <ChevronsUpDown className="ml-auto size-4" />
              </SidebarMenuButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="w-(--radix-dropdown-menu-trigger-width) min-w-60 rounded-lg"
              align="start"
              side={isMobile ? "bottom" : "right"}
              sideOffset={4}
            >
              <DropdownMenuLabel className="text-muted-foreground text-xs">
                Organizations
              </DropdownMenuLabel>
              {orgs?.map((org) => (
                <DropdownMenuItem
                  key={org.id}
                  className="gap-2 p-2"
                  disabled={switchMut.isPending}
                  onClick={() => {
                    if (org.id !== activeId) switchMut.mutate(org.id)
                  }}
                >
                  <div className="flex size-6 items-center justify-center rounded-md border">
                    <Building2 className="size-3.5 shrink-0" />
                  </div>
                  <span className="flex-1 truncate">{org.name}</span>
                  {org.id === activeId ? <Check className="size-4" /> : null}
                </DropdownMenuItem>
              ))}
              <DropdownMenuSeparator />
              <DropdownMenuItem className="gap-2 p-2" onClick={() => setCreateOpen(true)}>
                <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                  <Plus className="size-4" />
                </div>
                <div className="text-muted-foreground font-medium">Create organization</div>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create organization</DialogTitle>
            <DialogDescription>
              You'll be switched to the new organization once it's created.
            </DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              if (name.trim().length >= 2) createMut.mutate()
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="org-name">Name</Label>
              <Input
                id="org-name"
                value={name}
                autoFocus
                placeholder="Acme Inc."
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <DialogFooter className="mt-4">
              <Button
                type="submit"
                disabled={createMut.isPending || name.trim().length < 2}
              >
                {createMut.isPending ? "Creating…" : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
