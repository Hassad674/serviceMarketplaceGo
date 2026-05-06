import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
/// ReferralStatusChip renders the referral status as a colour-coded pill.
/// Tones mirror the web feature: amber for pending, emerald for active,
/// rose for failure-terminal, slate for success-terminal.
class ReferralStatusChip extends StatelessWidget {
  const ReferralStatusChip({super.key, required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final palette = _paletteFor(context, status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: palette.background,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: palette.border, width: 1),
      ),
      child: Text(
        _labelFor(status),
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: palette.foreground,
        ),
      ),
    );
  }

  static String _labelFor(String status) {
    switch (status) {
      case 'pending_provider':
        return 'Pending provider';
      case 'pending_referrer':
        return 'Pending referrer';
      case 'pending_client':
        return 'Pending client';
      case 'active':
        return 'Active';
      case 'rejected':
        return 'Rejected';
      case 'expired':
        return 'Expired';
      case 'cancelled':
        return 'Cancelled';
      case 'terminated':
        return 'Terminated';
      default:
        return status;
    }
  }

  static _Palette _paletteFor(BuildContext context, String status) {
    final cs = Theme.of(context).colorScheme;
    final ext = Theme.of(context).extension<AppColors>();
    final amberSoft = ext?.amberSoft ?? cs.secondaryContainer;
    final warning = ext?.warning ?? cs.tertiary;
    final successSoft = ext?.successSoft ?? cs.primaryContainer;
    final success = ext?.success ?? cs.primary;
    final primaryDeep = ext?.primaryDeep ?? cs.error;
    if (status.startsWith('pending_')) {
      return _Palette(
        background: amberSoft,
        border: warning,
        foreground: warning,
      );
    }
    if (status == 'active') {
      return _Palette(
        background: successSoft,
        border: success,
        foreground: success,
      );
    }
    if (status == 'terminated') {
      return _Palette(
        background: cs.surface,
        border: cs.outline,
        foreground: cs.onSurfaceVariant,
      );
    }
    // rejected / expired / cancelled
    return _Palette(
      background: cs.primaryContainer,
      border: cs.primary,
      foreground: primaryDeep,
    );
  }
}

class _Palette {
  const _Palette({
    required this.background,
    required this.border,
    required this.foreground,
  });

  final Color background;
  final Color border;
  final Color foreground;
}
