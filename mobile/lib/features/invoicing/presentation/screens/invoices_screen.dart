import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/invoice.dart';
import '../providers/invoicing_providers.dart';
import '../widgets/current_month_aggregate_card.dart';
import '../widgets/invoice_list_item.dart';

/// Page listing the org's invoice history with the running fee
/// aggregate at the top.
///
/// Pagination: V1 uses an explicit "Voir plus" button at the bottom
/// of the list — simpler than wiring scroll-listening on the
/// SingleChildScrollView and good enough for the typical volume.
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
    return Scaffold(
      appBar: AppBar(
        title: const Text('Mes factures'),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const CurrentMonthAggregateCard(),
              const SizedBox(height: 16),
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

    // Resolve every page that has been requested so far.
    final pages = cursors
        .map((c) => ref.watch(invoicesProvider(c)))
        .toList(growable: false);

    final allLoaded = pages.every((p) => p.hasValue);
    final anyError = pages.any((p) => p.hasError);
    final firstLoading = pages.first.isLoading && !pages.first.hasValue;

    if (firstLoading) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 48),
        child: Center(child: CircularProgressIndicator()),
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
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        children: [
          for (var i = 0; i < items.length; i++) ...[
            if (i > 0)
              Divider(
                height: 1,
                color: theme.dividerColor.withValues(alpha: 0.4),
              ),
            InvoiceListItem(invoice: items[i]),
          ],
          if (nextCursor != null && nextCursor.isNotEmpty)
            Padding(
              padding: const EdgeInsets.all(12),
              child: OutlinedButton(
                onPressed: isFetchingMore
                    ? null
                    : () => onLoadMore(nextCursor!),
                child: isFetchingMore
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Voir plus'),
              ),
            ),
        ],
      ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(32),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor,
          style: BorderStyle.solid,
        ),
      ),
      child: Column(
        children: [
          Icon(
            Icons.description_outlined,
            size: 40,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.3),
          ),
          const SizedBox(height: 12),
          Text(
            'Aucune facture pour le moment',
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            'Tes factures apparaîtront ici dès qu\'une opération '
            'facturable sera enregistrée.',
            textAlign: TextAlign.center,
            style: theme.textTheme.bodySmall,
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
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        children: [
          Text(
            'Impossible de charger les factures.',
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 8),
          OutlinedButton(
            onPressed: onRetry,
            child: const Text('Réessayer'),
          ),
        ],
      ),
    );
  }
}
