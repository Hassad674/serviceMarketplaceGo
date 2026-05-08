import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import 'billing_profile_form.dart';

/// Bottom-sheet wrapper that embeds the canonical [BillingProfileForm]
/// directly inline — used by the proposal payment flow when the
/// backend gate (412 + `billing_profile_incomplete`) refuses to issue
/// a PaymentIntent because the client organization has not yet filled
/// in its billing identity.
///
/// Distinct from [showBillingProfileCompletionModal] which redirects
/// the user to `/settings/billing-profile`. The inline variant keeps
/// the user on the payment screen so the moment they save, the parent
/// screen can retry the gated action without an extra navigation hop.
///
/// Returns `true` when the user saved successfully and the resulting
/// profile passes server-side completeness; `false` (or null) when the
/// user dismissed the sheet without a successful save.
Future<bool?> showBillingProfileInlineSheet(BuildContext context) {
  return showModalBottomSheet<bool>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(
        top: Radius.circular(AppTheme.radius2xl),
      ),
    ),
    builder: (sheetContext) => const _BillingProfileInlineSheet(),
  );
}

class _BillingProfileInlineSheet extends StatelessWidget {
  const _BillingProfileInlineSheet();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    // The sheet is height-constrained to the device viewport so a long
    // form scrolls inside the sheet rather than pushing the keyboard
    // off-screen. Padding tracks the bottom inset so an open keyboard
    // never hides the Save button.
    final viewInsetsBottom = MediaQuery.of(context).viewInsets.bottom;
    return Padding(
      padding: EdgeInsets.fromLTRB(20, 8, 20, 16 + viewInsetsBottom),
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(context).size.height * 0.92,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Center(
              child: Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: (appColors?.border ?? theme.dividerColor)
                      .withValues(alpha: 0.8),
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            const SizedBox(height: 20),
            Text(
              l10n.billingProfileInlineSheetTitle,
              style: SoleilTextStyles.titleLarge.copyWith(
                color: colorScheme.onSurface,
              ),
            ),
            const SizedBox(height: 8),
            Text(
              l10n.billingProfileInlineSheetSubtitle,
              style: SoleilTextStyles.body.copyWith(
                color: colorScheme.onSurfaceVariant,
                fontSize: 13,
              ),
            ),
            const SizedBox(height: 16),
            Expanded(
              child: SingleChildScrollView(
                child: BillingProfileForm(
                  // onSaved fires AFTER a successful update where the
                  // resulting profile is server-side complete. Pop the
                  // sheet with `true` so the caller can chain the retry
                  // of the gated action.
                  onSaved: () => Navigator.of(context).pop(true),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
