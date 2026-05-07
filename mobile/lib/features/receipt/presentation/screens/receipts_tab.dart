import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/receipt.dart';
import '../providers/receipt_providers.dart';
import '../widgets/receipt_card.dart';
import 'receipt_detail_screen.dart';

/// Soleil v2 "Reçus" tab — displayed inside the invoices Scaffold's
/// TabBarView. Mirrors the invoices tab pagination pattern (cursor
/// list + family-scoped FutureProviders).
///
/// Behavior:
///   - First load shows a centered spinner.
///   - Empty list shows the editorial empty state.
///   - Errors fold to a retry pill that invalidates every cached page.
///   - "Voir plus" pill appears only when the latest page returned a
///     non-null cursor.
class ReceiptsTab extends ConsumerStatefulWidget {
  const ReceiptsTab({super.key});

  @override
  ConsumerState<ReceiptsTab> createState() => _ReceiptsTabState();
}

class _ReceiptsTabState extends ConsumerState<ReceiptsTab> {
  // List of cursors fetched so far. The first entry is `null` (initial
  // page); each successful page push appends its `nextCursor` (when
  // non-null) so subsequent watches resolve all cached pages.
  final List<String?> _cursors = [null];

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
      child: _PaginatedReceiptList(
        cursors: _cursors,
        onLoadMore: (cursor) {
          setState(() => _cursors.add(cursor));
        },
      ),
    );
  }
}

class _PaginatedReceiptList extends ConsumerWidget {
  const _PaginatedReceiptList({
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

    final pages = cursors
        .map((c) => ref.watch(receiptsProvider(c)))
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
      return _ReceiptsErrorState(
        onRetry: () {
          for (final c in cursors) {
            ref.invalidate(receiptsProvider(c));
          }
        },
      );
    }

    final items = <Receipt>[];
    String? nextCursor;
    for (final p in pages) {
      final page = p.valueOrNull;
      if (page == null) continue;
      items.addAll(page.data);
      nextCursor = page.nextCursor;
    }

    if (items.isEmpty) {
      return const _ReceiptsEmptyState();
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
            ReceiptCard(
              receipt: items[i],
              onTap: () => _openDetail(context, items[i].id),
            ),
          ],
          if (nextCursor != null && nextCursor.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
              child: _LoadMoreReceiptsButton(
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

  void _openDetail(BuildContext context, String id) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => ReceiptDetailScreen(receiptId: id),
      ),
    );
  }
}

class _LoadMoreReceiptsButton extends StatelessWidget {
  const _LoadMoreReceiptsButton({
    required this.loading,
    required this.onPressed,
  });

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
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
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

class _ReceiptsEmptyState extends StatelessWidget {
  const _ReceiptsEmptyState();

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
              Icons.receipt_long_outlined,
              size: 24,
              color: colorScheme.primary,
            ),
          ),
          const SizedBox(height: 16),
          Text(
            "Aucun reçu pour l'instant",
            textAlign: TextAlign.center,
            style: SoleilTextStyles.titleMedium.copyWith(
              color: colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Tes paiements génèrent automatiquement des reçus que tu '
            'pourras retrouver ici dès la première transaction.',
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

class _ReceiptsErrorState extends StatelessWidget {
  const _ReceiptsErrorState({required this.onRetry});

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
            'Impossible de charger les reçus.',
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
