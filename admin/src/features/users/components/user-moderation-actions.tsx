import { useState } from "react"
import { Shield, ShieldOff, Ban, UserCheck } from "lucide-react"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { useSuspendUser, useUnsuspendUser, useBanUser, useUnbanUser } from "../hooks/use-users"
import { UserSuspendDialog } from "./user-suspend-dialog"
import { UserBanDialog } from "./user-ban-dialog"
import type { AdminUser } from "../types"

// The action bar on the user detail page: suspend / unsuspend / ban /
// unban, plus the two dialogs that capture reasons. Encapsulates its
// own mutations + dialog open state so the parent page stays thin.

type UserModerationActionsProps = {
  user: AdminUser
}

export function UserModerationActions({ user }: UserModerationActionsProps) {
  const [showSuspendDialog, setShowSuspendDialog] = useState(false)
  const [showBanDialog, setShowBanDialog] = useState(false)

  const suspendMutation = useSuspendUser(user.id)
  const unsuspendMutation = useUnsuspendUser(user.id)
  const banMutation = useBanUser(user.id)
  const unbanMutation = useUnbanUser(user.id)

  const hasError =
    suspendMutation.isError ||
    banMutation.isError ||
    unsuspendMutation.isError ||
    unbanMutation.isError

  return (
    <>
      <Card>
        <CardContent className="space-y-4 pt-6">
          <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            Actions
          </h3>
          <div className="flex flex-wrap gap-3">
            {user.status === "active" && (
              <>
                <Button variant="outline" onClick={() => setShowSuspendDialog(true)}>
                  <Shield className="h-4 w-4" /> Suspendre
                </Button>
                <Button variant="destructive" onClick={() => setShowBanDialog(true)}>
                  <Ban className="h-4 w-4" /> Bannir
                </Button>
              </>
            )}
            {user.status === "suspended" && (
              <Button
                variant="outline"
                onClick={() => unsuspendMutation.mutate()}
                disabled={unsuspendMutation.isPending}
              >
                <ShieldOff className="h-4 w-4" />
                {unsuspendMutation.isPending ? "En cours..." : "Lever la suspension"}
              </Button>
            )}
            {user.status === "banned" && (
              <Button
                variant="outline"
                onClick={() => unbanMutation.mutate()}
                disabled={unbanMutation.isPending}
              >
                <UserCheck className="h-4 w-4" />
                {unbanMutation.isPending ? "En cours..." : "Lever le bannissement"}
              </Button>
            )}
          </div>
          {hasError && (
            <p className="text-sm text-destructive">
              Une erreur est survenue. Veuillez r&eacute;essayer.
            </p>
          )}
        </CardContent>
      </Card>

      <UserSuspendDialog
        open={showSuspendDialog}
        onClose={() => setShowSuspendDialog(false)}
        onConfirm={(reason, expiresAt) => {
          suspendMutation.mutate(
            { reason, expires_at: expiresAt || undefined },
            { onSuccess: () => setShowSuspendDialog(false) },
          )
        }}
        isPending={suspendMutation.isPending}
      />

      <UserBanDialog
        open={showBanDialog}
        onClose={() => setShowBanDialog(false)}
        onConfirm={(reason) => {
          banMutation.mutate(
            { reason },
            { onSuccess: () => setShowBanDialog(false) },
          )
        }}
        isPending={banMutation.isPending}
      />
    </>
  )
}
