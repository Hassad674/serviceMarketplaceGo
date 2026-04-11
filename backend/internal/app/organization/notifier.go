package organization

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	notificationdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// teamNotifier dispatches org_* notifications via the NotificationSender
// port. All helpers are best-effort: a nil sender is a no-op, and any
// send error is swallowed with a structured log so the main org action
// can return success even if the notification backend is down.
//
// Both InvitationService and MembershipService embed this pattern —
// they hold a `notifications service.NotificationSender` field and call
// these helpers with their own sender.

// dispatch is the single call site that touches the NotificationSender
// port. Every helper below funnels through it so behaviour stays
// consistent (nil-check + swallow-error + log).
func dispatch(
	ctx context.Context,
	sender service.NotificationSender,
	userID uuid.UUID,
	nType notificationdomain.NotificationType,
	title, body string,
	data json.RawMessage,
) {
	if sender == nil {
		return
	}
	if userID == uuid.Nil {
		return
	}
	err := sender.Send(ctx, service.NotificationInput{
		UserID: userID,
		Type:   string(nType),
		Title:  title,
		Body:   body,
		Data:   data,
	})
	if err != nil {
		slog.Error("team notification dispatch failed",
			"type", string(nType),
			"recipient", userID.String(),
			"error", err.Error(),
		)
	}
}

// orgLabel is the human-readable name we embed in notification copy.
// V1 has no explicit org.name column — we fall back to the org type
// ("Agency" / "Enterprise") which is always populated. When the team
// V2 adds a name field, swap this for `org.Name`.
func orgLabel(org *organization.Organization) string {
	if org == nil {
		return "your organization"
	}
	switch org.Type {
	case organization.OrgTypeAgency:
		return "your agency"
	case organization.OrgTypeEnterprise:
		return "your enterprise"
	default:
		return "your organization"
	}
}

// actorIDString returns a safe string form of the actor's user id.
// Admin force overrides pass nil actor (see admin_overrides.go) —
// this helper lets notifier helpers blindly include actor_id in
// their payload without dereferencing a nil pointer.
func actorIDString(u *user.User) string {
	if u == nil {
		return ""
	}
	return u.ID.String()
}

// actorDisplayName returns a friendly name for the user who performed
// the action. Falls back to "Someone" if the lookup fails so the notif
// copy stays intelligible even when the user repository is flaky.
// Shared by both services — both hold a user repository.
func actorDisplayName(u *user.User) string {
	if u == nil {
		return "Someone"
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	full := u.FirstName
	if u.LastName != "" {
		if full != "" {
			full += " "
		}
		full += u.LastName
	}
	if full == "" {
		return "Someone"
	}
	return full
}

// marshalData serialises a map into a json.RawMessage for the
// notification payload. Never returns an error and never emits JSON
// null — on a nil map or serialisation failure we fall back to an
// empty object so downstream consumers can always assume they get an
// object shape they can read from.
func marshalData(m map[string]any) json.RawMessage {
	if m == nil {
		return json.RawMessage(`{}`)
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

// ---------------------------------------------------------------------------
// Invitation events
// ---------------------------------------------------------------------------

// notifyInvitationAccepted tells the original inviter (usually an
// Owner or Admin) that someone accepted their invitation. The new
// member's name is resolved before the call so we only need one
// user lookup per action.
func notifyInvitationAccepted(
	ctx context.Context,
	sender service.NotificationSender,
	recipientUserID uuid.UUID, // the inviter
	newMember *user.User,
	org *organization.Organization,
	invitationID uuid.UUID,
) {
	name := actorDisplayName(newMember)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"new_member_id":     newMember.ID.String(),
		"new_member_name":   name,
		"invitation_id":     invitationID.String(),
	})
	dispatch(ctx, sender, recipientUserID,
		notificationdomain.TypeOrgInvitationAccepted,
		"Invitation accepted",
		name+" accepted your invitation to join "+orgLabel(org),
		data,
	)
}

// ---------------------------------------------------------------------------
// Membership events
// ---------------------------------------------------------------------------

// notifyMemberRoleChanged tells the target their role was updated.
// Used for both promote and demote — the oldRole/newRole fields in
// the payload let the client decide how to phrase it in the UI.
func notifyMemberRoleChanged(
	ctx context.Context,
	sender service.NotificationSender,
	targetUserID uuid.UUID,
	actor *user.User,
	org *organization.Organization,
	oldRole, newRole organization.Role,
) {
	if org == nil {
		return
	}
	actorName := actorDisplayName(actor)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"actor_id":          actorIDString(actor),
		"actor_name":        actorName,
		"old_role":          string(oldRole),
		"new_role":          string(newRole),
	})
	dispatch(ctx, sender, targetUserID,
		notificationdomain.TypeOrgMemberRoleChanged,
		"Your role was updated",
		actorName+" changed your role to "+string(newRole)+" in "+orgLabel(org),
		data,
	)
}

