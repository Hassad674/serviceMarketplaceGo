import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/team_repository_impl.dart';
import '../../domain/entities/team_member.dart';
import '../providers/team_provider.dart';

/// Destructive confirmation dialog for `Remove member`.
///
/// Owner-safe by design: the trailing menu in the team list hides the
/// remove action for Owner rows, but if this dialog were ever opened
/// against an Owner the backend would reject the call with a 403 and
/// the localized error message would be surfaced inline.
class RemoveMemberDialog extends ConsumerStatefulWidget {
  const RemoveMemberDialog({
    super.key,
    required this.orgId,
    required this.member,
  });

  final String orgId;
  final TeamMember member;

  static Future<void> show(
    BuildContext context, {
    required String orgId,
    required TeamMember member,
  }) {
    return showDialog<void>(
      context: context,
      builder: (_) => RemoveMemberDialog(orgId: orgId, member: member),
    );
  }

  @override
  ConsumerState<RemoveMemberDialog> createState() =>
      _RemoveMemberDialogState();
}

class _RemoveMemberDialogState extends ConsumerState<RemoveMemberDialog> {
  bool _submitting = false;
  String? _serverError;

  Future<void> _confirm() async {
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _submitting = true;
      _serverError = null;
    });
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.removeMember(
        orgId: widget.orgId,
        userId: widget.member.userId,
      );
      if (!mounted) return;
      ref.invalidate(teamMembersProvider);
      final name = widget.member.displayLabel(l10n.teamMemberFallbackName);
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.teamRemoveMemberSuccess(name)),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final apiError = ApiException.fromDioException(e);
      setState(() {
        _serverError = apiError.localizedMessage(context).isNotEmpty
            ? apiError.localizedMessage(context)
            : l10n.teamRemoveMemberFailed;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _serverError = l10n.teamRemoveMemberFailed);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>()!;
    final displayName = widget.member.displayLabel(l10n.teamMemberFallbackName);
    return AlertDialog(
      backgroundColor: colorScheme.surfaceContainerLowest,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        side: BorderSide(color: colors.border),
      ),
      icon: CircleAvatar(
        backgroundColor: colors.accentSoft,
        child: Icon(Icons.person_remove_outlined, color: colors.primaryDeep),
      ),
      title: Text(
        l10n.teamRemoveMemberDialogTitle,
        style: SoleilTextStyles.titleLarge.copyWith(
          fontSize: 20,
          color: colorScheme.onSurface,
        ),
      ),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.teamRemoveMemberConfirm(displayName),
            style: SoleilTextStyles.body.copyWith(
              fontSize: 13.5,
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          if (_serverError != null) ...[
            const SizedBox(height: 12),
            Text(
              _serverError!,
              style: SoleilTextStyles.caption.copyWith(
                color: colors.primaryDeep,
              ),
            ),
          ],
        ],
      ),
      actions: [
        TextButton(
          onPressed: _submitting ? null : () => Navigator.of(context).pop(),
          child: Text(l10n.cancel),
        ),
        FilledButton.icon(
          style: FilledButton.styleFrom(
            backgroundColor: colorScheme.primary,
            foregroundColor: colorScheme.onPrimary,
            shape: const StadiumBorder(),
          ),
          onPressed: _submitting ? null : _confirm,
          icon: _submitting
              ? SizedBox(
                  height: 16,
                  width: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: colorScheme.onPrimary,
                  ),
                )
              : const Icon(Icons.delete_outline_rounded, size: 16),
          label: Text(l10n.teamRemoveMemberConfirmButton),
        ),
      ],
    );
  }
}
