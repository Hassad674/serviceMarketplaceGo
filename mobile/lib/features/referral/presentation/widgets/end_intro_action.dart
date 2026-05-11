import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/referral_provider.dart';
import 'end_intro_confirmation_dialog.dart';

/// EndIntroAction — WALLET-UNIFY Run D parity with web Run C
/// `EndIntroAction`. Button → dialog → mutation → badge state
/// machine for ending a single attribution.
///
/// - [initialEndedAt] != null → render the badge directly (the page
///   was reloaded after a previous successful end).
/// - On tap → open [EndIntroConfirmationDialog].
/// - On confirm → fire [endIntroAttribution]. On success → swap to
///   the badge with the returned `ended_at`. The provider
///   invalidations already happen inside the action helper, so the
///   parent ref tree refreshes automatically.
/// - On error → show a snackbar with the mapped error message.
class EndIntroAction extends ConsumerStatefulWidget {
  const EndIntroAction({
    super.key,
    required this.referralId,
    required this.attributionId,
    this.providerName,
    this.clientName,
    this.initialEndedAt,
  });

  final String referralId;
  final String attributionId;
  final String? providerName;
  final String? clientName;
  final String? initialEndedAt;

  @override
  ConsumerState<EndIntroAction> createState() => _EndIntroActionState();
}

class _EndIntroActionState extends ConsumerState<EndIntroAction> {
  String? _endedAt;
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _endedAt = widget.initialEndedAt;
  }

  Future<void> _onTap() async {
    if (_submitting) return;
    final confirmed = await showEndIntroConfirmationDialog(
      context: context,
      providerName: widget.providerName,
      clientName: widget.clientName,
    );
    if (confirmed != true || !mounted) return;
    setState(() => _submitting = true);
    final result = await endIntroAttribution(
      ref,
      referralId: widget.referralId,
      attributionId: widget.attributionId,
    );
    if (!mounted) return;
    if (result.isSuccess) {
      setState(() {
        _endedAt = result.endedAt;
        _submitting = false;
      });
    } else {
      setState(() => _submitting = false);
      _showError(result.errorCode);
    }
  }

  void _showError(String? code) {
    final l10n = AppLocalizations.of(context)!;
    String msg;
    switch (code) {
      case 'forbidden':
        msg = l10n.referralEndIntroErrorForbidden;
        break;
      case 'not_found':
        msg = l10n.referralEndIntroErrorNotFound;
        break;
      default:
        msg = l10n.referralEndIntroErrorGeneric;
    }
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    if (_endedAt != null) {
      return Container(
        key: const ValueKey('end-intro-badge'),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        decoration: BoxDecoration(
          color: (theme.extension<AppColors>()?.success ??
                  theme.colorScheme.primary)
              .withValues(alpha: 0.14),
          borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.check_circle,
              size: 14,
              color: theme.extension<AppColors>()?.success ??
                  theme.colorScheme.primary,
            ),
            const SizedBox(width: 6),
            Text(
              l10n.referralEndIntroBadge(_formatDate(_endedAt!)),
              style: theme.textTheme.labelSmall?.copyWith(
                fontWeight: FontWeight.w700,
                color: theme.extension<AppColors>()?.success ??
                    theme.colorScheme.primary,
              ),
            ),
          ],
        ),
      );
    }
    return OutlinedButton.icon(
      key: const ValueKey('end-intro-trigger'),
      onPressed: _submitting ? null : _onTap,
      icon: _submitting
          ? const SizedBox(
              width: 14,
              height: 14,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : Icon(
              Icons.stop_circle_outlined,
              size: 16,
              color: theme.colorScheme.error,
            ),
      label: Text(
        l10n.referralEndIntroCtaLabel,
        style: TextStyle(color: theme.colorScheme.error),
      ),
      style: OutlinedButton.styleFrom(
        side: BorderSide(
          color: theme.colorScheme.error.withValues(alpha: 0.4),
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        ),
      ),
    );
  }

  static String _formatDate(String iso) {
    try {
      final d = DateTime.parse(iso);
      final dd = d.day.toString().padLeft(2, '0');
      final mm = d.month.toString().padLeft(2, '0');
      final yy = d.year.toString();
      return '$dd/$mm/$yy';
    } catch (_) {
      return iso;
    }
  }
}
