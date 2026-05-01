import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/portfolio_item.dart';
import '../providers/portfolio_provider.dart';
import 'grid/portfolio_card.dart';
import 'grid/portfolio_empty_state.dart';
import 'grid/portfolio_section_wrapper.dart';
import 'grid/portfolio_skeleton.dart';
import 'portfolio_detail_sheet.dart';
import 'portfolio_form_sheet.dart';

const int _kMaxItems = 30;

/// Displays a grid of portfolio items for a given user.
///
/// Used on both own profile (edit mode) and public profiles (read-only).
class PortfolioGridWidget extends ConsumerWidget {
  const PortfolioGridWidget({
    super.key,
    required this.orgId,
    this.readOnly = true,
  });

  final String orgId;
  final bool readOnly;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncItems = ref.watch(portfolioByOrgProvider(orgId));

    return asyncItems.when(
      data: (items) {
        if (items.isEmpty) {
          if (readOnly) return const SizedBox.shrink();
          return PortfolioSectionWrapper(
            count: 0,
            onAdd: () => _openForm(context, ref, null, 0),
            child: PortfolioEmptyState(
              onCreate: () => _openForm(context, ref, null, 0),
            ),
          );
        }
        return PortfolioSectionWrapper(
          count: items.length,
          onAdd: readOnly || items.length >= _kMaxItems
              ? null
              : () => _openForm(context, ref, null, items.length),
          child: _PortfolioGrid(
            items: items,
            readOnly: readOnly,
            orgId: orgId,
            onEdit: (item) => _openForm(context, ref, item, items.length),
            onDelete: (item) => _confirmDelete(context, ref, item),
          ),
        );
      },
      loading: () => const PortfolioSkeleton(),
      error: (_, __) => const SizedBox.shrink(),
    );
  }

  void _openForm(
    BuildContext context,
    WidgetRef ref,
    PortfolioItem? item,
    int nextPosition,
  ) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      useSafeArea: true,
      builder: (_) => PortfolioFormSheet(
        orgId: orgId,
        item: item,
        nextPosition: nextPosition,
      ),
    );
  }

  Future<void> _confirmDelete(
    BuildContext context,
    WidgetRef ref,
    PortfolioItem item,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete project?'),
        content: Text('Delete "${item.title}"? This cannot be undone.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: TextButton.styleFrom(foregroundColor: Colors.red),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    await ref
        .read(portfolioMutationProvider.notifier)
        .deleteItem(orgId: orgId, id: item.id);
  }
}

/// Inner grid that lays out the cards.
class _PortfolioGrid extends StatelessWidget {
  const _PortfolioGrid({
    required this.items,
    required this.readOnly,
    required this.orgId,
    required this.onEdit,
    required this.onDelete,
  });

  final List<PortfolioItem> items;
  final bool readOnly;
  final String orgId;
  final void Function(PortfolioItem) onEdit;
  final void Function(PortfolioItem) onDelete;

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
        childAspectRatio: 0.78,
      ),
      itemCount: items.length,
      itemBuilder: (context, index) {
        final item = items[index];
        return PortfolioCard(
          item: item,
          readOnly: readOnly,
          onTap: () => _showDetail(context, item),
          onEdit: () => onEdit(item),
          onDelete: () => onDelete(item),
        );
      },
    );
  }

  void _showDetail(BuildContext context, PortfolioItem item) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      useSafeArea: true,
      builder: (_) => PortfolioDetailSheet(item: item),
    );
  }
}
