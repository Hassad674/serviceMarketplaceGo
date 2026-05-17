import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/auth_provider.dart';

/// Login screen — ported to Soleil v2.
///
/// Single column layout (DesignedTrust mobile design): brand mark on top, Fraunces
/// headline + italic subtitle, email + password fields, italic forgot link,
/// rounded-pill primary CTA, footer link to register. Source-of-truth:
/// `design/assets/sources/phase1/soleil-app-lot5.jsx` `AppLogin`.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _otpFormKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _otpController = TextEditingController();
  bool _obscurePassword = true;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    _otpController.dispose();
    super.dispose();
  }

  Future<void> _handleLogin() async {
    if (!_formKey.currentState!.validate()) return;

    final result = await ref.read(authProvider.notifier).login(
          email: _emailController.text.trim(),
          password: _passwordController.text,
        );

    if (!mounted) return;
    switch (result) {
      case LoginResult.success:
        context.go(RoutePaths.dashboard);
      case LoginResult.requires2fa:
        // State carries `pendingTwoFactor` — the build() method swaps to
        // the OTP form on the next rebuild. Reset the OTP controller so
        // the user starts fresh if they came back to this branch.
        _otpController.clear();
      case LoginResult.failed:
        // Error message lives on AuthState — banner already renders.
        break;
    }
  }

  Future<void> _handleVerify2fa() async {
    if (!_otpFormKey.currentState!.validate()) return;
    final ok = await ref
        .read(authProvider.notifier)
        .verifyTwoFactor(code: _otpController.text.trim());
    if (ok && mounted) {
      context.go(RoutePaths.dashboard);
    }
  }

  void _backToPassword() {
    ref.read(authProvider.notifier).cancelPendingTwoFactor();
    _otpController.clear();
  }

  Future<void> _resend2faCode() async {
    // Re-issuing a code requires re-running the password step. The
    // simplest UX-correct path is to bring the user back to the password
    // form so they confirm their credentials and receive a fresh
    // challenge. Cheaper + more secure than caching the password.
    _backToPassword();
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final l10n = AppLocalizations.of(context)!;
    final showOtpForm = authState.pendingTwoFactor != null;

    return Scaffold(
      backgroundColor: colorScheme.surfaceContainerLowest,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(28, 20, 28, 28),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const SizedBox(height: 40),
              const _BrandLogo(),
              const SizedBox(height: 36),
              Text(
                showOtpForm ? l10n.twoFactorTitle : l10n.loginTitle,
                style: SoleilTextStyles.displayM.copyWith(
                  color: colorScheme.onSurface,
                  fontWeight: FontWeight.w600,
                  height: 1.1,
                ),
              ),
              const SizedBox(height: 8),
              Text(
                showOtpForm ? l10n.twoFactorSubtitle : l10n.loginSubtitle,
                style: SoleilTextStyles.bodyLarge.copyWith(
                  fontStyle: FontStyle.italic,
                  color: colorScheme.onSurfaceVariant,
                  fontFamily: SoleilTextStyles.headlineLarge.fontFamily,
                ),
              ),
              const SizedBox(height: 36),

              if (authState.errorMessage != null) ...[
                _ErrorBanner(message: authState.errorMessage!),
                const SizedBox(height: 16),
              ],

              if (showOtpForm)
                _TwoFactorOtpForm(
                  formKey: _otpFormKey,
                  controller: _otpController,
                  isSubmitting: authState.isSubmitting,
                  l10n: l10n,
                  onSubmit: _handleVerify2fa,
                  onResend: _resend2faCode,
                  onBack: _backToPassword,
                )
              else
                _buildPasswordForm(l10n, colorScheme, authState),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildPasswordForm(
    AppLocalizations l10n,
    ColorScheme colorScheme,
    AuthState authState,
  ) {
    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _FieldLabel(text: l10n.email),
          const SizedBox(height: 6),
          TextFormField(
            controller: _emailController,
            decoration: InputDecoration(hintText: l10n.emailHint),
            keyboardType: TextInputType.emailAddress,
            textInputAction: TextInputAction.next,
            autofillHints: const [AutofillHints.email],
            validator: (value) {
              if (value == null || value.trim().isEmpty) {
                return l10n.fieldRequired;
              }
              if (!value.contains('@')) {
                return l10n.invalidEmail;
              }
              return null;
            },
          ),
          const SizedBox(height: 12),
          _FieldLabel(text: l10n.password),
          const SizedBox(height: 6),
          TextFormField(
            controller: _passwordController,
            decoration: InputDecoration(
              hintText: l10n.passwordHint,
              suffixIcon: IconButton(
                icon: Icon(
                  _obscurePassword
                      ? Icons.visibility_outlined
                      : Icons.visibility_off_outlined,
                  color: colorScheme.onSurfaceVariant,
                ),
                onPressed: () {
                  setState(() {
                    _obscurePassword = !_obscurePassword;
                  });
                },
              ),
            ),
            obscureText: _obscurePassword,
            textInputAction: TextInputAction.done,
            autofillHints: const [AutofillHints.password],
            onFieldSubmitted: (_) => _handleLogin(),
            validator: (value) {
              if (value == null || value.isEmpty) {
                return l10n.fieldRequired;
              }
              return null;
            },
          ),
          Align(
            alignment: Alignment.centerRight,
            child: TextButton(
              style: TextButton.styleFrom(
                padding: const EdgeInsets.symmetric(
                  horizontal: 4,
                  vertical: 8,
                ),
                minimumSize: Size.zero,
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
              ),
              onPressed: () {
                // Forgot-password navigation is handled outside this
                // visual port (separate feature dispatch).
              },
              child: Text(
                l10n.forgotPassword,
                style: SoleilTextStyles.caption.copyWith(
                  color: colorScheme.primary,
                  fontWeight: FontWeight.w600,
                  fontStyle: FontStyle.italic,
                  fontFamily: SoleilTextStyles.headlineLarge.fontFamily,
                ),
              ),
            ),
          ),
          const SizedBox(height: 16),
          SizedBox(
            height: 52,
            child: ElevatedButton(
              onPressed: authState.isSubmitting ? null : _handleLogin,
              child: authState.isSubmitting
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: Colors.white,
                      ),
                    )
                  : Text(l10n.signIn),
            ),
          ),
          const SizedBox(height: 32),
          _RegisterPrompt(
            promptText: l10n.noAccount,
            ctaText: l10n.signUp,
            onTap: () => context.go(RoutePaths.register),
          ),
        ],
      ),
    );
  }
}

