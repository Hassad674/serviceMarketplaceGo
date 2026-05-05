import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/invoice.dart';
import '../providers/invoicing_providers.dart';
import '../widgets/current_month_aggregate_card.dart';
import '../widgets/invoice_list_item.dart';

/// M-15 — Soleil v2 invoices screen.
///
/// Editorial header (corail mono eyebrow + Fraunces italic accent +
/// tabac subtitle) followed by the running monthly aggregate card and
/// a Soleil ivoire list of invoices. Pagination keeps the explicit
/// "Voir plus" pill the existing test pins.
///
/// Behavior preservation:
///   - Riverpod providers (`invoicesProvider`, `currentMonthProvider`)
///     unchanged, only the visual chrome around them is ported.
///   - Cursors / load-more flow / error+empty branches identical to
///     the legacy implementation.
class InvoicesScreen extends ConsumerStatefulWidget {
  const InvoicesScreen({super.key});

  @override
  ConsumerState<InvoicesScreen> createState() => _InvoicesScreenState();
}

class _InvoicesScreenState extends ConsumerState<InvoicesScreen> {
  // Cursors fetched so far. The first entry is `null` (initial page);
  // each successful page push appends its `nextCursor` (when non-null)
  // so subsequent watches resolve all cached pages.
  final List<String?> _cursors = [null];

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Scaffold(
      backgroundColor: colorScheme.surface,
      appBar: AppBar(
        backgroundColor: colorScheme.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: Text(
          'Factures',
          style: SoleilTextStyles.titleMedium.copyWith(
            color: colorScheme.onSurface,
          ),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const _EditorialHeader(),
              const SizedBox(height: 20),
              const CurrentMonthAggregateCard(),
              const SizedBox(height: 20),
              _PaginatedList(
                cursors: _cursors,
                onLoadMore: (cursor) {
                  setState(() => _cursors.add(cursor));
                },
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _EditorialHeader extends StatelessWidget {
  const _EditorialHeader();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final primary = colorScheme.primary;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'ATELIER · FACTURES',
          style: SoleilTextStyles.mono.copyWith(
            color: primary,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 1.4,
          ),
        ),
        const SizedBox(height: 8),
        RichText(
          text: TextSpan(
            style: SoleilTextStyles.headlineLarge.copyWith(
              color: colorScheme.onSurface,
            ),
            children: [
              const TextSpan(text: 'Tes '),
              TextSpan(
                text: 'factures et reçus.',
                style: SoleilTextStyles.headlineLarge.copyWith(
                  color: primary,
                  fontStyle: FontStyle.italic,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 8),
        Text(
          "Retrouve les factures que la plateforme émet à ton organisation : "
          "abonnement Premium, commissions mensuelles et avoirs éventuels.",
          style: SoleilTextStyles.bodyLarge.copyWith(
            color: colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _PaginatedList extends ConsumerWidget {
  const _PaginatedList({
    required this.cursors,
    required this.onLoadMore,
  });

  final List<String?> cursors;
  final ValueChanged<String> onLoadMore;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();

    // Resolve every page that has been requested so far.
    final pages = cursors
        .map((c) => ref.watch(invoicesProvider(c)))
        .toList(growable: false);

    final allLoaded = pages.every((p) => p.hasValue);
    final anyError = pages.any((p) => p.hasError);
    final firstLoading = pages.first.isLoading && !pages.first.hasValue;

    if (firstLoading) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 48),
        child: Center(
          child: CircularProgressIndicator(color: colorScheme.primary),
        ),
      );
    }

    if (anyError && !allLoaded) {
      return _ErrorState(
        onRetry: () {
          for (final c in cursors) {
            ref.invalidate(invoicesProvider(c));
          }
        },
      );
    }

    final items = <Invoice>[];
    String? nextCursor;
    for (final p in pages) {
      final page = p.valueOrNull;
      if (page == null) continue;
      items.addAll(page.data);
      nextCursor = page.nextCursor;
    }

    if (items.isEmpty) {
      return const _EmptyState();
    }

    final isFetchingMore = pages.last.isLoading && pages.length > 1;

    return Container(
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          for (var i = 0; i < items.length; i++) ...[
            if (i > 0)
              Divider(
                height: 1,
                color: (appColors?.border ?? theme.dividerColor)
                    .withValues(alpha: 0.7),
              ),
            InvoiceListItem(invoice: items[i]),
          ],
          if (nextCursor != null && nextCursor.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
              child: _LoadMoreButton(
                loading: isFetchingMore,
                onPressed: isFetchingMore
                    ? null
                    : () => onLoadMore(nextCursor!),
              ),
            ),
        ],
      ),
    );
  }
}

class _LoadMoreButton extends StatelessWidget {
  const _LoadMoreButton({required this.loading, required this.onPressed});

  final bool loading;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Center(
      child: OutlinedButton(
        onPressed: onPressed,
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(0, 40),
          padding:
              const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
          foregroundColor: colorScheme.onSurface,
          side: BorderSide(
            color: appColors?.borderStrong ?? theme.dividerColor,
          ),
          shape: const StadiumBorder(),
          textStyle: SoleilTextStyles.button,
        ),
        child: loading
            ? SizedBox(
                width: 14,
                height: 14,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  color: colorScheme.primary,
                ),
              )
            : const Text('Voir plus'),
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.fromLTRB(28, 40, 28, 44),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        children: [
          Container(
            width: 56,
            height: 56,
            decoration: BoxDecoration(
              color: appColors?.accentSoft ?? colorScheme.primaryContainer,
              borderRadius: BorderRadius.circular(AppTheme.radiusFull),
            ),
            child: Icon(
              Icons.description_outlined,
              size: 24,
              color: colorScheme.primary,
            ),
          ),
          const SizedBox(height: 16),
          Text(
            'Aucune facture archivée',
            textAlign: TextAlign.center,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'La facture consolidée des commissions du mois en cours sera '
            'émise automatiquement le 1er du mois suivant. Les factures '
            "d'abonnement Premium apparaîtront dès le premier paiement.",
            textAlign: TextAlign.center,
            style: SoleilTextStyles.bodyLarge.copyWith(
              color: colorScheme.onSurfaceVariant,
              fontStyle: FontStyle.italic,
            ),
          ),
        ],
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'Impossible de charger les factures.',
            style: SoleilTextStyles.bodyLarge.copyWith(
              color: colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 12),
          OutlinedButton(
            onPressed: onRetry,
            style: OutlinedButton.styleFrom(
              minimumSize: const Size(double.infinity, 44),
              foregroundColor: colorScheme.onSurface,
              side: BorderSide(
                color: appColors?.borderStrong ?? theme.dividerColor,
              ),
              shape: const StadiumBorder(),
              textStyle: SoleilTextStyles.button,
            ),
            child: const Text('Réessayer'),
          ),
        ],
      ),
    );
  }
}
