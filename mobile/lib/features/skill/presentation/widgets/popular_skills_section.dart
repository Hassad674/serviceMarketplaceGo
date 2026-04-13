import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/catalog_entry.dart';
import '../../domain/repositories/skill_repository.dart';
import '../providers/skill_catalog_provider.dart';
import 'skill_chip_widget.dart';

/// "Popular in your domains" row shown at the top of the editor.
///
/// Merges the top-N skills from every expertise domain the operator
/// has declared, sorts by [CatalogEntry.usageCount] descending, and
/// hides any skill already in the draft selection.
///
/// Uses [ref.watch] on `skillCatalogProvider(key)` for each key —
/// Riverpod caches per-family entries so switching between editor
/// sessions is cheap.
class PopularSkillsSection extends ConsumerWidget {
  const PopularSkillsSection({
    super.key,
    required this.expertiseKeys,
    required this.selectedKeys,
    required this.onPick,
    this.maxDisplayed = 8,
  });

  final List<String> expertiseKeys;
  final Set<String> selectedKeys;
  final ValueChanged<CatalogEntry> onPick;
  final int maxDisplayed;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (expertiseKeys.isEmpty) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    // Collect the AsyncValues for each expertise key.
    final pages = <AsyncValue<SkillCatalogPage>>[
      for (final key in expertiseKeys) ref.watch(skillCatalogProvider(key)),
    ];

    final isLoading = pages.any((p) => p.isLoading);
    final merged = <String, CatalogEntry>{};
    for (final async in pages) {
      final list = async.maybeWhen(
        data: (page) => page.skills,
        orElse: () => const <CatalogEntry>[],
      );
      for (final entry in list) {
        if (selectedKeys.contains(entry.skillText.toLowerCase())) continue;
        // Dedupe by canonical skill text across domains.
        merged.putIfAbsent(entry.skillText, () => entry);
      }
    }

    final sorted = merged.values.toList(growable: false)
      ..sort((a, b) => b.usageCount.compareTo(a.usageCount));
    final top = sorted.take(maxDisplayed).toList(growable: false);

    if (top.isEmpty && !isLoading) return const SizedBox.shrink();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 4),
          child: Text(
            l10n.skillsPopularHeading,
            style: theme.textTheme.titleSmall,
          ),
        ),
        const SizedBox(height: 10),
        if (isLoading && top.isEmpty)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 8),
            child: Center(child: CircularProgressIndicator.adaptive()),
          )
        else
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final entry in top)
                SelectableSkillChip(
                  label: entry.displayText,
                  onTap: () => onPick(entry),
                ),
            ],
          ),
      ],
    );
  }
}
