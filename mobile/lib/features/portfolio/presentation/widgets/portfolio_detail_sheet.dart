import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../shared/widgets/video_player_widget.dart';
import '../../domain/entities/portfolio_item.dart';

/// Bottom sheet showing full portfolio item details with media gallery.
class PortfolioDetailSheet extends StatefulWidget {
  final PortfolioItem item;

  const PortfolioDetailSheet({super.key, required this.item});

  @override
  State<PortfolioDetailSheet> createState() => _PortfolioDetailSheetState();
}

class _PortfolioDetailSheetState extends State<PortfolioDetailSheet> {
  late final PageController _pageController;
  int _currentPage = 0;

  List<PortfolioMedia> get _sortedMedia =>
      [...widget.item.media]..sort((a, b) => a.position.compareTo(b.position));

  @override
  void initState() {
    super.initState();
    _pageController = PageController();
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final media = _sortedMedia;

    return DraggableScrollableSheet(
      initialChildSize: 0.85,
      maxChildSize: 0.95,
      minChildSize: 0.5,
      builder: (context, scrollController) {
        return Container(
          decoration: BoxDecoration(
            color: Theme.of(context).cardColor,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(20)),
          ),
          child: Column(
            children: [
              // Handle
              Center(
                child: Container(
                  margin: const EdgeInsets.only(top: 12, bottom: 8),
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Theme.of(context).dividerColor,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              // Content
              Expanded(
                child: ListView(
                  controller: scrollController,
                  padding: EdgeInsets.zero,
                  children: [
                    // Media gallery
                    if (media.isNotEmpty) ...[
                      AspectRatio(
                        aspectRatio: 16 / 10,
                        child: Stack(
                          children: [
                            PageView.builder(
                              controller: _pageController,
                              itemCount: media.length,
                              onPageChanged: (i) =>
                                  setState(() => _currentPage = i),
                              itemBuilder: (context, index) {
                                final m = media[index];
                                if (m.isVideo) {
                                  return Container(
                                    color: Colors.black,
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 8,
                                      vertical: 8,
                                    ),
                                    child: Center(
                                      child: VideoPlayerWidget(
                                        videoUrl: m.mediaUrl,
                                        autoPlay: false,
                                      ),
                                    ),
                                  );
                                }
                                // Detail sheet shows the original at
                                // full screen — cap raster width at
                                // 1080 px (3x DPR × 360 lp portrait
                                // phone). RepaintBoundary keeps the
                                // PageView swipe smooth: only the
                                // active page repaints during a
                                // gesture (PERF-M-08).
                                return RepaintBoundary(
                                  child: CachedNetworkImage(
                                    imageUrl: m.mediaUrl,
                                    fit: BoxFit.contain,
                                    memCacheWidth: 1080,
                                    maxWidthDiskCache: 1080,
                                    placeholder: (_, __) => Container(
                                      color: Colors.black12,
                                    ),
                                    errorWidget: (_, __, ___) => const Center(
                                      child: Icon(
                                        Icons.broken_image,
                                        color: Colors.white54,
                                        size: 48,
                                      ),
                                    ),
                                  ),
                                );
                              },
                            ),
                            // Dots indicator
                            if (media.length > 1)
                              Positioned(
                                bottom: 12,
                                left: 0,
                                right: 0,
                                child: Row(
                                  mainAxisAlignment: MainAxisAlignment.center,
                                  children: List.generate(
                                    media.length,
                                    (i) => Container(
                                      margin: const EdgeInsets.symmetric(
                                        horizontal: 3,
                                      ),
                                      width: 8,
                                      height: 8,
                                      decoration: BoxDecoration(
                                        shape: BoxShape.circle,
                                        color: i == _currentPage
                                            ? Colors.white
                                            : Colors.white38,
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                          ],
                        ),
                      ),
                    ],

                    // Title
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
                      child: Text(
                        widget.item.title,
                        softWrap: true,
                        style:
                            Theme.of(context).textTheme.titleLarge?.copyWith(
                                  fontWeight: FontWeight.w600,
                                ),
                      ),
                    ),

                    // Description
                    if (widget.item.description.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                        child: Text(
                          widget.item.description,
                          softWrap: true,
                          style: Theme.of(context)
                              .textTheme
                              .bodyMedium
                              ?.copyWith(
                                color: Theme.of(context)
                                    .colorScheme
                                    .onSurfaceVariant,
                              ),
                        ),
                      ),

                    // Link
                    if (widget.item.linkUrl.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                        child: InkWell(
                          onTap: () => _launchUrl(widget.item.linkUrl),
                          borderRadius: BorderRadius.circular(8),
                          child: Padding(
                            padding: const EdgeInsets.symmetric(vertical: 4),
                            child: Row(
                              children: [
                                Icon(
                                  Icons.open_in_new,
                                  size: 18,
                                  color:
                                      Theme.of(context).colorScheme.primary,
                                ),
                                const SizedBox(width: 8),
                                Text(
                                  'View project',
                                  style: TextStyle(
                                    color:
                                        Theme.of(context).colorScheme.primary,
                                    fontWeight: FontWeight.w500,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),

                    const SizedBox(height: 24),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  Future<void> _launchUrl(String url) async {
    final uri = Uri.tryParse(url);
    if (uri != null && await canLaunchUrl(uri)) {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    }
  }
}
