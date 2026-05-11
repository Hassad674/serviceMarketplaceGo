import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/two_factor_api.dart';
import '../providers/auth_provider.dart';

/// TwoFactorSection — toggle for the email-based 2FA flag.
///
/// Lives inside the AccountScreen. Three states drive the UX:
///
/// * OFF: shows the OFF description + an "Activer" CTA. First tap kicks off
///   `POST /me/two-factor/enable` (no body) so the backend emails a 6-digit
///   challenge, then opens a modal asking for that code. On submit, calls
///   the same endpoint with `{code}` to flip the flag on.
/// * ON: shows the ON description + a "Désactiver" CTA. Tap opens a modal
///   asking for the current password, then `POST /me/two-factor/disable`
///   with `{current_password}` flips the flag off.
///
/// FIX-2FA: the backend now surfaces `two_factor_email_enabled` on
/// `/auth/me`, so the toggle reads the persisted value from the auth
/// state on every mount (and after every refresh). The local
/// `_enabled` field still exists for the optimistic flip during a
/// mutation but is re-synced from the auth state whenever the
/// underlying map changes.
class TwoFactorSection extends ConsumerStatefulWidget {
  const TwoFactorSection({super.key});

  @override
  ConsumerState<TwoFactorSection> createState() => _TwoFactorSectionState();
}

class _TwoFactorSectionState extends ConsumerState<TwoFactorSection> {
  // Local "optimistic" anchor. The toggle's source of truth is the
  // auth state (see `build` — the watched value overrides this field
  // on every rebuild). _enabled is only used during the brief moment
  // between a mutation succeeding and the refreshSession()/auth
  // state landing the new value.
  bool _enabled = false;
  bool _busy = false;

  Future<void> _handleToggle(bool desired) async {
    if (_busy) return;
    if (desired) {
      await _enableFlow();
    } else {
      await _disableFlow();
    }
  }

  Future<void> _enableFlow() async {
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    final api = ref.read(twoFactorApiProvider);

    setState(() => _busy = true);
    try {
      await api.startEnable();
    } on DioException catch (e) {
      setState(() => _busy = false);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            ApiException.fromDioException(e).message,
          ),
        ),
      );
      return;
    } catch (_) {
      setState(() => _busy = false);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.twoFactorErrorGeneric)),
      );
      return;
    }

    if (!mounted) return;
    final code = await showDialog<String>(
      context: context,
      barrierDismissible: false,
      builder: (_) => _EnableTwoFactorDialog(l10n: l10n),
    );

    if (!mounted) {
      setState(() => _busy = false);
      return;
    }

    if (code == null) {
      // User cancelled. Backend has issued a challenge but it will expire
      // on its own — nothing to clean up client-side.
      setState(() => _busy = false);
      return;
    }

    try {
      await api.confirmEnable(code: code);
      if (!mounted) return;
      setState(() {
        _enabled = true;
        _busy = false;
      });
      // FIX-2FA: pull the persisted flag back from /auth/me so the
      // toggle's source of truth (auth state) reflects the new value
      // immediately. Without this, switching tabs and coming back to
      // the security section would briefly show the old value until
      // the next session refresh.
      unawaited(ref.read(authProvider.notifier).refreshSession());
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.twoFactorEnabledToast)),
      );
    } on DioException catch (e) {
      setState(() => _busy = false);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(ApiException.fromDioException(e).message)),
      );
    } catch (_) {
      setState(() => _busy = false);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.twoFactorErrorGeneric)),
      );
    }
  }

  Future<void> _disableFlow() async {
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    final api = ref.read(twoFactorApiProvider);

    final password = await showDialog<String>(
      context: context,
      barrierDismissible: false,
      builder: (_) => _DisableTwoFactorDialog(l10n: l10n),
    );

    if (!mounted || password == null) return;

    setState(() => _busy = true);
    try {
      await api.disable(currentPassword: password);
      if (!mounted) return;
      setState(() {
        _enabled = false;
        _busy = false;
      });
      // FIX-2FA: see _enableFlow above — refresh auth so other screens
      // observing the user object pick up the new flag value.
      unawaited(ref.read(authProvider.notifier).refreshSession());
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.twoFactorDisabledToast)),
      );
    } on DioException catch (e) {
      setState(() => _busy = false);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(ApiException.fromDioException(e).message)),
      );
    } catch (_) {
      setState(() => _busy = false);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(l10n.twoFactorErrorGeneric)),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    // FIX-2FA: derive the toggle's truthy state from the auth state's
    // user.two_factor_email_enabled field. Each rebuild re-reads it so
    // a freshly-landed /auth/me payload (e.g. after a refresh) is
    // reflected without an explicit listener. The local `_enabled`
    // field is preserved as the optimistic anchor during mutations —
    // it gets re-aligned with the auth state on the next rebuild after
    // _refreshAuthUser() lands.
    final authUser = ref.watch(authProvider.select((s) => s.user));
    final flagFromAuth = authUser?['two_factor_email_enabled'];
    final enabled = _busy
        ? _enabled
        : (flagFromAuth is bool ? flagFromAuth : _enabled);
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final colors = theme.extension<AppColors>();
    final description = enabled
        ? l10n.twoFactorToggleDescOn
        : l10n.twoFactorToggleDescOff;

    return Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                l10n.twoFactorToggleTitle,
                style: SoleilTextStyles.body.copyWith(
                  color: theme.colorScheme.onSurface,
                  fontWeight: FontWeight.w600,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                description,
                style: SoleilTextStyles.caption.copyWith(
                  color: colors?.mutedForeground ??
                      theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 12),
        Switch(
          value: enabled,
          onChanged: _busy ? null : _handleToggle,
        ),
      ],
    );
  }
}

