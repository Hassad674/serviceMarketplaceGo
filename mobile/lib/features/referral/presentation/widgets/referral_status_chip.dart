import 'package:flutter/material.dart';
import '../../../../core/theme/app_palette.dart';

/// ReferralStatusChip renders the referral status as a colour-coded pill.
/// Tones mirror the web feature: amber for pending, emerald for active,
/// rose for failure-terminal, slate for success-terminal.
class ReferralStatusChip extends StatelessWidget {
  const ReferralStatusChip({super.key, required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final palette = _paletteFor(status);
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

  static _Palette _paletteFor(String status) {
    if (status.startsWith('pending_')) {
      return const _Palette(
        background: AppPalette.amber100,
        border: AppPalette.amber300,
        foreground: AppPalette.amber700,
      );
    }
    if (status == 'active') {
      return const _Palette(
        background: AppPalette.emerald100,
        border: AppPalette.emerald300,
        foreground: AppPalette.emerald700,
      );
    }
    if (status == 'terminated') {
      return const _Palette(
        background: AppPalette.slate100,
        border: AppPalette.slate300,
        foreground: AppPalette.slate700,
      );
    }
    // rejected / expired / cancelled
    return const _Palette(
      background: AppPalette.rose100,
      border: AppPalette.rose300,
      foreground: AppPalette.rose700,
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
