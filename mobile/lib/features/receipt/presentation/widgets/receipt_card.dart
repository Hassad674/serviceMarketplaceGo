import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/receipt.dart';

/// Soleil v2 receipt row — a single entry in the "Reçus" list.
///
/// Anatomy:
///   - corail-soft icon disc on the left,
///   - Geist Mono short receipt id pill + optional "Reçu antérieur" badge
///     when the snapshot is unavailable (legacy data),
///   - counterpart name (provider seen by client, client seen by provider,
///     etc.) + relative date in tabac small caps,
///   - Geist Mono semibold amount on the right,
///   - chevron to push the detail screen.
///
/// The widget is intentionally stateless and `const`-friendly — perf
/// budget is 60fps on a screen that scrolls through many rows.
class ReceiptCard extends StatelessWidget {
  const ReceiptCard({
    super.key,
    required this.receipt,
    required this.onTap,
  });

  final Receipt receipt;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _IconDisc(appColors: appColors, colorScheme: colorScheme),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Wrap(
                    spacing: 8,
                    runSpacing: 4,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      _IdPill(id: receipt.id),
                      if (!receipt.snapshotAvailable)
                        _LegacyBadge(appColors: appColors),
                    ],
                  ),
                  const SizedBox(height: 6),
                  Text(
                    _counterpartLabel(receipt),
                    style: SoleilTextStyles.body.copyWith(
                      color: colorScheme.onSurface,
                      fontSize: 13.5,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 2),
                  Text(
                    formatReceiptDate(receipt.createdAt).toUpperCase(),
                    style: SoleilTextStyles.mono.copyWith(
                      color: appColors?.subtleForeground ??
                          colorScheme.onSurfaceVariant,
                      fontSize: 11,
                      letterSpacing: 0.6,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                formatReceiptAmount(receipt.amountCents, receipt.currency),
                style: SoleilTextStyles.mono.copyWith(
                  color: colorScheme.onSurface,
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  letterSpacing: 0.2,
                ),
              ),
            ),
            const SizedBox(width: 4),
            Icon(
              Icons.chevron_right_rounded,
              size: 20,
              color: appColors?.subtleForeground ??
                  colorScheme.onSurfaceVariant,
            ),
          ],
        ),
      ),
    );
  }
}

class _IconDisc extends StatelessWidget {
  const _IconDisc({required this.appColors, required this.colorScheme});

  final AppColors? appColors;
  final ColorScheme colorScheme;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 40,
      height: 40,
      decoration: BoxDecoration(
        color: appColors?.accentSoft ?? colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
      ),
      child: Icon(
        Icons.receipt_long_outlined,
        size: 18,
        color: colorScheme.primary,
      ),
    );
  }
}

class _IdPill extends StatelessWidget {
  const _IdPill({required this.id});

  final String id;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        _shortId(id),
        style: SoleilTextStyles.mono.copyWith(
          color: colorScheme.onSurface,
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.2,
        ),
      ),
    );
  }
}

class _LegacyBadge extends StatelessWidget {
  const _LegacyBadge({required this.appColors});

  final AppColors? appColors;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: appColors?.accentSoft ?? colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
      ),
      child: Text(
        'Reçu antérieur',
        style: SoleilTextStyles.bodyEmphasis.copyWith(
          color: colorScheme.primary,
          fontSize: 11,
        ),
      ),
    );
  }
}

String _shortId(String id) {
  if (id.length <= 10) return id;
  return id.substring(0, 8).toUpperCase();
}

/// Returns the most-relevant counterpart label for the row. The current
/// org never appears (the user already knows which org they belong to);
/// the fallback chain is: provider name > client name > "Transaction".
String _counterpartLabel(Receipt receipt) {
  final providerName = receipt.provider?.name;
  if (providerName != null && providerName.isNotEmpty) {
    return providerName;
  }
  final clientName = receipt.client?.name;
  if (clientName != null && clientName.isNotEmpty) {
    return clientName;
  }
  return 'Transaction';
}

/// Formats the receipt date as "dd/MM/yyyy" — kept identical to the
/// invoice list item for visual parity.
String formatReceiptDate(DateTime d) => DateFormat('dd/MM/yyyy').format(d);

/// Formats the gross amount as "1 234,56 €". Falls back to a manual
/// build when the locale-aware NumberFormat throws (e.g. test
/// environment without intl data).
String formatReceiptAmount(int cents, String currency) {
  final amount = cents / 100;
  final symbol = _currencySymbol(currency);
  try {
    return NumberFormat.currency(
      locale: 'fr_FR',
      symbol: symbol,
      decimalDigits: 2,
    ).format(amount);
  } catch (_) {
    final whole = cents ~/ 100;
    final remainder = (cents.abs() % 100).toString().padLeft(2, '0');
    final sign = cents < 0 ? '-' : '';
    return '$sign$whole,$remainder $symbol';
  }
}

String _currencySymbol(String raw) {
  switch (raw.toLowerCase()) {
    case 'eur':
      return '€';
    case 'usd':
      return r'$';
    case 'gbp':
      return '£';
    default:
      return raw.toUpperCase();
  }
}
