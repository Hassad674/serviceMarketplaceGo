import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/team_repository_impl.dart';
import '../providers/team_provider.dart';

/// Banner shown at the TOP of the team screen whenever an ownership
/// transfer is pending on the operator's organization.
///
/// Three flavours driven by who the operator is relative to the
/// transfer:
///   - target  → "You have been offered ownership" + Accept/Decline.
///   - initiator → "Transfer pending" + Cancel transfer.
///   - other  → read-only info banner.
///
/// All actions refresh the auth state via `AuthNotifier.refreshSession`
/// so the local `state.organization` map (which carries
/// pending_transfer_*, member_role, permissions) stays in sync. After
/// a successful Accept the operator is now the new Owner and their
/// session_version was bumped by the backend — we explicitly call
/// `refreshSession()` to fetch the new state from `/auth/me`.
class PendingTransferBanner extends ConsumerStatefulWidget {
  const PendingTransferBanner({super.key, required this.orgId});

  final String orgId;

  @override
  ConsumerState<PendingTransferBanner> createState() =>
      _PendingTransferBannerState();
}

class _PendingTransferBannerState extends ConsumerState<PendingTransferBanner> {
  bool _busy = false;

  @override
  Widget build(BuildContext context) {
    final transfer = ref.watch(pendingTransferProvider);
    if (transfer == null) return const SizedBox.shrink();

    final currentUserId = ref.watch(currentUserIdProvider);
    final memberRole = ref.watch(currentMemberRoleProvider);
    final isTarget =
        currentUserId != null && currentUserId == transfer.targetUserId;
    final isInitiator = !isTarget && memberRole == 'owner';

    if (isTarget) {
      return _TargetBanner(
        expiresAt: transfer.expiresAt,
        busy: _busy,
        onAccept: _accept,
        onDecline: _decline,
      );
    }
    if (isInitiator) {
      return _InitiatorBanner(
        expiresAt: transfer.expiresAt,
        busy: _busy,
        onCancel: _cancel,
      );
    }
    return _ReadOnlyBanner(expiresAt: transfer.expiresAt);
  }

  Future<void> _accept() async {
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.acceptTransfer(widget.orgId);
      // Critical: the backend bumped session_version on Accept. Mobile
      // uses a token auth mode so the response is the plain transfer
      // payload — refresh the local session via /auth/me to pick up
      // the new owner role + permissions + cleared pending_transfer_*.
      await ref.read(authProvider.notifier).refreshSession();
      ref.invalidate(teamMembersProvider);
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamPendingTransferAcceptSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      _showError(e, l10n.teamPendingTransferAcceptFailed);
    } catch (_) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamPendingTransferAcceptFailed),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _decline() async {
    final l10n = AppLocalizations.of(context)!;
    final confirmed = await _confirmDialog(
      title: l10n.teamPendingTransferDeclineDialogTitle,
      body: l10n.teamPendingTransferDeclineDialogBody,
      confirmLabel: l10n.teamPendingTransferDecline,
      destructive: true,
    );
    if (confirmed != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.declineTransfer(widget.orgId);
      await ref.read(authProvider.notifier).refreshSession();
      ref.invalidate(teamMembersProvider);
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamPendingTransferDeclineSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      _showError(e, l10n.teamPendingTransferDeclineFailed);
    } catch (_) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamPendingTransferDeclineFailed),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _cancel() async {
    final l10n = AppLocalizations.of(context)!;
    final confirmed = await _confirmDialog(
      title: l10n.teamPendingTransferCancelDialogTitle,
      body: l10n.teamPendingTransferCancelDialogBody,
      confirmLabel: l10n.teamPendingTransferCancel,
      destructive: false,
    );
    if (confirmed != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.cancelTransfer(widget.orgId);
      await ref.read(authProvider.notifier).refreshSession();
      ref.invalidate(teamMembersProvider);
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamPendingTransferCancelSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      _showError(e, l10n.teamPendingTransferCancelFailed);
    } catch (_) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamPendingTransferCancelFailed),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _showError(DioException e, String fallback) {
    if (!mounted) return;
    final apiError = ApiException.fromDioException(e);
    final localized = apiError.localizedMessage(context);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(localized.isNotEmpty ? localized : fallback),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  Future<bool?> _confirmDialog({
    required String title,
    required String body,
    required String confirmLabel,
    required bool destructive,
  }) {
    final l10n = AppLocalizations.of(context)!;
    return showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(title),
        content: Text(body),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.cancel),
          ),
          FilledButton(
            style: destructive
                ? FilledButton.styleFrom(
                    backgroundColor:
                        Theme.of(dialogContext).colorScheme.primary,
                    foregroundColor:
                        Theme.of(dialogContext).colorScheme.onPrimary,
                    shape: const StadiumBorder(),
                  )
                : null,
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(confirmLabel),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Banner variants
// ---------------------------------------------------------------------------

class _TargetBanner extends StatelessWidget {
  const _TargetBanner({
    required this.expiresAt,
    required this.busy,
    required this.onAccept,
    required this.onDecline,
  });

  final DateTime? expiresAt;
  final bool busy;
  final VoidCallback onAccept;
  final VoidCallback onDecline;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    return _Shell(
      title: l10n.teamPendingTransferTargetTitle,
      body: l10n.teamPendingTransferTargetBody,
      expiresAt: expiresAt,
      child: Row(
        children: [
          Expanded(
            child: FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: colorScheme.primary,
                foregroundColor: colorScheme.onPrimary,
                shape: const StadiumBorder(),
              ),
              onPressed: busy ? null : onAccept,
              icon: busy
                  ? SizedBox(
                      height: 16,
                      width: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: colorScheme.onPrimary,
                      ),
                    )
                  : const Icon(Icons.check_rounded, size: 16),
              label: Text(l10n.teamPendingTransferAccept),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: OutlinedButton.icon(
              style: OutlinedButton.styleFrom(
                foregroundColor: colorScheme.onSurface,
                side: BorderSide(color: colors.borderStrong),
                shape: const StadiumBorder(),
              ),
              onPressed: busy ? null : onDecline,
              icon: const Icon(Icons.close_rounded, size: 16),
              label: Text(l10n.teamPendingTransferDecline),
            ),
          ),
        ],
      ),
    );
  }
}

