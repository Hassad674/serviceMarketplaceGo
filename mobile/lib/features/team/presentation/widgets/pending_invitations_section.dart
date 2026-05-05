import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/team_repository_impl.dart';
import '../../domain/entities/pending_invitation.dart';
import '../providers/team_provider.dart';

/// Section displayed below the members list on the team screen.
/// Mirrors the web `TeamInvitationsList` (R20 phase 2):
///   - shows pending invitations with sent/expires metadata;
///   - resend + cancel actions per row;
///   - hidden when the operator lacks `team.invite`.
///
/// The team screen is responsible for the gating — this widget
/// assumes the section must render when it is included in the tree.
class PendingInvitationsSection extends ConsumerWidget {
  const PendingInvitationsSection({super.key, required this.orgId});

  final String orgId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final invitationsAsync = ref.watch(pendingInvitationsProvider);

    return invitationsAsync.when(
      data: (invitations) => _SectionShell(
        title: l10n.teamInvitationsCountLabel(invitations.length),
        child: invitations.isEmpty
            ? _EmptyState(appColors: appColors)
            : Column(
                children: [
                  for (final inv in invitations)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: _InvitationTile(
                        orgId: orgId,
                        invitation: inv,
                      ),
                    ),
                ],
              ),
      ),
      loading: () => _SectionShell(
        title: l10n.teamInvitationsSection,
        child: const _LoadingSkeleton(),
      ),
      error: (_, __) => _SectionShell(
        title: l10n.teamInvitationsSection,
        child: _ErrorState(appColors: appColors),
      ),
    );
  }
}

class _SectionShell extends StatelessWidget {
  const _SectionShell({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 8),
        child,
      ],
    );
  }
}

class _InvitationTile extends ConsumerStatefulWidget {
  const _InvitationTile({required this.orgId, required this.invitation});

  final String orgId;
  final PendingInvitation invitation;

  @override
  ConsumerState<_InvitationTile> createState() => _InvitationTileState();
}

class _InvitationTileState extends ConsumerState<_InvitationTile> {
  bool _busy = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final inv = widget.invitation;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        border: Border.all(
          color: appColors?.accentSoft ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Row(
        children: [
          const _MailAvatar(),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  inv.displayName(),
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  inv.email,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: appColors?.mutedForeground,
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 4),
                _MetaLine(invitation: inv),
              ],
            ),
          ),
          const SizedBox(width: 8),
          _RoleChip(role: inv.role),
          const SizedBox(width: 4),
          IconButton(
            tooltip: l10n.teamInvitationResendTooltip,
            icon: _busy
                ? const SizedBox(
                    height: 16,
                    width: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.refresh, size: 18),
            onPressed: _busy ? null : _resend,
          ),
          IconButton(
            tooltip: l10n.teamInvitationCancelTooltip,
            icon: Icon(
              Icons.delete_outline_rounded,
              size: 18,
              color: theme.colorScheme.error,
            ),
            onPressed: _busy ? null : _cancel,
          ),
        ],
      ),
    );
  }

  Future<void> _resend() async {
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.resendInvitation(
        orgId: widget.orgId,
        invitationId: widget.invitation.id,
      );
      ref.invalidate(pendingInvitationsProvider);
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamInvitationResendSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final apiError = ApiException.fromDioException(e);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            apiError.localizedMessage(context).isNotEmpty
                ? apiError.localizedMessage(context)
                : l10n.teamInvitationResendFailed,
          ),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } catch (_) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamInvitationResendFailed),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _cancel() async {
    final l10n = AppLocalizations.of(context)!;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(l10n.teamInvitationCancelDialogTitle),
        content: Text(
          l10n.teamInvitationCancelDialogBody(widget.invitation.email),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.teamInvitationCancelKeep),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(dialogContext).colorScheme.primary,
              foregroundColor: Theme.of(dialogContext).colorScheme.onPrimary,
              shape: const StadiumBorder(),
            ),
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(l10n.teamInvitationCancelConfirm),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.cancelInvitation(
        orgId: widget.orgId,
        invitationId: widget.invitation.id,
      );
      ref.invalidate(pendingInvitationsProvider);
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamInvitationCancelSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final apiError = ApiException.fromDioException(e);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            apiError.localizedMessage(context).isNotEmpty
                ? apiError.localizedMessage(context)
                : l10n.teamInvitationCancelFailed,
          ),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } catch (_) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(l10n.teamInvitationCancelFailed),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

class _MetaLine extends StatelessWidget {
  const _MetaLine({required this.invitation});

  final PendingInvitation invitation;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final now = DateTime.now();
    final sentDays = now.difference(invitation.sentAt).inDays;
    final expiresIn = invitation.expiresAt.difference(now).inDays;
    final sentLabel = sentDays <= 0
        ? l10n.teamInvitationSentToday
        : l10n.teamInvitationSentAgo(sentDays);
    final expiresLabel = expiresIn < 0
        ? l10n.teamInvitationExpired
        : l10n.teamInvitationExpiresIn(expiresIn);
    return Text(
      '$sentLabel · $expiresLabel',
      style: theme.textTheme.bodySmall?.copyWith(
        color: appColors?.mutedForeground,
      ),
      maxLines: 1,
      overflow: TextOverflow.ellipsis,
    );
  }
}

class _RoleChip extends StatelessWidget {
  const _RoleChip({required this.role});

  final String role;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    final label = switch (role) {
      'admin' => l10n.teamRoleAdmin,
      'member' => l10n.teamRoleMember,
      'viewer' => l10n.teamRoleViewer,
      _ => role,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        label,
        style: SoleilTextStyles.mono.copyWith(
          color: colorScheme.onSurfaceVariant,
          fontSize: 10,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}

class _MailAvatar extends StatelessWidget {
  const _MailAvatar();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Container(
      height: 44,
      width: 44,
      decoration: BoxDecoration(
        color: colors.accentSoft,
        shape: BoxShape.circle,
      ),
      alignment: Alignment.center,
      child: Icon(
        Icons.mail_outline_rounded,
        color: colorScheme.primary,
        size: 18,
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.appColors});

  final AppColors? appColors;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
          style: BorderStyle.solid,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      alignment: Alignment.center,
      child: Text(
        l10n.teamInvitationsEmpty,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: appColors?.mutedForeground,
        ),
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.appColors});

  final AppColors? appColors;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      alignment: Alignment.center,
      child: Text(
        l10n.teamInvitationsLoadFailed,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: appColors?.mutedForeground,
        ),
      ),
    );
  }
}

class _LoadingSkeleton extends StatelessWidget {
  const _LoadingSkeleton();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    return Column(
      children: List.generate(
        2,
        (_) => Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Container(
            height: 72,
            decoration: BoxDecoration(
              color: colors.muted,
              borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            ),
          ),
        ),
      ),
    );
  }
}
