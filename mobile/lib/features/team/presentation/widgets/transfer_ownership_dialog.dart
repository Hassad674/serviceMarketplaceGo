import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/team_repository_impl.dart';
import '../../domain/entities/team_member.dart';
import '../providers/team_provider.dart';

/// Modal that lets the current Owner pick an Admin to receive
/// ownership. Eligible targets are members with `role == 'admin'`
/// excluding the current user. If no eligible Admin exists, the
/// dialog renders a guidance message instead of a form.
class TransferOwnershipDialog extends ConsumerStatefulWidget {
  const TransferOwnershipDialog({
    super.key,
    required this.orgId,
    required this.members,
  });

  final String orgId;
  final List<TeamMember> members;

  static Future<void> show(
    BuildContext context, {
    required String orgId,
    required List<TeamMember> members,
  }) {
    return showDialog<void>(
      context: context,
      builder: (_) => TransferOwnershipDialog(orgId: orgId, members: members),
    );
  }

  @override
  ConsumerState<TransferOwnershipDialog> createState() =>
      _TransferOwnershipDialogState();
}

class _TransferOwnershipDialogState
    extends ConsumerState<TransferOwnershipDialog> {
  String? _targetUserId;
  bool _submitting = false;
  String? _serverError;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final l10n = AppLocalizations.of(context)!;
    final currentUserId = ref.watch(currentUserIdProvider);
    final eligible = widget.members
        .where((m) => m.role == 'admin' && m.userId != currentUserId)
        .toList();

    return Dialog(
      backgroundColor: colorScheme.surfaceContainerLowest,
      surfaceTintColor: Colors.transparent,
      insetPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 32),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        side: BorderSide(color: colors.border),
      ),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _Header(title: l10n.teamTransferDialogTitle),
              const SizedBox(height: 12),
              _WarningBanner(text: l10n.teamTransferDialogBody),
              const SizedBox(height: 16),
              if (eligible.isEmpty)
                _EmptyEligible(text: l10n.teamTransferNoEligible)
              else
                _buildTargetList(l10n, eligible),
              if (_serverError != null) ...[
                const SizedBox(height: 12),
                Text(
                  _serverError!,
                  style: SoleilTextStyles.caption.copyWith(
                    color: colors.primaryDeep,
                  ),
                ),
              ],
              const SizedBox(height: 20),
              _buildActions(l10n, eligible.isEmpty),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildTargetList(AppLocalizations l10n, List<TeamMember> eligible) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.teamTransferTargetLabel,
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        Container(
          decoration: BoxDecoration(
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          child: Column(
            children: [
              for (final m in eligible)
                _TargetTile(
                  member: m,
                  selected: _targetUserId == m.userId,
                  enabled: !_submitting,
                  onTap: () => setState(() => _targetUserId = m.userId),
                  fallbackName: l10n.teamMemberFallbackName,
                ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildActions(AppLocalizations l10n, bool noEligible) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        TextButton(
          onPressed: _submitting ? null : () => Navigator.of(context).pop(),
          child: Text(l10n.cancel),
        ),
        const SizedBox(width: 8),
        FilledButton.icon(
          style: FilledButton.styleFrom(
            backgroundColor: colorScheme.primary,
            foregroundColor: colorScheme.onPrimary,
            shape: const StadiumBorder(),
          ),
          onPressed: (_submitting || noEligible || _targetUserId == null)
              ? null
              : _submit,
          icon: _submitting
              ? SizedBox(
                  height: 16,
                  width: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: colorScheme.onPrimary,
                  ),
                )
              : const Icon(Icons.swap_horiz_rounded, size: 16),
          label: Text(l10n.teamTransferConfirmButton),
        ),
      ],
    );
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context)!;
    final target = _targetUserId;
    if (target == null) return;
    setState(() {
      _submitting = true;
      _serverError = null;
    });
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.initiateTransfer(
        orgId: widget.orgId,
        targetUserId: target,
      );
      // Refresh the auth state so the team screen banner picks up
      // the new pending_transfer_* fields.
      await ref.read(authProvider.notifier).refreshSession();
      if (!mounted) return;
      ref.invalidate(teamMembersProvider);
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.teamTransferSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final apiError = ApiException.fromDioException(e);
      setState(() {
        _serverError = apiError.localizedMessage(context).isNotEmpty
            ? apiError.localizedMessage(context)
            : l10n.teamTransferFailed;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _serverError = l10n.teamTransferFailed);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Row(
      children: [
        Container(
          height: 40,
          width: 40,
          decoration: BoxDecoration(
            color: colors.amberSoft,
            shape: BoxShape.circle,
          ),
          alignment: Alignment.center,
          child: Icon(
            Icons.workspace_premium_outlined,
            color: colors.warning,
            size: 18,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            title,
            style: SoleilTextStyles.titleLarge.copyWith(
              fontSize: 18,
              color: colorScheme.onSurface,
            ),
          ),
        ),
        IconButton(
          tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
          icon: Icon(Icons.close_rounded, color: colorScheme.onSurfaceVariant),
          onPressed: () => Navigator.of(context).pop(),
        ),
      ],
    );
  }
}

class _WarningBanner extends StatelessWidget {
  const _WarningBanner({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colors.amberSoft,
        border: Border.all(color: colors.amberSoft),
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.warning_amber_outlined,
            color: colors.warning,
            size: 18,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              text,
              style: SoleilTextStyles.body.copyWith(
                fontSize: 13,
                fontStyle: FontStyle.italic,
                color: colors.warning,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyEligible extends StatelessWidget {
  const _EmptyEligible({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        border: Border.all(color: colors.border),
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      child: Text(
        text,
        style: SoleilTextStyles.body.copyWith(
          fontSize: 13,
          fontStyle: FontStyle.italic,
          color: colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}

class _TargetTile extends StatelessWidget {
  const _TargetTile({
    required this.member,
    required this.selected,
    required this.enabled,
    required this.onTap,
    required this.fallbackName,
  });

  final TeamMember member;
  final bool selected;
  final bool enabled;
  final VoidCallback onTap;
  final String fallbackName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final subtitle = member.title.isNotEmpty
        ? member.title
        : (member.user?.email ?? '');
    return InkWell(
      onTap: enabled ? onTap : null,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        child: Row(
          children: [
            Icon(
              selected
                  ? Icons.radio_button_checked_rounded
                  : Icons.radio_button_unchecked_rounded,
              size: 20,
              color: selected
                  ? colorScheme.primary
                  : colorScheme.onSurfaceVariant,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    member.displayLabel(fallbackName),
                    style: SoleilTextStyles.body.copyWith(
                      fontWeight: FontWeight.w600,
                      color: colorScheme.onSurface,
                    ),
                  ),
                  if (subtitle.isNotEmpty) ...[
                    const SizedBox(height: 2),
                    Text(
                      subtitle,
                      style: SoleilTextStyles.caption.copyWith(
                        color: colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
