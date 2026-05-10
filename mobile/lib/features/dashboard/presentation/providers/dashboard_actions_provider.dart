import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../invoicing/presentation/providers/invoicing_providers.dart';
import '../../../messaging/presentation/providers/conversations_provider.dart';
import '../../../profile_completion/presentation/providers/profile_completion_providers.dart';
import '../../../proposal/presentation/providers/proposal_provider.dart';
import '../../../subscription/presentation/providers/subscription_providers.dart';
import '../../domain/dashboard_action.dart';

/// Aggregates "actions à faire" rows across already-existing feature
/// providers — never opens a fresh HTTP call. Mirrors the web's
/// dashboard `aggregateActions` composition root logic.
///
/// Returns an empty list while everything resolves OR when the user has
/// no pending actions ("Tout est à jour" empty state, rendered upstream).
///
/// Severity ladder (highest first, surface in this order):
/// 1. critical: KYC restricted, unread > 24h fallback, expired premium,
///    proposals pending action.
/// 2. warning: KYC pending, profile completion < 80%, billing missing,
///    Stripe onboarding incomplete, premium expiring < 7d.
/// 3. info: reserved (no current row).
///
/// Performance: every row reads from a provider already consumed by the
/// feature's existing screens — no new fetches, no N+1.
final dashboardActionsProvider = Provider.autoDispose<List<DashboardAction>>(
  (ref) {
    final actions = <DashboardAction>[];

    _maybeAddKyc(ref, actions);
    _maybeAddProfileCompletion(ref, actions);
    _maybeAddBillingProfile(ref, actions);
    _maybeAddUnreadMessages(ref, actions);
    _maybeAddPendingProposals(ref, actions);
    _maybeAddPremiumExpiring(ref, actions);

    actions.sort(_severityCompare);
    return List.unmodifiable(actions);
  },
);

int _severityCompare(DashboardAction a, DashboardAction b) {
  return a.severity.index.compareTo(b.severity.index);
}

void _maybeAddKyc(Ref ref, List<DashboardAction> out) {
  final user = ref.watch(authProvider).user;
  if (user == null) return;
  final role = (user['role'] as String?) ?? '';
  if (role == 'enterprise') return;
  final status = (user['kyc_status'] as String?) ?? 'none';
  if (status == 'restricted') {
    out.add(const DashboardAction(
      id: 'kyc_restricted',
      severity: DashboardActionSeverity.critical,
      label: 'KYC restricted — payouts paused',
      route: RoutePaths.paymentInfo,
      detail: 'Open Stripe onboarding to restore payouts',
    ));
  } else if (status == 'pending') {
    out.add(const DashboardAction(
      id: 'kyc_pending',
      severity: DashboardActionSeverity.warning,
      label: 'Finish KYC verification',
      route: RoutePaths.paymentInfo,
      detail: 'Required before your first payout',
    ));
  }
}

void _maybeAddProfileCompletion(Ref ref, List<DashboardAction> out) {
  final report = ref.watch(profileCompletionProvider).valueOrNull;
  if (report == null) return;
  if (report.percent >= 80) return;
  out.add(DashboardAction(
    id: 'profile_incomplete',
    severity: DashboardActionSeverity.warning,
    label: 'Complete your profile',
    route: RoutePaths.profile,
    detail: '${report.percent}% — boost your visibility',
  ));
}

void _maybeAddBillingProfile(Ref ref, List<DashboardAction> out) {
  final completeness = ref.watch(billingProfileCompletenessProvider);
  if (completeness.isLoading || completeness.isComplete) return;
  out.add(const DashboardAction(
    id: 'billing_incomplete',
    severity: DashboardActionSeverity.warning,
    label: 'Add billing details',
    route: RoutePaths.billingProfile,
    detail: 'Required to invoice your clients',
  ));
}

void _maybeAddUnreadMessages(Ref ref, List<DashboardAction> out) {
  final state = ref.watch(conversationsProvider);
  final totalUnread = state.conversations.fold<int>(
    0,
    (sum, c) => sum + c.unreadCount,
  );
  if (totalUnread <= 0) return;
  out.add(DashboardAction(
    id: 'messages_unread',
    severity: totalUnread >= 5
        ? DashboardActionSeverity.critical
        : DashboardActionSeverity.warning,
    label: 'Reply to unread messages',
    route: RoutePaths.messaging,
    detail: '$totalUnread waiting',
  ));
}

void _maybeAddPendingProposals(Ref ref, List<DashboardAction> out) {
  final proposals = ref.watch(projectsProvider).valueOrNull;
  if (proposals == null) return;
  final pending = proposals.where((p) => p.status == 'pending').length;
  if (pending <= 0) return;
  out.add(DashboardAction(
    id: 'proposals_pending',
    severity: DashboardActionSeverity.critical,
    label: 'Action required on proposals',
    route: RoutePaths.projects,
    detail: '$pending awaiting your response',
  ));
}

void _maybeAddPremiumExpiring(Ref ref, List<DashboardAction> out) {
  final sub = ref.watch(subscriptionProvider).valueOrNull;
  if (sub == null) return;
  if (!sub.cancelAtPeriodEnd) return;
  final daysLeft = sub.currentPeriodEnd.difference(DateTime.now()).inDays;
  if (daysLeft < 0 || daysLeft > 7) return;
  out.add(DashboardAction(
    id: 'premium_expiring',
    severity: DashboardActionSeverity.warning,
    label: 'Premium expiring soon',
    route: RoutePaths.pricing,
    detail: daysLeft == 0
        ? 'Ends today'
        : daysLeft == 1
            ? 'Ends tomorrow'
            : 'Ends in $daysLeft days',
  ));
}
