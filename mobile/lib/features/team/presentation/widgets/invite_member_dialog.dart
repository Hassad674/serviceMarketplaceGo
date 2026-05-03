import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/team_repository_impl.dart';
import '../providers/team_provider.dart';
import '../../../../core/theme/app_palette.dart';

/// Modal form used to invite a new teammate. Permission gating is
/// upstream (the caller decides whether to show it) — this widget
/// focuses on validation, submission, and error surfacing.
///
/// The backend requires email + first_name + last_name + role. The
/// title is optional. After a successful call, the parent list is
/// invalidated so pull-to-refresh is not required.
class InviteMemberDialog extends ConsumerStatefulWidget {
  const InviteMemberDialog({super.key, required this.orgId});

  final String orgId;

  static Future<void> show(BuildContext context, String orgId) {
    return showDialog<void>(
      context: context,
      builder: (_) => InviteMemberDialog(orgId: orgId),
    );
  }

  @override
  ConsumerState<InviteMemberDialog> createState() => _InviteMemberDialogState();
}

class _InviteMemberDialogState extends ConsumerState<InviteMemberDialog> {
  static final RegExp _emailRegExp = RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$');

  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _firstNameController = TextEditingController();
  final _lastNameController = TextEditingController();
  final _titleController = TextEditingController();
  String _role = 'member';
  bool _submitting = false;
  String? _serverError;

  @override
  void dispose() {
    _emailController.dispose();
    _firstNameController.dispose();
    _lastNameController.dispose();
    _titleController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context)!;
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _submitting = true;
      _serverError = null;
    });
    try {
      final repo = ref.read(teamRepositoryProvider);
      await repo.inviteMember(
        orgId: widget.orgId,
        email: _emailController.text.trim(),
        firstName: _firstNameController.text.trim(),
        lastName: _lastNameController.text.trim(),
        role: _role,
        title: _titleController.text.trim(),
      );
      if (!mounted) return;
      ref.invalidate(teamMembersProvider);
      final email = _emailController.text.trim();
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.teamInviteSuccess(email)),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      final apiError = ApiException.fromDioException(e);
      setState(() {
        _serverError = apiError.message.isNotEmpty
            ? apiError.message
            : l10n.teamInviteFailed;
      });
    } catch (_) {
      setState(() => _serverError = l10n.teamInviteFailed);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    return Dialog(
      insetPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 32),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Form(
            key: _formKey,
            autovalidateMode: AutovalidateMode.onUserInteraction,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _DialogHeader(title: l10n.teamInviteDialogTitle),
                const SizedBox(height: 8),
                Text(
                  l10n.teamInviteDialogDescription,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 16),
                SingleChildScrollView(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      _buildEmailField(l10n),
                      const SizedBox(height: 12),
                      _buildNameRow(l10n),
                      const SizedBox(height: 12),
                      _buildTitleField(l10n),
                      const SizedBox(height: 12),
                      _buildRoleField(l10n),
                      if (_serverError != null) ...[
                        const SizedBox(height: 12),
                        _ServerErrorBanner(message: _serverError!),
                      ],
                    ],
                  ),
                ),
                const SizedBox(height: 20),
                _buildActions(l10n),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildEmailField(AppLocalizations l10n) {
    return TextFormField(
      controller: _emailController,
      keyboardType: TextInputType.emailAddress,
      autofillHints: const [AutofillHints.email],
      enabled: !_submitting,
      decoration: InputDecoration(
        labelText: l10n.teamInviteEmailLabel,
        hintText: l10n.teamInviteEmailHint,
        border: const OutlineInputBorder(),
      ),
      validator: (raw) {
        final value = (raw ?? '').trim();
        if (value.isEmpty) return l10n.teamInviteEmailRequired;
        if (!_emailRegExp.hasMatch(value)) {
          return l10n.teamInviteEmailInvalid;
        }
        return null;
      },
    );
  }

  Widget _buildNameRow(AppLocalizations l10n) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: TextFormField(
            controller: _firstNameController,
            enabled: !_submitting,
            textCapitalization: TextCapitalization.words,
            decoration: InputDecoration(
              labelText: l10n.teamInviteFirstNameLabel,
              border: const OutlineInputBorder(),
            ),
            validator: (raw) {
              if ((raw ?? '').trim().isEmpty) {
                return l10n.teamInviteFirstNameRequired;
              }
              return null;
            },
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: TextFormField(
            controller: _lastNameController,
            enabled: !_submitting,
            textCapitalization: TextCapitalization.words,
            decoration: InputDecoration(
              labelText: l10n.teamInviteLastNameLabel,
              border: const OutlineInputBorder(),
            ),
            validator: (raw) {
              if ((raw ?? '').trim().isEmpty) {
                return l10n.teamInviteLastNameRequired;
              }
              return null;
            },
          ),
        ),
      ],
    );
  }

  Widget _buildTitleField(AppLocalizations l10n) {
    return TextFormField(
      controller: _titleController,
      enabled: !_submitting,
      maxLength: 100,
      decoration: InputDecoration(
        labelText: l10n.teamInviteTitleLabel,
        hintText: l10n.teamInviteTitleHint,
        border: const OutlineInputBorder(),
        counterText: '',
      ),
    );
  }

  Widget _buildRoleField(AppLocalizations l10n) {
    return DropdownButtonFormField<String>(
      initialValue: _role,
      decoration: InputDecoration(
        labelText: l10n.teamInviteRoleLabel,
        border: const OutlineInputBorder(),
        helperText: l10n.teamInviteRoleHelp,
        helperMaxLines: 2,
      ),
      items: [
        DropdownMenuItem(
          value: 'admin',
          child: Text(l10n.teamInviteRoleAdmin),
        ),
        DropdownMenuItem(
          value: 'member',
          child: Text(l10n.teamInviteRoleMember),
        ),
        DropdownMenuItem(
          value: 'viewer',
          child: Text(l10n.teamInviteRoleViewer),
        ),
      ],
      onChanged: _submitting
          ? null
          : (value) {
              if (value != null) setState(() => _role = value);
            },
    );
  }

  Widget _buildActions(AppLocalizations l10n) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: [
        TextButton(
          onPressed: _submitting ? null : () => Navigator.of(context).pop(),
          child: Text(l10n.teamInviteCancelButton),
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
              : const Icon(Icons.send_outlined, size: 18),
          label: Text(l10n.teamInviteSendButton),
        ),
      ],
    );
  }
}

class _DialogHeader extends StatelessWidget {
  const _DialogHeader({required this.title});

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
            color: AppPalette.rose100,
            shape: BoxShape.circle,
          ),
          alignment: Alignment.center,
          child: const Icon(
            Icons.mail_outline,
            color: AppPalette.rose600,
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

class _ServerErrorBanner extends StatelessWidget {
  const _ServerErrorBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppPalette.red50,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: AppPalette.red300),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(
            Icons.error_outline,
            color: AppPalette.red600,
            size: 18,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(
                color: AppPalette.red700,
                fontSize: 13,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
