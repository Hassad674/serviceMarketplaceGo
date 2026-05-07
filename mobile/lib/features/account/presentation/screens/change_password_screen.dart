import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/account_repository_impl.dart';
import '../../domain/entities/account_failure.dart';

/// ChangePasswordScreen — mobile counterpart of `PasswordSettings` (web).
///
/// Mirrors the web complexity rules locally so we don't waste a round
/// trip on obvious failures: ≥10 chars, at least one upper, lower, digit
/// and special character, plus confirm == new.
///
/// On success the backend bumps the session version → we run the
/// standard logout flow and route back to `/login` with a SnackBar
/// inviting the user to reconnect with the new password.
class ChangePasswordScreen extends ConsumerStatefulWidget {
  const ChangePasswordScreen({super.key});

  @override
  ConsumerState<ChangePasswordScreen> createState() =>
      _ChangePasswordScreenState();
}

class _ChangePasswordScreenState extends ConsumerState<ChangePasswordScreen> {
  final _formKey = GlobalKey<FormState>();
  final _currentPasswordCtrl = TextEditingController();
  final _newPasswordCtrl = TextEditingController();
  final _confirmPasswordCtrl = TextEditingController();
  bool _obscureCurrent = true;
  bool _obscureNew = true;
  bool _obscureConfirm = true;
  bool _submitting = false;

  String? _currentServerError;
  String? _newServerError;

  @override
  void dispose() {
    _currentPasswordCtrl.dispose();
    _newPasswordCtrl.dispose();
    _confirmPasswordCtrl.dispose();
    super.dispose();
  }

  // ---------------------------------------------------------------------------
  // Validation — mirrors backend complexity rules.
  // ---------------------------------------------------------------------------

  // Pre-compiled regexes so we don't recompile on every form rebuild.
  static final RegExp _hasUpper = RegExp(r'[A-Z]');
  static final RegExp _hasLower = RegExp(r'[a-z]');
  static final RegExp _hasDigit = RegExp(r'\d');
  static final RegExp _hasSpecial = RegExp(r'[^A-Za-z0-9]');

  String? _validateCurrent(String? value, AppLocalizations l10n) {
    if (value == null || value.isEmpty) {
      return l10n.accountErrorPasswordRequired;
    }
    return null;
  }

  String? _validateNew(String? value, AppLocalizations l10n) {
    if (value == null || value.isEmpty) {
      return l10n.accountErrorPasswordRequired;
    }
    if (value.length < 10 ||
        !_hasUpper.hasMatch(value) ||
        !_hasLower.hasMatch(value) ||
        !_hasDigit.hasMatch(value) ||
        !_hasSpecial.hasMatch(value)) {
      return l10n.accountErrorWeakPassword;
    }
    return null;
  }

  String? _validateConfirm(String? value, AppLocalizations l10n) {
    if (value == null || value.isEmpty) {
      return l10n.accountErrorPasswordRequired;
    }
    if (value != _newPasswordCtrl.text) {
      return l10n.accountErrorPasswordMismatch;
    }
    return null;
  }

