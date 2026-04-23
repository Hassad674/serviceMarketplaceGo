import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/subscription/domain/entities/subscription.dart';
import 'package:marketplace_mobile/features/subscription/presentation/providers/subscription_providers.dart';
import 'package:marketplace_mobile/features/subscription/presentation/widgets/subscription_badge.dart';

import '../helpers/subscription_test_helpers.dart';

/// Overrides [subscriptionProvider] to immediately resolve with [value].
Override _subOverride(Subscription? value) {
  return subscriptionProvider.overrideWith((ref) async => value);
}

/// Keeps the provider forever loading.
Override _loadingSubOverride() {
  return subscriptionProvider.overrideWith(
    (ref) => Completer<Subscription?>().future,
  );
}

void main() {
  testWidgets('loading state renders the skeleton pill (no label)',
      (tester) async {
    await tester.pumpWidget(
      wrapWidget(
        overrides: [_loadingSubOverride()],
        child: SubscriptionBadge(onTap: () {}),
      ),
    );
    // First frame: the FutureProvider is still resolving — no label visible.
    expect(find.text('Passer Premium'), findsNothing);
    expect(find.text("Gérer l'abonnement"), findsNothing);
    // The skeleton is a plain Container without an InkWell / Semantics label.
    expect(find.byType(InkWell), findsNothing);
  });

  testWidgets('null sub renders "Passer Premium"', (tester) async {
    await tester.pumpWidget(
      wrapWidget(
        overrides: [_subOverride(null)],
        child: SubscriptionBadge(onTap: () {}),
      ),
    );
    await tester.pump();
    expect(find.text('Passer Premium'), findsOneWidget);
  });

  testWidgets('past_due status renders the orange "Paiement échoué" label',
      (tester) async {
    final sub = buildSubscription(status: SubscriptionStatus.pastDue);
    await tester.pumpWidget(
      wrapWidget(
        overrides: [_subOverride(sub)],
        child: SubscriptionBadge(onTap: () {}),
      ),
    );
    await tester.pump();
    expect(find.text('Paiement échoué · gérer'), findsOneWidget);

    // Verify the orange palette was picked.
    final container = tester.widget<Container>(
      find
          .descendant(of: find.byType(InkWell), matching: find.byType(Container))
          .first,
    );
    final deco = container.decoration as BoxDecoration;
    expect(deco.color, const Color(0xFFFFEDD5));
  });

  testWidgets('active sub renders "Gérer l\'abonnement"', (tester) async {
    final sub = buildSubscription();
    await tester.pumpWidget(
      wrapWidget(
        overrides: [_subOverride(sub)],
        child: SubscriptionBadge(onTap: () {}),
      ),
    );
    await tester.pump();
    expect(find.text("Gérer l'abonnement"), findsOneWidget);
  });

  testWidgets('tap invokes the onTap callback', (tester) async {
    var tapped = 0;
    await tester.pumpWidget(
      wrapWidget(
        overrides: [_subOverride(null)],
        child: SubscriptionBadge(onTap: () => tapped++),
      ),
    );
    await tester.pump();
    await tester.tap(find.byType(InkWell));
    await tester.pump();
    expect(tapped, 1);
  });
}
