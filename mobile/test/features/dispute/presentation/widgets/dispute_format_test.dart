import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/dispute/presentation/widgets/dispute_format.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Future<AppLocalizations> _loadEn() async {
  return AppLocalizations.delegate.load(const Locale('en'));
}

/// Wrap a color resolver in a real MaterialApp so it can read theme tokens.
Future<Color> _resolveColor(
  WidgetTester tester,
  Color Function(BuildContext) builder,
) async {
  late Color resolved;
  await tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.light,
      home: Builder(
        builder: (context) {
          resolved = builder(context);
          return const SizedBox.shrink();
        },
      ),
    ),
  );
  return resolved;
}

void main() {
  group('disputeStatusColor', () {
    testWidgets('returns warning for open and negotiation', (tester) async {
      final open = await _resolveColor(
        tester,
        (ctx) => disputeStatusColor(ctx, 'open'),
      );
      final negotiation = await _resolveColor(
        tester,
        (ctx) => disputeStatusColor(ctx, 'negotiation'),
      );
      expect(open, AppTheme.light.extension<AppColors>()!.warning);
      expect(negotiation, AppTheme.light.extension<AppColors>()!.warning);
    });

    testWidgets('returns error for escalated', (tester) async {
      final c = await _resolveColor(
        tester,
        (ctx) => disputeStatusColor(ctx, 'escalated'),
      );
      expect(c, AppTheme.light.colorScheme.error);
    });

    testWidgets('returns success for resolved', (tester) async {
      final c = await _resolveColor(
        tester,
        (ctx) => disputeStatusColor(ctx, 'resolved'),
      );
      expect(c, AppTheme.light.extension<AppColors>()!.success);
    });

    testWidgets('returns onSurfaceVariant for cancelled', (tester) async {
      final c = await _resolveColor(
        tester,
        (ctx) => disputeStatusColor(ctx, 'cancelled'),
      );
      expect(c, AppTheme.light.colorScheme.onSurfaceVariant);
    });

    testWidgets('falls back to warning for unknown', (tester) async {
      final c = await _resolveColor(
        tester,
        (ctx) => disputeStatusColor(ctx, 'mystery'),
      );
      expect(c, AppTheme.light.extension<AppColors>()!.warning);
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
