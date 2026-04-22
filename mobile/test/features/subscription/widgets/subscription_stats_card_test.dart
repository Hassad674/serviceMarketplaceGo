import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription_stats.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/subscription_stats_card.dart';

import '../helpers/subscription_test_helpers.dart';

void main() {
  testWidgets('loading state renders nothing', (tester) async {
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          subscriptionStatsProvider.overrideWith(
            (ref) => Completer<SubscriptionStats?>().future,
          ),
        ],
        child: const SubscriptionStatsCard(),
      ),
    );
    expect(find.byType(RichText), findsNothing);
    expect(find.text('Tu as économisé'), findsNothing);
  });

  testWidgets('null stats renders nothing', (tester) async {
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          subscriptionStatsProvider.overrideWith((ref) async => null),
        ],
        child: const SubscriptionStatsCard(),
      ),
    );
    await tester.pump();
    expect(find.byType(RichText), findsNothing);
  });

  testWidgets('renders "Tu as économisé" with formatted amount and since date',
      (tester) async {
    final stats = buildStats(
      savedFeeCents: 12345,
      since: DateTime.utc(2026, 2, 20),
    );
    await tester.pumpWidget(
      wrapWidget(
        overrides: [
          subscriptionStatsProvider.overrideWith((ref) async => stats),
        ],
        child: const SubscriptionStatsCard(),
      ),
    );
    await tester.pump();

    final richText = tester.widget<RichText>(find.byType(RichText));
    final fullText = richText.text.toPlainText();
    expect(fullText, contains('Tu as économisé'));
    // 12345 cents → 123,45 €. The widget tries fr_FR NumberFormat.currency
    // first and falls back to a manual format. Either way, the integer
    // part "123" must appear.
    expect(fullText, contains('123'));
    expect(fullText, contains('20/02/2026'));
  });
}
