import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/portfolio_item.dart';
import '../providers/portfolio_provider.dart';
import 'portfolio_detail_sheet.dart';
import 'portfolio_form_sheet.dart';
import 'portfolio_video_thumbnail.dart';

const int _kMaxItems = 30;

/// Displays a grid of portfolio items for a given user.
///
/// Used on both own profile (edit mode) and public profiles (read-only).
class PortfolioGridWidget extends ConsumerWidget {
  final String orgId;
  final bool readOnly;

  const PortfolioGridWidget({
    super.key,
    required this.orgId,
    this.readOnly = true,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncItems = ref.watch(portfolioByOrgProvider(orgId));

    return asyncItems.when(
      data: (items) {
        if (items.isEmpty) {
          if (readOnly) return const SizedBox.shrink();
          return _SectionWrapper(
            count: 0,
            onAdd: () => _openForm(context, ref, null, 0),
            child: _EmptyState(
              onCreate: () => _openForm(context, ref, null, 0),
            ),
          );
        }
        return _SectionWrapper(
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
      loading: () => const _PortfolioSkeleton(),
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

// ---------------------------------------------------------------------------
// Section wrapper (header + content)
// ---------------------------------------------------------------------------

class _SectionWrapper extends StatelessWidget {
  final int count;
  final VoidCallback? onAdd;
  final Widget child;

  const _SectionWrapper({
    required this.count,
    required this.onAdd,
    required this.child,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.cardColor,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 36,
                height: 36,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(10),
                  gradient: const LinearGradient(
                    colors: [Color(0xFFFFE4E6), Color(0xFFFEF2F2)],
                  ),
                ),
                child: const Icon(
                  Icons.work_outline,
                  size: 18,
                  color: Color(0xFFE11D48),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Portfolio',
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    Text(
                      count == 0
                          ? 'Showcase your best work'
                          : '$count ${count > 1 ? 'projects' : 'project'}',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              if (onAdd != null && count > 0)
                FilledButton.icon(
                  onPressed: onAdd,
                  icon: const Icon(Icons.add, size: 16),
                  label: const Text('Add'),
                  style: FilledButton.styleFrom(
                    backgroundColor: const Color(0xFFE11D48),
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 8,
                    ),
                    minimumSize: const Size(0, 32),
                    visualDensity: VisualDensity.compact,
                  ),
                ),
            ],
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Empty state with CTA
// ---------------------------------------------------------------------------

class _EmptyState extends StatelessWidget {
  final VoidCallback onCreate;

  const _EmptyState({required this.onCreate});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 28),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(16),
        border: Border.all(
          color: const Color(0xFFFECDD3),
          width: 2,
          style: BorderStyle.solid,
        ),
        gradient: const LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            Color(0xFFFFF1F2),
            Color(0xFFFFFFFF),
            Color(0xFFFAF5FF),
          ],
        ),
      ),
      child: Column(
        children: [
          Container(
            width: 56,
            height: 56,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(16),
              gradient: const LinearGradient(
                colors: [Color(0xFFF43F5E), Color(0xFFE11D48)],
              ),
              boxShadow: [
                BoxShadow(
                  color: const Color(0xFFE11D48).withValues(alpha: 0.3),
                  blurRadius: 16,
                  offset: const Offset(0, 6),
                ),
              ],
            ),
            child: const Icon(
              Icons.add_photo_alternate,
              color: Colors.white,
              size: 26,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            'No projects yet',
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            'Build trust with clients by showcasing your best work.',
            textAlign: TextAlign.center,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: onCreate,
            icon: const Icon(Icons.auto_awesome, size: 16),
            label: const Text('Add your first project'),
            style: FilledButton.styleFrom(
              backgroundColor: const Color(0xFFE11D48),
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 10,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Grid of cards
// ---------------------------------------------------------------------------

class _PortfolioGrid extends StatelessWidget {
  final List<PortfolioItem> items;
  final bool readOnly;
  final String orgId;
  final void Function(PortfolioItem) onEdit;
  final void Function(PortfolioItem) onDelete;

  const _PortfolioGrid({
    required this.items,
    required this.readOnly,
    required this.orgId,
    required this.onEdit,
    required this.onDelete,
  });

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
        return _PortfolioCard(
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

// ---------------------------------------------------------------------------
// Single card
// ---------------------------------------------------------------------------

class _PortfolioCard extends StatelessWidget {
  final PortfolioItem item;
  final bool readOnly;
  final VoidCallback onTap;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  const _PortfolioCard({
    required this.item,
    required this.readOnly,
    required this.onTap,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final sortedMedia = [...item.media]
      ..sort((a, b) => a.position.compareTo(b.position));
    final cover = sortedMedia.isNotEmpty ? sortedMedia.first : null;
    final coverIsVideo = cover?.isVideo ?? false;

    return GestureDetector(
      onTap: onTap,
      child: Container(
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(16),
          color: const Color(0xFF0F172A),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.08),
              blurRadius: 12,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        clipBehavior: Clip.antiAlias,
        child: Stack(
          fit: StackFit.expand,
          children: [
            // Cover — custom thumbnail (videos) > image > video first frame > placeholder
            if (coverIsVideo && cover != null && cover.hasCustomThumbnail)
              CachedNetworkImage(
                imageUrl: cover.thumbnailUrl,
                fit: BoxFit.cover,
                placeholder: (_, __) => Container(
                  color: Theme.of(context).colorScheme.surfaceContainerHighest,
                ),
                // If the custom thumbnail fails to decode (e.g. truncated upload),
                // fall back to extracting the video's first frame.
                errorWidget: (_, __, ___) =>
                    PortfolioVideoThumbnail(videoUrl: cover.mediaUrl),
              )
            else if (coverIsVideo && cover != null)
              PortfolioVideoThumbnail(videoUrl: cover.mediaUrl)
            else if (cover != null && cover.mediaUrl.isNotEmpty)
              CachedNetworkImage(
                imageUrl: cover.mediaUrl,
                fit: BoxFit.cover,
                placeholder: (_, __) => Container(
                  color: Theme.of(context).colorScheme.surfaceContainerHighest,
                ),
                errorWidget: (_, __, ___) => _placeholderCover(context),
              )
            else
              _placeholderCover(context),

            // Play icon for videos
            if (coverIsVideo)
              Center(
                child: Container(
                  width: 52,
                  height: 52,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: Colors.black.withValues(alpha: 0.5),
                  ),
                  child: const Icon(
                    Icons.play_arrow,
                    color: Colors.white,
                    size: 28,
                  ),
                ),
              ),

            // Media count badge
            if (item.media.length > 1)
              Positioned(
                top: 10,
                left: 10,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.black.withValues(alpha: 0.6),
                    borderRadius: BorderRadius.circular(99),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      if (item.imageCount > 0) ...[
                        const Icon(
                          Icons.image,
                          color: Colors.white,
                          size: 12,
                        ),
                        const SizedBox(width: 3),
                        Text(
                          '${item.imageCount}',
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                      if (item.imageCount > 0 && item.videoCount > 0)
                        const SizedBox(width: 6),
                      if (item.videoCount > 0) ...[
                        const Icon(
                          Icons.movie,
                          color: Colors.white,
                          size: 12,
                        ),
                        const SizedBox(width: 3),
                        Text(
                          '${item.videoCount}',
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 11,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ),

            // Edit/Delete buttons (top right) — only in edit mode
            if (!readOnly)
              Positioned(
                top: 8,
                right: 8,
                child: Row(
                  children: [
                    _CardActionButton(
                      icon: Icons.edit,
                      onTap: onEdit,
                    ),
                    const SizedBox(width: 6),
                    _CardActionButton(
                      icon: Icons.delete_outline,
                      onTap: onDelete,
                      destructive: true,
                    ),
                  ],
                ),
              ),

            // Bottom gradient + title
            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              child: Container(
                padding: const EdgeInsets.fromLTRB(12, 32, 12, 12),
                decoration: const BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      Colors.transparent,
                      Color(0xCC000000),
                      Color(0xF2000000),
                    ],
                  ),
                ),
                child: Text(
                  item.title,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  softWrap: true,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _placeholderCover(BuildContext context) {
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [Color(0xFFE2E8F0), Color(0xFFCBD5E1)],
        ),
      ),
      child: const Center(
        child: Icon(
          Icons.image_outlined,
          size: 36,
          color: Color(0xFF94A3B8),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Card action button (edit / delete)
// ---------------------------------------------------------------------------

class _CardActionButton extends StatelessWidget {
  final IconData icon;
  final VoidCallback onTap;
  final bool destructive;

  const _CardActionButton({
    required this.icon,
    required this.onTap,
    this.destructive = false,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: 30,
        height: 30,
        decoration: BoxDecoration(
          color: Colors.white.withValues(alpha: 0.95),
          shape: BoxShape.circle,
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.2),
              blurRadius: 6,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Icon(
          icon,
          size: 16,
          color: destructive ? Colors.red : const Color(0xFF334155),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Skeleton
// ---------------------------------------------------------------------------

class _PortfolioSkeleton extends StatelessWidget {
  const _PortfolioSkeleton();

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: GridView.builder(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 2,
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          childAspectRatio: 0.78,
        ),
        itemCount: 4,
        itemBuilder: (context, _) => Container(
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(16),
            color: Theme.of(context).colorScheme.surfaceContainerHighest,
          ),
        ),
      ),
    );
  }
}
