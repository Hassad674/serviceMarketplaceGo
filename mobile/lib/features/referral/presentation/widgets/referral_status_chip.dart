import 'package:flutter/material.dart';

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
        background: Color(0xFFFEF3C7),
        border: Color(0xFFFCD34D),
        foreground: Color(0xFFB45309),
      );
    }
    if (status == 'active') {
      return const _Palette(
        background: Color(0xFFD1FAE5),
        border: Color(0xFF6EE7B7),
        foreground: Color(0xFF047857),
      );
    }
    if (status == 'terminated') {
      return const _Palette(
        background: Color(0xFFF1F5F9),
        border: Color(0xFFCBD5E1),
        foreground: Color(0xFF334155),
      );
    }
    // rejected / expired / cancelled
    return const _Palette(
      background: Color(0xFFFFE4E6),
      border: Color(0xFFFDA4AF),
      foreground: Color(0xFFBE123C),
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
