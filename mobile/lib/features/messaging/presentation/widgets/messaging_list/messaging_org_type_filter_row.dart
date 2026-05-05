import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';

// M-18 filter row — Soleil v2 pills (corail accents).
class MessagingOrgTypeFilterRow extends StatelessWidget {
  const MessagingOrgTypeFilterRow({
    super.key,
    required this.selected,
    required this.onChanged,
  });

  final String selected;
  final void Function(String) onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    final filters = <(String, String)>[
      ('all', l10n.messagingAllRoles),
      ('agency', l10n.messagingAgency),
      ('provider_personal', l10n.messagingFreelancer),
      ('enterprise', l10n.messagingEnterprise),
    ];

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: filters.map((filter) {
          final (key, label) = filter;
          final isSelected = selected == key;

          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: GestureDetector(
              onTap: () => onChanged(key),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                padding: const EdgeInsets.symmetric(
                  horizontal: 14,
                  vertical: 8,
                ),
                decoration: BoxDecoration(
                  color: isSelected
                      ? theme.colorScheme.onSurface
                      : (appColors?.muted ?? theme.colorScheme.surface),
                  borderRadius: BorderRadius.circular(999),
                  border: Border.all(
                    color: isSelected
                        ? theme.colorScheme.onSurface
                        : (appColors?.border ?? theme.dividerColor),
                  ),
                ),
                child: Text(
                  label,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: isSelected
                        ? theme.colorScheme.surface
                        : appColors?.mutedForeground,
                  ),
                ),
              ),
            ),
          );
        }).toList(),
      ),
    );
  }
}