/// DesignedTrust Services brand mark — the "Key Hole" pictogram.
///
/// Drawn natively via [CustomPainter] (no bitmap asset) so the mark stays
/// crisp at any density, mirroring the web `BrandLogo` SVG: a quarter-round
/// "D" in the brand orange with a keyhole punched out in negative (white).
///
/// BRAND COLOR: 0xFFFF7A1F is the fixed DesignedTrust brand orange — this
/// widget IS the brand asset (like [Portrait]'s painter), so the literal is
/// intentional; no Soleil theme token applies to a logo. 48x48 footprint,
/// matching the previous mark.
class _BrandLogo extends StatelessWidget {
  const _BrandLogo();

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Semantics(
        label: 'DesignedTrust Services',
        image: true,
        child: const SizedBox(
          width: 48,
          height: 48,
          child: CustomPaint(painter: _BrandLogoPainter()),
        ),
      ),
    );
  }
}

/// Paints the Key Hole mark. Geometry is normalized from the canonical
/// 200x200 SVG box (D spans x:0->180, y:0->200) onto the widget size.
class _BrandLogoPainter extends CustomPainter {
  const _BrandLogoPainter();

  static const Color _brandOrange = Color(0xFFFF7A1F);

  @override
  void paint(Canvas canvas, Size size) {
    final double s = size.width / 200.0; // uniform scale from the SVG box
    final Paint orange = Paint()
      ..color = _brandOrange
      ..isAntiAlias = true;
    final Paint white = Paint()
      ..color = const Color(0xFFFFFFFF)
      ..isAntiAlias = true;

    // D shape: M0 0 H80 A100 100 0 0 1 80 200 H0 Z (quarter-round right edge).
    final Path d = Path()
      ..moveTo(0, 0)
      ..lineTo(80 * s, 0)
      ..arcToPoint(
        Offset(180 * s, 100 * s),
        radius: Radius.circular(100 * s),
      )
      ..arcToPoint(
        Offset(80 * s, 200 * s),
        radius: Radius.circular(100 * s),
      )
      ..lineTo(0, 200 * s)
      ..close();
    canvas.drawPath(d, orange);

    // Keyhole in negative: circle (cx100 cy80 r22) + stem (M86 80 V148 H114).
    canvas.drawCircle(Offset(100 * s, 80 * s), 22 * s, white);
    final Path stem = Path()
      ..moveTo(86 * s, 80 * s)
      ..lineTo(86 * s, 148 * s)
      ..lineTo(114 * s, 148 * s)
      ..lineTo(114 * s, 80 * s)
      ..close();
    canvas.drawPath(stem, white);
  }

