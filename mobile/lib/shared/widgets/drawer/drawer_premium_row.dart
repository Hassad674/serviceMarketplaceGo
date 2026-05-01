import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/router/app_router.dart';
import '../../../features/subscription/presentation/providers/subscription_providers.dart';
import '../../../features/subscription/presentation/widgets/manage_bottom_sheet.dart';
import '../../../features/subscription/presentation/widgets/subscription_badge.dart';

/// Premium pill rendered under the drawer header.
///
/// Routes the user to the manage bottom-sheet when the org is already
/// subscribed, or to the `/pricing` screen otherwise. Hidden for
/// enterprise (buyer side — they don't subscribe).
class DrawerPremiumRow extends ConsumerWidget {
  const DrawerPremiumRow({super.key, required this.role});

  final String role;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
      child: Align(
        alignment: Alignment.centerLeft,
        child: SubscriptionBadge(
          onTap: () => _onTap(context, ref),
        ),
      ),
    );
  }

  void _onTap(BuildContext context, WidgetRef ref) {
    // Close the drawer so the next surface (sheet or page) has the
    // full viewport. Navigator.pop() is safe here — Drawer is always
    // on the stack when its content is tapped.
    Navigator.of(context).pop();
    final current = ref.read(subscriptionProvider).valueOrNull;
    if (current != null) {
      // Give the pop animation a frame to settle before pushing the
      // modal — otherwise Flutter complains about lost focus traversal.
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!context.mounted) return;
        showManageBottomSheet(context);
      });
      return;
    }
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!context.mounted) return;
      context.push(RoutePaths.pricing);
    });
  }
}
