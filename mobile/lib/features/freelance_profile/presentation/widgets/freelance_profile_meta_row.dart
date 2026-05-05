import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/money_format.dart';
import '../../../../shared/widgets/availability_pill.dart';
import '../../domain/entities/freelance_pricing.dart';

/// Compact horizontal meta strip rendered under the avatar + name in
/// the freelance profile header. Surfaces only the fields the entity
/// actually carries — rating and years-of-experience are skipped on
/// purpose because the freelance entity does not expose them yet.
///
/// Currently shown:
///   · day rate ("/jour" — only when [pricing] is daily)
///   · availability dot + label (always — repositioned from the
///     legacy bottom-pill spot to keep the meta row self-contained)
class FreelanceProfileMetaRow extends StatelessWidget {
  const FreelanceProfileMetaRow({
    super.key,
    required this.pricing,
    required this.availabilityWireValue,
    required this.locale,
  });

  final FreelancePricing? pricing;
  final String availabilityWireValue;
  final String locale;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final children = <Widget>[];

    final dailyLabel = _dailyRateLabel(l10n);
    if (dailyLabel != null) {
      children.add(
        _MetaItem(
          icon: Icons.payments_outlined,
          label: dailyLabel,
          textStyle: SoleilTextStyles.caption.copyWith(
            color: theme.colorScheme.onSurface,
            fontWeight: FontWeight.w600,
          ),
          iconColor: theme.colorScheme.primary,
        ),
      );
    }

    children.add(
      AvailabilityPill(
        compact: true,
        wireValue: availabilityWireValue,
        label: _availabilityLabel(l10n, availabilityWireValue),
      ),
    );

    return Wrap(
      alignment: WrapAlignment.center,
      spacing: 8,
      runSpacing: 6,
      children: children,
    );
  }

  /// Returns the formatted "/jour" label only when the freelance has
  /// declared a daily-type pricing row. Hourly / project pricing is
  /// out of scope for this meta strip.
  String? _dailyRateLabel(AppLocalizations l10n) {
    final p = pricing;
    if (p == null || p.type != FreelancePricingType.daily) return null;
    if (p.minAmount <= 0) return null;
    final amount = formatMoney(p.minAmount, p.currency, locale);
    return l10n.freelanceMetaPerDay(amount);
  }

  String _availabilityLabel(AppLocalizations l10n, String wire) {
    switch (wire) {
      case 'available_soon':
        return l10n.tier1AvailabilityStatusAvailableSoon;
      case 'not_available':
        return l10n.tier1AvailabilityStatusNotAvailable;
      case 'available_now':
      default:
        return l10n.tier1AvailabilityStatusAvailableNow;
    }
  }
}

class _MetaItem extends StatelessWidget {
  const _MetaItem({
    required this.icon,
    required this.label,
    required this.textStyle,
    required this.iconColor,
  });

  final IconData icon;
  final String label;
  final TextStyle textStyle;
  final Color iconColor;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 14, color: iconColor),
        const SizedBox(width: 4),
        Text(label, style: textStyle),
      ],
    );
  }
}