  @override
  bool shouldRepaint(covariant _BrandLogoPainter oldDelegate) => false;
}

/// Small uppercase-style label used above each form field.
class _FieldLabel extends StatelessWidget {
  const _FieldLabel({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Text(
      text,
      style: SoleilTextStyles.caption.copyWith(
        color: colorScheme.onSurface,
        fontWeight: FontWeight.w600,
      ),
    );
  }
}

/// Footer prompt — "No account yet? Sign Up" with the CTA in corail.
class _RegisterPrompt extends StatelessWidget {
  const _RegisterPrompt({
    required this.promptText,
    required this.ctaText,
    required this.onTap,
  });

  final String promptText;
  final String ctaText;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Align(
      alignment: Alignment.center,
      child: Wrap(
        crossAxisAlignment: WrapCrossAlignment.center,
        alignment: WrapAlignment.center,
        children: [
          Text(
            promptText,
            style: SoleilTextStyles.caption.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(width: 6),
          GestureDetector(
            onTap: onTap,
            behavior: HitTestBehavior.opaque,
            child: Text(
              ctaText,
              style: SoleilTextStyles.caption.copyWith(
                color: colorScheme.primary,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// Error banner shown above the form when authentication fails.
class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message});

  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: colorScheme.error.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: colorScheme.error.withValues(alpha: 0.3),
        ),
      ),
      child: Row(
        children: [
          Icon(
            Icons.error_outline,
            color: colorScheme.error,
            size: 20,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: SoleilTextStyles.caption.copyWith(
                color: colorScheme.error,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// Form rendered when the password step succeeded but the user has 2FA on.
///
/// 6-digit numeric input, primary CTA "Verify code", "Resend code" pill and a
/// "Back to sign in" tertiary button. The field validates length only —
/// server-side errors flow through [AuthState.errorMessage] like the password
/// path so the same banner above is re-used.
class _TwoFactorOtpForm extends StatelessWidget {
  const _TwoFactorOtpForm({
    required this.formKey,
    required this.controller,
    required this.isSubmitting,
    required this.l10n,
    required this.onSubmit,
    required this.onResend,
    required this.onBack,
  });

  final GlobalKey<FormState> formKey;
  final TextEditingController controller;
  final bool isSubmitting;
  final AppLocalizations l10n;
  final VoidCallback onSubmit;
  final VoidCallback onResend;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Form(
      key: formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _FieldLabel(text: l10n.twoFactorCodeLabel),
          const SizedBox(height: 6),
          TextFormField(
            controller: controller,
            keyboardType: TextInputType.number,
            textInputAction: TextInputAction.done,
            autofocus: true,
            maxLength: 6,
            inputFormatters: [
              FilteringTextInputFormatter.digitsOnly,
            ],
            decoration: InputDecoration(
              hintText: l10n.twoFactorCodeHint,
              counterText: '',
            ),
            style: SoleilTextStyles.headlineLarge.copyWith(
              letterSpacing: 6,
              color: colorScheme.onSurface,
            ),
            textAlign: TextAlign.center,
            onFieldSubmitted: (_) => onSubmit(),
            validator: (value) {
              final v = (value ?? '').trim();
              if (v.length != 6) {
                return l10n.twoFactorCodeLengthError;
              }
              return null;
            },
          ),
          const SizedBox(height: 16),
          SizedBox(
            height: 52,
            child: ElevatedButton(
              onPressed: isSubmitting ? null : onSubmit,
              child: isSubmitting
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        color: Colors.white,
                      ),
                    )
                  : Text(l10n.twoFactorVerifyCta),
            ),
          ),
          const SizedBox(height: 12),
          OutlinedButton(
            onPressed: isSubmitting ? null : onResend,
            child: Text(l10n.twoFactorResend),
          ),
          const SizedBox(height: 8),
          TextButton(
            onPressed: isSubmitting ? null : onBack,
            child: Text(l10n.twoFactorBackToLogin),
          ),
        ],
      ),
    );
  }
}
