import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/catalog_entry.dart';
import '../providers/skill_catalog_provider.dart';
import '../utils/skill_labels.dart';
import 'skill_chip_widget.dart';

/// Collapsible panel that renders all curated skills for one
/// expertise domain as tappable chips.
///
/// The panel is a thin wrapper around [ExpansionTile] plus a
/// [skillCatalogProvider(key)] watch. Loading / error / empty states
/// are handled inline — we never surface a blocking spinner.
///
/// Already-selected skills are filtered out of the chip list so
/// the editor never shows a duplicate. Tapping a chip calls
/// [onPick] with the catalog entry.
class ExpertiseSkillsPanel extends ConsumerWidget {
  const ExpertiseSkillsPanel({
    super.key,
    required this.expertiseKey,
    required this.selectedKeys,
    required this.onPick,
    this.initiallyExpanded = false,
  });

  final String expertiseKey;
  final Set<String> selectedKeys;
  final ValueChanged<CatalogEntry> onPick;
  final bool initiallyExpanded;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final asyncPage = ref.watch(skillCatalogProvider(expertiseKey));
    // Force the title color to the standard surface foreground so the
    // panel headers stay readable in light mode regardless of any
    // ambient ExpansionTileTheme that might leak a white text colour
    // from elsewhere in the app theme.
    final titleColor = theme.colorScheme.onSurface;

    return Card(
      margin: EdgeInsets.zero,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.dividerColor),
      ),
      child: ExpansionTile(
        initiallyExpanded: initiallyExpanded,
        tilePadding: const EdgeInsets.symmetric(horizontal: 16),
        childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
        shape: const Border(),
        collapsedShape: const Border(),
        textColor: titleColor,
        collapsedTextColor: titleColor,
        iconColor: titleColor,
        collapsedIconColor: titleColor,
        title: Text(
          localizedDomainLabel(context, expertiseKey),
          style: theme.textTheme.titleSmall?.copyWith(
            color: titleColor,
            fontWeight: FontWeight.w600,
          ),
        ),
        children: [
          asyncPage.when(
            loading: () => const _PanelLoading(),
            error: (_, __) => const _PanelError(),
            data: (page) => _PanelChips(
              entries: page.skills,
              selectedKeys: selectedKeys,
              onPick: onPick,
            ),
          ),
        ],
      ),
    );
  }
}

class _PanelLoading extends StatelessWidget {
  const _PanelLoading();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 12),
      child: Center(child: CircularProgressIndicator.adaptive()),
    );
  }
}

class _PanelError extends StatelessWidget {
  const _PanelError();

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Text(
        l10n.skillsErrorGeneric,
        style:
            theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.error),
      ),
    );
  }
}

class _PanelChips extends StatelessWidget {
  const _PanelChips({
    required this.entries,
    required this.selectedKeys,
    required this.onPick,
  });

  final List<CatalogEntry> entries;
  final Set<String> selectedKeys;
  final ValueChanged<CatalogEntry> onPick;

  @override
  Widget build(BuildContext context) {
    final filtered = entries
        .where((e) => !selectedKeys.contains(e.skillText.toLowerCase()))
        .toList(growable: false);

    if (filtered.isEmpty) {
      final l10n = AppLocalizations.of(context)!;
      final theme = Theme.of(context);
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Text(
          l10n.skillsEmpty,
          style: theme.textTheme.bodySmall?.copyWith(color: theme.hintColor),
        ),
      );
    }

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (final entry in filtered)
          SelectableSkillChip(
            label: entry.displayText,
            onTap: () => onPick(entry),
          ),
      ],
    );
  }
}