/// Dialog asking the user for the 6-digit code that just landed in their
/// inbox after the first POST /me/two-factor/enable call.
class _EnableTwoFactorDialog extends StatefulWidget {
  const _EnableTwoFactorDialog({required this.l10n});

  final AppLocalizations l10n;

  @override
  State<_EnableTwoFactorDialog> createState() => _EnableTwoFactorDialogState();
}

class _EnableTwoFactorDialogState extends State<_EnableTwoFactorDialog> {
  final _controller = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() {
    final code = _controller.text.trim();
    if (code.length != 6) {
      setState(() => _error = widget.l10n.twoFactorCodeLengthError);
      return;
    }
    Navigator.of(context).pop(code);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = widget.l10n;
    return AlertDialog(
      title: Text(l10n.twoFactorSectionTitle),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l10n.twoFactorEnablePrompt),
          const SizedBox(height: 16),
          TextField(
            controller: _controller,
            keyboardType: TextInputType.number,
            autofocus: true,
            maxLength: 6,
            textAlign: TextAlign.center,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration: InputDecoration(
              labelText: l10n.twoFactorCodeLabel,
              hintText: l10n.twoFactorCodeHint,
              counterText: '',
              errorText: _error,
            ),
            onSubmitted: (_) => _submit(),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(l10n.twoFactorCancel),
        ),
        FilledButton(
          onPressed: _submit,
          child: Text(l10n.twoFactorConfirmEnableCta),
        ),
      ],
    );
  }
}

/// Dialog asking the user for their current password so the backend can
/// verify identity before flipping the flag off.
class _DisableTwoFactorDialog extends StatefulWidget {
  const _DisableTwoFactorDialog({required this.l10n});

  final AppLocalizations l10n;

  @override
  State<_DisableTwoFactorDialog> createState() =>
      _DisableTwoFactorDialogState();
}

class _DisableTwoFactorDialogState extends State<_DisableTwoFactorDialog> {
  final _controller = TextEditingController();
  String? _error;
  bool _obscure = true;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() {
    final value = _controller.text;
    if (value.isEmpty) {
      setState(() => _error = widget.l10n.twoFactorErrorPasswordRequired);
      return;
    }
    Navigator.of(context).pop(value);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = widget.l10n;
    return AlertDialog(
      title: Text(l10n.twoFactorSectionTitle),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l10n.twoFactorDisablePrompt),
          const SizedBox(height: 16),
          TextField(
            controller: _controller,
            obscureText: _obscure,
            autofocus: true,
            decoration: InputDecoration(
              labelText: l10n.twoFactorCurrentPasswordLabel,
              errorText: _error,
              suffixIcon: IconButton(
                icon: Icon(
                  _obscure
                      ? Icons.visibility_outlined
                      : Icons.visibility_off_outlined,
                ),
                onPressed: () => setState(() => _obscure = !_obscure),
              ),
            ),
            onSubmitted: (_) => _submit(),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(l10n.twoFactorCancel),
        ),
        FilledButton(
          onPressed: _submit,
          child: Text(l10n.twoFactorConfirmDisableCta),
        ),
      ],
    );
  }
}
