import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../features/auth/presentation/providers/auth_provider.dart';

// ---------------------------------------------------------------------------
// Permission constants — match the backend exactly
// ---------------------------------------------------------------------------

/// Organization role permission strings as enforced by the backend.
///
/// These constants mirror the Go backend's `orgpermission` package.
/// When the backend returns 403 `permission_denied`, the user's org role
/// lacks the permission listed here.
abstract final class OrgPermission {
  // Jobs
  static const jobsView = 'jobs.view';
  static const jobsCreate = 'jobs.create';
  static const jobsEdit = 'jobs.edit';
  static const jobsDelete = 'jobs.delete';

  // Proposals
  static const proposalsView = 'proposals.view';
  static const proposalsCreate = 'proposals.create';
  static const proposalsRespond = 'proposals.respond';

  // Messaging
  static const messagingView = 'messaging.view';
  static const messagingSend = 'messaging.send';

  // Wallet
  static const walletView = 'wallet.view';
  static const walletWithdraw = 'wallet.withdraw';

  // Org profile
  static const orgProfileEdit = 'org_profile.edit';

  // Team
  static const teamView = 'team.view';
  static const teamInvite = 'team.invite';
  static const teamManage = 'team.manage';
  static const teamTransferOwnership = 'team.transfer_ownership';
  static const teamManageRolePermissions = 'team.manage_role_permissions';

  // Billing
  static const billingView = 'billing.view';
  static const billingManage = 'billing.manage';

  // Other
  static const orgDelete = 'org.delete';
  static const kycManage = 'kyc.manage';
  static const reviewsRespond = 'reviews.respond';
}

// ---------------------------------------------------------------------------
// Riverpod provider — reactive permission checks
// ---------------------------------------------------------------------------

/// Exposes the current user's org permissions as a [Set<String>].
///
/// Returns an empty set when the user has no organization context
/// (solo user or not yet loaded).
final orgPermissionsProvider = Provider<Set<String>>((ref) {
  final auth = ref.watch(authProvider);
  final org = auth.organization;
  if (org == null) return const {};

  final raw = org['permissions'];
  if (raw is List) {
    return raw.cast<String>().toSet();
  }
  return const {};
});

/// Returns `true` when the authenticated user holds [permission] in their
/// current organization context.
///
/// Usage in a widget:
/// ```dart
/// final canCreate = ref.watch(hasPermissionProvider(OrgPermission.jobsCreate));
/// ```
final hasPermissionProvider = Provider.family<bool, String>((ref, permission) {
  return ref.watch(orgPermissionsProvider).contains(permission);
});