  // ---------------------------------------------------------------------------
  // Submit
  // ---------------------------------------------------------------------------

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _currentServerError = null;
      _newServerError = null;
    });
    if (!(_formKey.currentState?.validate() ?? false)) return;

    setState(() => _submitting = true);
    try {
      final repo = ref.read(accountRepositoryProvider);
      await repo.changePassword(
        currentPassword: _currentPasswordCtrl.text,
        newPassword: _newPasswordCtrl.text,
      );
      if (!mounted) return;
      // Wipe credential fields immediately on success so they never linger.
      _currentPasswordCtrl.clear();
      _newPasswordCtrl.clear();
      _confirmPasswordCtrl.clear();
      await ref.read(authProvider.notifier).logout();
      if (!mounted) return;
      _showSnack(l10n.accountChangePasswordSuccess);
      context.go(RoutePaths.login);
    } on AccountFailureException catch (e) {
      if (!mounted) return;
      // Always clear the typed-in passwords on error — never let
      // credentials linger in a controlled input after a failure.
      _currentPasswordCtrl.clear();
      _newPasswordCtrl.clear();
      _confirmPasswordCtrl.clear();
      _handleFailure(e.failure, l10n);
    } catch (_) {
      if (!mounted) return;
      _showSnack(l10n.accountErrorGeneric);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  void _handleFailure(AccountFailure failure, AppLocalizations l10n) {
    failure.when(
      invalidCredentials: () => setState(() {
        _currentServerError = l10n.accountErrorInvalidCredentials;
        _formKey.currentState?.validate();
      }),
      weakPassword: () => setState(() {
        _newServerError = l10n.accountErrorWeakPassword;
        _formKey.currentState?.validate();
      }),
      samePassword: () => setState(() {
        _newServerError = l10n.accountErrorSamePassword;
        _formKey.currentState?.validate();
      }),
      sessionInvalid: () async {
        await ref.read(authProvider.notifier).logout();
        if (!mounted) return;
        _showSnack(l10n.accountErrorSessionInvalid);
        context.go(RoutePaths.login);
      },
      // The remaining variants only happen on the change-email path.
      invalidEmail: () => _showSnack(l10n.accountErrorGeneric),
      sameEmail: () => _showSnack(l10n.accountErrorGeneric),
      emailAlreadyExists: () => _showSnack(l10n.accountErrorGeneric),
      network: () => _showSnack(l10n.accountErrorNetwork),
      unknown: (_) => _showSnack(l10n.accountErrorGeneric),
    );
  }

  void _showSnack(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  // ---------------------------------------------------------------------------
  // UI
  // ---------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();

    return Scaffold(
      appBar: AppBar(title: Text(l10n.accountChangePasswordTitle)),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  l10n.accountChangePasswordSubtitle,
                  style: SoleilTextStyles.body.copyWith(
                    color: colors?.mutedForeground ??
                        theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 20),
                _PasswordField(
                  label: l10n.accountCurrentPassword,
                  controller: _currentPasswordCtrl,
                  obscure: _obscureCurrent,
                  toggleObscure: () =>
                      setState(() => _obscureCurrent = !_obscureCurrent),
                  enabled: !_submitting,
                  autofill: AutofillHints.password,
                  validator: (v) =>
                      _currentServerError ?? _validateCurrent(v, l10n),
                  onChanged: () {
                    if (_currentServerError != null) {
                      setState(() => _currentServerError = null);
                    }
                  },
                  showLabel: l10n.passwordShow,
                  hideLabel: l10n.passwordHide,
                ),
                const SizedBox(height: 16),
                _PasswordField(
                  label: l10n.accountNewPassword,
                  controller: _newPasswordCtrl,
                  obscure: _obscureNew,
                  toggleObscure: () =>
                      setState(() => _obscureNew = !_obscureNew),
                  enabled: !_submitting,
                  autofill: AutofillHints.newPassword,
                  validator: (v) => _newServerError ?? _validateNew(v, l10n),
                  onChanged: () {
                    if (_newServerError != null) {
                      setState(() => _newServerError = null);
                    }
                  },
                  showLabel: l10n.passwordShow,
                  hideLabel: l10n.passwordHide,
                ),
                const SizedBox(height: 8),
                Text(
                  l10n.accountPasswordHint,
                  style: SoleilTextStyles.caption.copyWith(
                    color: colors?.mutedForeground ??
                        theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 16),
                _PasswordField(
                  label: l10n.accountConfirmPassword,
                  controller: _confirmPasswordCtrl,
                  obscure: _obscureConfirm,
                  toggleObscure: () =>
                      setState(() => _obscureConfirm = !_obscureConfirm),
                  enabled: !_submitting,
                  autofill: AutofillHints.newPassword,
                  validator: (v) => _validateConfirm(v, l10n),
                  showLabel: l10n.passwordShow,
                  hideLabel: l10n.passwordHide,
                ),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: _submitting ? null : _submit,
                  child: _submitting
                      ? const SizedBox(
                          height: 18,
                          width: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Text(l10n.accountChangePasswordCta),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Reusable password input row with show/hide toggle. Kept private so
/// the change-password screen owns its visual treatment without
/// leaking into the rest of the app.
class _PasswordField extends StatelessWidget {
  const _PasswordField({
    required this.label,
    required this.controller,
    required this.obscure,
    required this.toggleObscure,
    required this.enabled,
    required this.validator,
    required this.autofill,
    required this.showLabel,
    required this.hideLabel,
    this.onChanged,
  });

  final String label;
  final TextEditingController controller;
  final bool obscure;
  final VoidCallback toggleObscure;
  final bool enabled;
  final FormFieldValidator<String> validator;
  final String autofill;
  final String showLabel;
  final String hideLabel;
  final VoidCallback? onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: SoleilTextStyles.bodyEmphasis.copyWith(
            color: theme.colorScheme.onSurface,
          ),
        ),
        const SizedBox(height: 6),
        TextFormField(
          controller: controller,
          obscureText: obscure,
          enabled: enabled,
          autofillHints: [autofill],
          decoration: InputDecoration(
            suffixIcon: IconButton(
              icon: Icon(
                obscure
                    ? Icons.visibility_outlined
                    : Icons.visibility_off_outlined,
              ),
              onPressed: toggleObscure,
              tooltip: obscure ? showLabel : hideLabel,
            ),
          ),
          validator: validator,
          onChanged: onChanged != null ? (_) => onChanged!() : null,
        ),
      ],
    );
  }
}