// notifyMemberTitleChanged reuses the role-changed type with a
// title_only flag in the payload, so the client can render a softer
// copy without us inventing a new type.
func notifyMemberTitleChanged(
	ctx context.Context,
	sender service.NotificationSender,
	targetUserID uuid.UUID,
	actor *user.User,
	org *organization.Organization,
	newTitle string,
) {
	if org == nil {
		return
	}
	actorName := actorDisplayName(actor)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"actor_id":          actorIDString(actor),
		"actor_name":        actorName,
		"new_title":         newTitle,
		"title_only":        true,
	})
	dispatch(ctx, sender, targetUserID,
		notificationdomain.TypeOrgMemberRoleChanged,
		"Your title was updated",
		actorName+" updated your title in "+orgLabel(org),
		data,
	)
}

// notifyMemberRemoved tells a user they were evicted from an org.
// The user row may be deleted right after this (operators get purged)
// but the in-app notification is dispatched BEFORE the delete so the
// notification queue always has a valid user to write to.
func notifyMemberRemoved(
	ctx context.Context,
	sender service.NotificationSender,
	targetUserID uuid.UUID,
	actor *user.User,
	org *organization.Organization,
) {
	if org == nil {
		return
	}
	actorName := actorDisplayName(actor)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"actor_id":          actorIDString(actor),
		"actor_name":        actorName,
	})
	dispatch(ctx, sender, targetUserID,
		notificationdomain.TypeOrgMemberRemoved,
		"You were removed",
		actorName+" removed you from "+orgLabel(org),
		data,
	)
}

// notifyMemberLeft tells the Owner that one of their members walked
// out on their own. Only the Owner is notified — admins are not
// broadcast to in V1 to avoid notification noise on larger teams.
func notifyMemberLeft(
	ctx context.Context,
	sender service.NotificationSender,
	ownerUserID uuid.UUID,
	leaver *user.User,
	org *organization.Organization,
) {
	leaverName := actorDisplayName(leaver)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"leaver_id":         leaver.ID.String(),
		"leaver_name":       leaverName,
	})
	dispatch(ctx, sender, ownerUserID,
		notificationdomain.TypeOrgMemberLeft,
		"A member left",
		leaverName+" left "+orgLabel(org),
		data,
	)
}

// ---------------------------------------------------------------------------
// Ownership transfer events
// ---------------------------------------------------------------------------

func notifyTransferInitiated(
	ctx context.Context,
	sender service.NotificationSender,
	targetUserID uuid.UUID,
	currentOwner *user.User,
	org *organization.Organization,
) {
	ownerName := actorDisplayName(currentOwner)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"current_owner_id":  currentOwner.ID.String(),
		"current_owner":     ownerName,
	})
	dispatch(ctx, sender, targetUserID,
		notificationdomain.TypeOrgTransferInitiated,
		"Ownership transfer requested",
		ownerName+" wants to transfer ownership of "+orgLabel(org)+" to you",
		data,
	)
}

func notifyTransferCancelled(
	ctx context.Context,
	sender service.NotificationSender,
	targetUserID uuid.UUID,
	currentOwner *user.User,
	org *organization.Organization,
) {
	ownerName := actorDisplayName(currentOwner)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"current_owner_id":  currentOwner.ID.String(),
		"current_owner":     ownerName,
	})
	dispatch(ctx, sender, targetUserID,
		notificationdomain.TypeOrgTransferCancelled,
		"Ownership transfer cancelled",
		ownerName+" cancelled the ownership transfer of "+orgLabel(org),
		data,
	)
}

func notifyTransferDeclined(
	ctx context.Context,
	sender service.NotificationSender,
	currentOwnerID uuid.UUID,
	target *user.User,
	org *organization.Organization,
) {
	targetName := actorDisplayName(target)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"target_id":         target.ID.String(),
		"target_name":       targetName,
	})
	dispatch(ctx, sender, currentOwnerID,
		notificationdomain.TypeOrgTransferDeclined,
		"Ownership transfer declined",
		targetName+" declined the ownership transfer of "+orgLabel(org),
		data,
	)
}

func notifyTransferAccepted(
	ctx context.Context,
	sender service.NotificationSender,
	oldOwnerID uuid.UUID,
	newOwner *user.User,
	org *organization.Organization,
) {
	newOwnerName := actorDisplayName(newOwner)
	data := marshalData(map[string]any{
		"organization_id":   org.ID.String(),
		"organization_type": string(org.Type),
		"new_owner_id":      newOwner.ID.String(),
		"new_owner_name":    newOwnerName,
	})
	dispatch(ctx, sender, oldOwnerID,
		notificationdomain.TypeOrgTransferAccepted,
		"Ownership transferred",
		newOwnerName+" accepted ownership of "+orgLabel(org)+". You are now an Admin.",
		data,
	)
}
