import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import {
  getCurrentAdmin,
  listMembers,
  inviteMember,
  deleteInvite,
  removeMember,
} from "@/features/admin/api";
import { AdminShell } from "@/components/admin/AdminShell";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Trash2, UserPlus } from "lucide-react";

export const Route = createFileRoute("/_dash/organization/")({
  component: OrganizationPage,
});

function OrganizationPage() {
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("admin");
  const [inviteLink, setInviteLink] = useState("");

  const { data: admin } = useQuery({ queryKey: ["currentAdmin"], queryFn: getCurrentAdmin });
  const { data, isLoading } = useQuery({ queryKey: ["orgMembers"], queryFn: listMembers });

  const inviteMut = useMutation({
    mutationFn: () => inviteMember(email.trim(), role),
    onSuccess: (res) => {
      toast.success("Invite sent");
      queryClient.invalidateQueries({ queryKey: ["orgMembers"] });
      if (res.token) {
        setInviteLink(`${window.location.origin}/invite/${res.token}`);
      } else {
        resetDialog();
      }
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to invite"),
  });

  const deleteInviteMut = useMutation({
    mutationFn: deleteInvite,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["orgMembers"] }),
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to revoke invite"),
  });

  const removeMemberMut = useMutation({
    mutationFn: removeMember,
    onSuccess: () => {
      toast.success("Member removed");
      queryClient.invalidateQueries({ queryKey: ["orgMembers"] });
    },
    onError: (e) => toast.error(e instanceof Error ? e.message : "Failed to remove member"),
  });

  function resetDialog() {
    setDialogOpen(false);
    setEmail("");
    setRole("admin");
    setInviteLink("");
  }

  const inviteDialog = (
    <Dialog open={dialogOpen} onOpenChange={(o) => (o ? setDialogOpen(true) : resetDialog())}>
      <DialogTrigger asChild>
        <Button>
          <UserPlus className="size-4" /> Invite member
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Invite member</DialogTitle>
          <DialogDescription>Send an email invite to join this organization.</DialogDescription>
        </DialogHeader>
        {inviteLink ? (
          <div className="space-y-3">
            <p className="text-sm font-medium">Invite link (no mailer configured — share it manually):</p>
            <div className="bg-muted rounded-md p-3 font-mono text-xs break-all">{inviteLink}</div>
            <Button
              variant="outline"
              className="w-full"
              onClick={() => navigator.clipboard.writeText(inviteLink)}
            >
              Copy link
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="invite-email">Email</Label>
              <Input
                id="invite-email"
                type="email"
                value={email}
                placeholder="teammate@example.com"
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="invite-role">Role</Label>
              <Select value={role} onValueChange={setRole}>
                <SelectTrigger id="invite-role" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="admin">Admin</SelectItem>
                  <SelectItem value="owner">Owner</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        )}
        <DialogFooter>
          {inviteLink ? (
            <Button onClick={resetDialog}>Done</Button>
          ) : (
            <Button
              onClick={() => inviteMut.mutate()}
              disabled={inviteMut.isPending || !email.trim()}
            >
              {inviteMut.isPending ? "Sending…" : "Send invite"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  return (
    <AdminShell title="Organization">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">Members</h1>
          <p className="text-muted-foreground text-sm">
            People with access to {admin?.organizationName ?? "this organization"}.
          </p>
        </div>
        {inviteDialog}
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Email</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Last login</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              Array.from({ length: 3 }).map((_, i) => (
                <TableRow key={i}>
                  <TableCell><Skeleton className="h-4 w-48" /></TableCell>
                  <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                  <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                  <TableCell />
                </TableRow>
              ))
            ) : !data?.members.length ? (
              <TableRow>
                <TableCell colSpan={4} className="text-muted-foreground h-24 text-center">No members</TableCell>
              </TableRow>
            ) : (
              data.members.map((m) => (
                <TableRow key={m.id}>
                  <TableCell className="font-medium">
                    {m.email}
                    {m.id === admin?.id ? (
                      <span className="text-muted-foreground ml-2 text-xs">(you)</span>
                    ) : null}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="capitalize">{m.role}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {m.lastLoginAt ? new Date(m.lastLoginAt).toLocaleDateString() : "Never"}
                  </TableCell>
                  <TableCell className="text-right">
                    {m.id !== admin?.id ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={removeMemberMut.isPending}
                        onClick={() => removeMemberMut.mutate(m.id)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {data?.invites.length ? (
        <>
          <div className="space-y-1">
            <h2 className="text-lg font-semibold tracking-tight">Pending invites</h2>
            <p className="text-muted-foreground text-sm">Invites that haven't been accepted yet.</p>
          </div>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Expires</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.invites.map((inv) => (
                  <TableRow key={inv.id}>
                    <TableCell className="font-medium">{inv.email}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="capitalize">{inv.role}</Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {new Date(inv.expiresAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled={deleteInviteMut.isPending}
                        onClick={() => deleteInviteMut.mutate(inv.id)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </>
      ) : null}
    </AdminShell>
  );
}
