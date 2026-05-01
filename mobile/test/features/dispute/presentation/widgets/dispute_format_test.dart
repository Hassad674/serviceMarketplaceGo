import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/dispute/presentation/widgets/dispute_format.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Future<AppLocalizations> _loadEn() async {
  return AppLocalizations.delegate.load(const Locale('en'));
}

void main() {
  group('disputeStatusColor', () {
    test('returns orange for open and negotiation', () {
      expect(disputeStatusColor('open'), const Color(0xFFEA580C));
      expect(disputeStatusColor('negotiation'), const Color(0xFFEA580C));
    });

    test('returns red for escalated', () {
      expect(disputeStatusColor('escalated'), const Color(0xFFDC2626));
    });

    test('returns green for resolved', () {
      expect(disputeStatusColor('resolved'), const Color(0xFF16A34A));
    });

    test('returns slate for cancelled', () {
      expect(disputeStatusColor('cancelled'), const Color(0xFF64748B));
    });

    test('falls back to orange for unknown', () {
      expect(disputeStatusColor('mystery'), const Color(0xFFEA580C));
    });
  });

  group('disputeStatusIcon', () {
    test('returns the right icon per status', () {
      expect(disputeStatusIcon('resolved'), Icons.check_circle_outline);
      expect(disputeStatusIcon('cancelled'), Icons.cancel_outlined);
      expect(disputeStatusIcon('escalated'), Icons.shield_outlined);
      expect(disputeStatusIcon('open'), Icons.warning_amber_rounded);
    });
  });

  group('disputeStatusLabel', () {
    test('localizes known statuses', () async {
      final l10n = await _loadEn();
      expect(disputeStatusLabel(l10n, 'open'), l10n.disputeStatusOpen);
      expect(
        disputeStatusLabel(l10n, 'escalated'),
        l10n.disputeStatusEscalated,
      );
      expect(disputeStatusLabel(l10n, 'resolved'), l10n.disputeStatusResolved);
    });

    test('falls back to raw status on unknown', () async {
      final l10n = await _loadEn();
      expect(disputeStatusLabel(l10n, 'mystery'), 'mystery');
    });
  });

  group('disputeReasonLabel', () {
    test('localizes known reasons', () async {
      final l10n = await _loadEn();
      expect(
        disputeReasonLabel(l10n, 'work_not_conforming'),
        l10n.disputeReasonWorkNotConforming,
      );
      expect(
        disputeReasonLabel(l10n, 'harassment'),
        l10n.disputeReasonHarassment,
      );
    });

    test('falls back to disputeReasonOther on unknown', () async {
      final l10n = await _loadEn();
      expect(
        disputeReasonLabel(l10n, 'mystery'),
        l10n.disputeReasonOther,
      );
    });
  });

  group('formatEur', () {
    test('formats amounts in centimes to euros', () {
      // The exact formatting depends on intl's locale data; assert the
      // numeric portion + the euro symbol.
      final formatted = formatEur(12345);
      expect(formatted, contains('€'));
      expect(formatted, contains('123'));
    });

    test('handles zero', () {
      final formatted = formatEur(0);
      expect(formatted, contains('0'));
      expect(formatted, contains('€'));
    });
  });

  group('daysSinceCreation', () {
    test('returns positive int for past dates', () {
      final pastDate = DateTime.now()
          .subtract(const Duration(days: 3))
          .toIso8601String();
      expect(daysSinceCreation(pastDate), greaterThanOrEqualTo(2));
    });

    test('returns 0 for malformed strings', () {
      expect(daysSinceCreation('not-a-date'), 0);
    });
  });
}
