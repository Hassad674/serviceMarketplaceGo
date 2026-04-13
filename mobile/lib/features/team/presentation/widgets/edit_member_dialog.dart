import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/team_repository_impl.dart';
import '../../domain/entities/team_member.dart';
import '../providers/team_provider.dart';

/// Modal form to edit a team member's role and/or title.
///
/// Mirrors the web `EditMemberModal` (R20 phase 1):
///   - role dropdown excluding "owner" (promotion goes through the
///     transfer ownership flow);
///   - title text field (optional, max 100 chars);
///   - sends only the fields that actually changed;
///   - role and title travel through dedicated repository methods so
///     the audit trail stays accurate (the backend handler accepts
///     both fields in a single PATCH but updates them sequentially).
class EditMemberDialog extends ConsumerStatefulWidget {
  const EditMemberDialog({
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
      builder: (_) => EditMemberDialog(orgId: orgId, member: member),
    );
  }

  @override
  ConsumerState<EditMemberDialog> createState() => _EditMemberDialogState();
}

class _EditMemberDialogState extends ConsumerState<EditMemberDialog> {
  late String _role;
  late TextEditingController _titleController;
  bool _submitting = false;
  String? _serverError;

  @override
  void initState() {
    super.initState();
    // The dropdown excludes "owner". Owners cannot be edited from
    // this dialog — the trailing menu hides the action — but if the
    // dialog were opened anyway we fall back to "admin" to keep the
    // form valid.
    _role = widget.member.role == 'owner' ? 'admin' : widget.member.role;
    _titleController = TextEditingController(text: widget.member.title);
  }

  @override
  void dispose() {
    _titleController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context)!;
    final newTitle = _titleController.text.trim();
    final roleChanged = _role != widget.member.role;
    final titleChanged = newTitle != widget.member.title;
    if (!roleChanged && !titleChanged) {
      setState(() => _serverError = l10n.teamEditMemberNoChanges);
      return;
    }
    setState(() {
      _submitting = true;
      _serverError = null;
    });
    try {
      final repo = ref.read(teamRepositoryProvider);
      if (roleChanged) {
        await repo.updateMemberRole(
          orgId: widget.orgId,
          userId: widget.member.userId,
          role: _role,
        );
      }
      if (titleChanged) {
        await repo.updateMemberTitle(
          orgId: widget.orgId,
          userId: widget.member.userId,
          title: newTitle,
        );
      }
      if (!mounted) return;
      ref.invalidate(teamMembersProvider);
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.teamEditMemberSuccess),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final apiError = ApiException.fromDioException(e);
      setState(() {
        _serverError = apiError.localizedMessage(context).isNotEmpty
            ? apiError.localizedMessage(context)
            : l10n.teamEditMemberFailed;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _serverError = l10n.teamEditMemberFailed);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final displayName = widget.member.displayLabel(l10n.teamMemberFallbackName);
    return Dialog(
      insetPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 32),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _Header(title: l10n.teamEditMemberDialogTitle(displayName)),
              const SizedBox(height: 16),
              _buildRoleField(l10n),
              const SizedBox(height: 12),
              _buildTitleField(l10n),
              if (_serverError != null) ...[
                const SizedBox(height: 12),
                _ErrorBanner(message: _serverError!),
              ],
              const SizedBox(height: 20),
              _buildActions(l10n),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildRoleField(AppLocalizations l10n) {
    return DropdownButtonFormField<String>(
      initialValue: _role,
      decoration: InputDecoration(
        labelText: l10n.teamEditMemberRoleLabel,
        border: const OutlineInputBorder(),
      ),
      items: [
        DropdownMenuItem(value: 'admin', child: Text(l10n.teamRoleAdmin)),
        DropdownMenuItem(value: 'member', child: Text(l10n.teamRoleMember)),
        DropdownMenuItem(value: 'viewer', child: Text(l10n.teamRoleViewer)),
      ],
      onChanged: _submitting
          ? null
          : (value) {
              if (value != null) setState(() => _role = value);
            },
    );
  }

  Widget _buildTitleField(AppLocalizations l10n) {
    return TextField(
      controller: _titleController,
      enabled: !_submitting,
      maxLength: 100,
      decoration: InputDecoration(
        labelText: l10n.teamEditMemberTitleLabel,
        hintText: l10n.teamEditMemberTitleHint,
        border: const OutlineInputBorder(),
        counterText: '',
      ),
    );
  }

  Widget _buildActions(AppLocalizations l10n) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        TextButton(
          onPressed: _submitting ? null : () => Navigator.of(context).pop(),
          child: Text(l10n.cancel),
        ),
        const SizedBox(width: 8),
        FilledButton.icon(
          onPressed: _submitting ? null : _submit,
          icon: _submitting
              ? const SizedBox(
                  height: 16,
                  width: 16,
                  child: CircularProgressIndicator(
                    strokeWidth: 2,
                    color: Colors.white,
                  ),
                )
              : const Icon(Icons.check, size: 18),
          label: Text(l10n.teamEditMemberSave),
        ),
      ],
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Container(
          height: 36,
          width: 36,
          decoration: const BoxDecoration(
            color: Color(0xFFEDE9FE), // violet-100
            shape: BoxShape.circle,
          ),
          alignment: Alignment.center,
          child: const Icon(
            Icons.edit_outlined,
            color: Color(0xFF6D28D9), // violet-700
            size: 18,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            title,
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
        IconButton(
          tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
          icon: const Icon(Icons.close),
          onPressed: () => Navigator.of(context).pop(),
        ),
      ],
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0xFFFEF2F2),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: const Color(0xFFFCA5A5)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(
            Icons.error_outline,
            color: Color(0xFFDC2626),
            size: 18,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(
                color: Color(0xFFB91C1C),
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
