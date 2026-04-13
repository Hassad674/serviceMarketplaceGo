import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/team_repository_impl.dart';

/// Strong-confirmation dialog for self-leaving an organization.
///
/// Mirrors the web `LeaveOrgDialog` (R20 phase 3) with an extra
/// safety net: the operator must type the localized confirmation
/// keyword (e.g. "LEAVE" / "QUITTER") before the destructive button
/// becomes enabled. On success the auth state is cleared (R16: the
/// backend invalidates the session of an operator that leaves their
/// org) and the user is sent back to the login screen.
class LeaveOrganizationDialog extends ConsumerStatefulWidget {
  const LeaveOrganizationDialog({super.key, required this.orgId});

  final String orgId;

  static Future<void> show(BuildContext context, String orgId) {
    return showDialog<void>(
      context: context,
      builder: (_) => LeaveOrganizationDialog(orgId: orgId),
    );
  }

  @override
  ConsumerState<LeaveOrganizationDialog> createState() =>
      _LeaveOrganizationDialogState();
}

class _LeaveOrganizationDialogState
    extends ConsumerState<LeaveOrganizationDialog> {
  final _confirmController = TextEditingController();
  bool _submitting = false;
  String? _serverError;

  @override
  void initState() {
    super.initState();
    _confirmController.addListener(() => setState(() {}));
  }

  @override
  void dispose() {
    _confirmController.dispose();
    super.dispose();
  }

  Future<void> _confirm() async {
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _submitting = true;
      _serverError = null;
    });
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.leaveOrganization(widget.orgId);
      if (!mounted) return;
      // The backend invalidated the operator's session as part of the
      // leave call (R16). Clear local credentials so the auth guard
      // bounces the user to /login on the next frame.
      await ref.read(authProvider.notifier).logout();
      if (!mounted) return;
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.teamLeaveSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final apiError = ApiException.fromDioException(e);
      setState(() {
        _serverError = apiError.localizedMessage(context).isNotEmpty
            ? apiError.localizedMessage(context)
            : l10n.teamLeaveFailed;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _serverError = l10n.teamLeaveFailed);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final keyword = l10n.teamLeaveConfirmKeyword;
    final canSubmit = _confirmController.text.trim().toUpperCase() == keyword;

    return AlertDialog(
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      icon: const CircleAvatar(
        backgroundColor: Color(0xFFFEE2E2),
        child: Icon(Icons.logout, color: Color(0xFFDC2626)),
      ),
      title: Text(l10n.teamLeaveDialogTitle),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.teamLeaveDialogBody,
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _confirmController,
            enabled: !_submitting,
            textCapitalization: TextCapitalization.characters,
            decoration: InputDecoration(
              labelText: l10n.teamLeaveConfirmHint,
              border: const OutlineInputBorder(),
            ),
          ),
          if (_serverError != null) ...[
            const SizedBox(height: 12),
            Text(
              _serverError!,
              style: const TextStyle(
                color: Color(0xFFB91C1C),
                fontSize: 13,
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
            backgroundColor: const Color(0xFFDC2626),
            foregroundColor: Colors.white,
          ),
          onPressed: (!canSubmit || _submitting) ? null : _confirm,
          icon: _submitting
              ? const SizedBox(
                  height: 16,
                  width: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: Colors.white,
                  ),
                )
              : const Icon(Icons.logout, size: 18),
          label: Text(l10n.teamLeaveConfirmButton),
        ),
      ],
    );
  }
}
