import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/account_repository_impl.dart';
import '../../domain/entities/account_failure.dart';

/// ChangeEmailScreen — mobile counterpart of `EmailSettings` (web).
///
/// On success the backend bumps the session version, so the in-flight
/// access token is already invalid by the time we get the 200 response.
/// We mirror the web flow by clearing local credentials (`logout()`)
/// and routing back to `/login` with a SnackBar inviting the user to
/// reconnect with the new email.
class ChangeEmailScreen extends ConsumerStatefulWidget {
  const ChangeEmailScreen({super.key});

  @override
  ConsumerState<ChangeEmailScreen> createState() => _ChangeEmailScreenState();
}

class _ChangeEmailScreenState extends ConsumerState<ChangeEmailScreen> {
  final _formKey = GlobalKey<FormState>();
  final _currentPasswordCtrl = TextEditingController();
  final _newEmailCtrl = TextEditingController();
  bool _obscurePassword = true;
  bool _submitting = false;

  // Server-driven inline errors. Reset before each submit. We hold these
  // in state instead of relying on FormFieldState validators so the same
  // field can be re-validated client-side on the next keystroke.
  String? _passwordServerError;
  String? _emailServerError;

  @override
  void dispose() {
    _currentPasswordCtrl.dispose();
    _newEmailCtrl.dispose();
    super.dispose();
  }

  // ---------------------------------------------------------------------------
  // Validation — local mirror of backend rules
  // ---------------------------------------------------------------------------

  // RFC 5322-lite, matches the backend's validator. Kept as a single
  // RegExp constant so we don't recompile on every form rebuild.
  static final RegExp _emailRegex = RegExp(
    r'^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$',
  );

  String? _validateEmail(String? value, AppLocalizations l10n) {
    final v = value?.trim() ?? '';
    if (v.isEmpty) return l10n.accountErrorEmailRequired;
    if (!_emailRegex.hasMatch(v)) return l10n.accountErrorInvalidEmail;
    return null;
  }

  String? _validatePassword(String? value, AppLocalizations l10n) {
    if (value == null || value.isEmpty) {
      return l10n.accountErrorPasswordRequired;
    }
    return null;
  }

  // ---------------------------------------------------------------------------
  // Submit
  // ---------------------------------------------------------------------------

  Future<void> _submit() async {
    final l10n = AppLocalizations.of(context)!;
    setState(() {
      _passwordServerError = null;
      _emailServerError = null;
    });
    if (!(_formKey.currentState?.validate() ?? false)) return;

    setState(() => _submitting = true);
    try {
      final repo = ref.read(accountRepositoryProvider);
      await repo.changeEmail(
        currentPassword: _currentPasswordCtrl.text,
        newEmail: _newEmailCtrl.text.trim(),
      );
      if (!mounted) return;
      // Backend bumped session version → drop tokens and route to login.
      await ref.read(authProvider.notifier).logout();
      if (!mounted) return;
      _showSnack(l10n.accountChangeEmailSuccess);
      context.go(RoutePaths.login);
    } on AccountFailureException catch (e) {
      if (!mounted) return;
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
      invalidEmail: () => setState(() {
        _emailServerError = l10n.accountErrorInvalidEmail;
        _formKey.currentState?.validate();
      }),
      sameEmail: () => setState(() {
        _emailServerError = l10n.accountErrorSameEmail;
        _formKey.currentState?.validate();
      }),
      emailAlreadyExists: () => setState(() {
        _emailServerError = l10n.accountErrorEmailAlreadyExists;
        _formKey.currentState?.validate();
      }),
      invalidCredentials: () => setState(() {
        _passwordServerError = l10n.accountErrorInvalidCredentials;
        _currentPasswordCtrl.clear();
        _formKey.currentState?.validate();
      }),
      sessionInvalid: () async {
        // Already logged out server-side. Mirror locally + redirect.
        await ref.read(authProvider.notifier).logout();
        if (!mounted) return;
        _showSnack(l10n.accountErrorSessionInvalid);
        context.go(RoutePaths.login);
      },
      weakPassword: () => _showSnack(l10n.accountErrorGeneric),
      samePassword: () => _showSnack(l10n.accountErrorGeneric),
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
    final authState = ref.watch(authProvider);
    final currentEmail = authState.user?['email'] as String? ?? '—';

    return Scaffold(
      appBar: AppBar(title: Text(l10n.accountChangeEmailTitle)),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  l10n.accountChangeEmailSubtitle,
                  style: SoleilTextStyles.body.copyWith(
                    color: colors?.mutedForeground ??
                        theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 16),
                _CurrentEmailCard(email: currentEmail, label: l10n.accountCurrentEmail),
                const SizedBox(height: 16),
                _LabeledField(
                  label: l10n.accountCurrentPassword,
                  child: TextFormField(
                    controller: _currentPasswordCtrl,
                    obscureText: _obscurePassword,
                    enabled: !_submitting,
                    autofillHints: const [AutofillHints.password],
                    decoration: InputDecoration(
                      hintText: l10n.accountCurrentPasswordHint,
                      suffixIcon: IconButton(
                        icon: Icon(
                          _obscurePassword
                              ? Icons.visibility_outlined
                              : Icons.visibility_off_outlined,
                        ),
                        onPressed: () => setState(
                          () => _obscurePassword = !_obscurePassword,
                        ),
                        tooltip: _obscurePassword
                            ? l10n.passwordShow
                            : l10n.passwordHide,
                      ),
                    ),
                    validator: (v) =>
                        _passwordServerError ?? _validatePassword(v, l10n),
                    onChanged: (_) {
                      if (_passwordServerError != null) {
                        setState(() => _passwordServerError = null);
                      }
                    },
                  ),
                ),
                const SizedBox(height: 16),
                _LabeledField(
                  label: l10n.accountNewEmail,
                  child: TextFormField(
                    controller: _newEmailCtrl,
                    enabled: !_submitting,
                    keyboardType: TextInputType.emailAddress,
                    autofillHints: const [AutofillHints.email],
                    decoration: InputDecoration(
                      hintText: l10n.accountNewEmailHint,
                    ),
                    validator: (v) =>
                        _emailServerError ?? _validateEmail(v, l10n),
                    onChanged: (_) {
                      if (_emailServerError != null) {
                        setState(() => _emailServerError = null);
                      }
                    },
                  ),
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
                      : Text(l10n.accountChangeEmailCta),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Read-only card showing the current account email above the form.
class _CurrentEmailCard extends StatelessWidget {
  const _CurrentEmailCard({required this.email, required this.label});

  final String email;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: colors?.border ?? theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label.toUpperCase(),
            style: SoleilTextStyles.caption.copyWith(
              color: colors?.mutedForeground ??
                  theme.colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w600,
              letterSpacing: 1.2,
            ),
          ),
          const SizedBox(height: 6),
          SelectableText(
            email,
            style: SoleilTextStyles.monoLarge.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
        ],
      ),
    );
  }
}

/// A small label + input wrapper to keep visual rhythm with the rest
/// of the Soleil v2 forms.
class _LabeledField extends StatelessWidget {
  const _LabeledField({required this.label, required this.child});

  final String label;
  final Widget child;

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
        child,
      ],
    );
  }
}
