/// expertise_section.dart — checkbox list of expertise domains.
/// Mirrors the web `ExpertiseSection`. Uses the canonical list
/// from [kMobileExpertiseDomains].
library;

import 'package:flutter/material.dart';

import '../../../../../shared/search/search_filters.dart';
import 'filter_primitives.dart';

class ExpertiseSection extends StatelessWidget {
  const ExpertiseSection({
    super.key,
    required this.selected,
    required this.onChanged,
    required this.title,
  });

  final Set<String> selected;
  final ValueChanged<Set<String>> onChanged;
  final String title;

  @override
  Widget build(BuildContext context) {
    return FilterSectionShell(
      title: title,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: kMobileExpertiseDomains
            .map(
              (key) => FilterCheckboxRow(
                key: ValueKey('expertise-$key'),
                label: _humanize(key),
                checked: selected.contains(key),
                onChanged: (now) => _toggle(key, now),
              ),
            )
            .toList(growable: false),
      ),
    );
  }

  void _toggle(String key, bool present) {
    final next = Set<String>.from(selected);
    if (present) {
      next.add(key);
    } else {
      next.remove(key);
    }
    onChanged(next);
  }

  /// _humanize converts a snake_case expertise key into a human
  /// label. Temporary: when mobile l10n ships expertise labels,
  /// swap to `AppLocalizations.of(context).expertiseDomain(key)`.
  static String _humanize(String key) {
    return key
        .split('_')
        .map((p) => p.isEmpty ? p : '${p[0].toUpperCase()}${p.substring(1)}')
        .join(' ');
  }
}