class _InitiatorBanner extends StatelessWidget {
  const _InitiatorBanner({
    required this.expiresAt,
    required this.busy,
    required this.onCancel,
  });

  final DateTime? expiresAt;
  final bool busy;
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    return _Shell(
      title: l10n.teamPendingTransferInitiatorTitle,
      body: l10n.teamPendingTransferInitiatorBody,
      expiresAt: expiresAt,
      child: Align(
        alignment: Alignment.centerRight,
        child: OutlinedButton.icon(
          style: OutlinedButton.styleFrom(
            foregroundColor: colorScheme.onSurface,
            side: BorderSide(color: colors.borderStrong),
            shape: const StadiumBorder(),
          ),
          onPressed: busy ? null : onCancel,
          icon: busy
              ? SizedBox(
                  height: 14,
                  width: 14,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: colorScheme.primary,
                  ),
                )
              : const Icon(Icons.close_rounded, size: 14),
          label: Text(l10n.teamPendingTransferCancel),
        ),
      ),
    );
  }
}

class _ReadOnlyBanner extends StatelessWidget {
  const _ReadOnlyBanner({required this.expiresAt});

  final DateTime? expiresAt;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return _Shell(
      title: l10n.teamPendingTransferReadOnlyTitle,
      body: l10n.teamPendingTransferReadOnlyBody,
      expiresAt: expiresAt,
      child: const SizedBox.shrink(),
    );
  }
}

class _Shell extends StatelessWidget {
  const _Shell({
    required this.title,
    required this.body,
    required this.expiresAt,
    required this.child,
  });

  final String title;
  final String body;
  final DateTime? expiresAt;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final formattedExpiry = expiresAt != null
        ? '${expiresAt!.year.toString().padLeft(4, '0')}-${expiresAt!.month.toString().padLeft(2, '0')}-${expiresAt!.day.toString().padLeft(2, '0')}'
        : null;
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            colors.amberSoft,
            colors.accentSoft,
          ],
        ),
        border: Border.all(color: colors.accentSoft),
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: colorScheme.surfaceContainerLowest,
                  shape: BoxShape.circle,
                ),
                alignment: Alignment.center,
                child: Icon(
                  Icons.workspace_premium_outlined,
                  color: colors.warning,
                  size: 20,
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  title,
                  style: SoleilTextStyles.titleLarge.copyWith(
                    fontSize: 17,
                    color: colorScheme.onSurface,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(
            body,
            style: SoleilTextStyles.body.copyWith(
              fontSize: 13,
              fontStyle: FontStyle.italic,
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          if (formattedExpiry != null) ...[
            const SizedBox(height: 4),
            Text(
              l10n.teamPendingTransferExpiresOn(formattedExpiry),
              style: SoleilTextStyles.mono.copyWith(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                letterSpacing: 0.5,
                color: colors.subtleForeground,
              ),
            ),
          ],
          if (child is! SizedBox) ...[
            const SizedBox(height: 14),
            child,
          ],
        ],
      ),
    );
  }
}
