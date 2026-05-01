import 'package:flutter/material.dart';

import '../../../../../l10n/app_localizations.dart';

/// Horizontal filter row at the top of the conversations list — All
/// / Agency / Freelancer / Enterprise pills.
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
    final l10n = AppLocalizations.of(context)!;
    final filters = [
      ('all', l10n.messagingAllRoles, null),
      ('agency', l10n.messagingAgency, const Color(0xFF2563EB)),
      (
        'provider_personal',
        l10n.messagingFreelancer,
        const Color(0xFFF43F5E),
      ),
      ('enterprise', l10n.messagingEnterprise, const Color(0xFF8B5CF6)),
    ];

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: filters.map((filter) {
          final (key, label, color) = filter;
          final isSelected = selected == key;

          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: FilterChip(
              label: Text(
                label,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: isSelected
                      ? Colors.white
                      : (color ?? Theme.of(context).colorScheme.onSurface),
                ),
              ),
              selected: isSelected,
              onSelected: (_) => onChanged(key),
              backgroundColor: color?.withValues(alpha: 0.08) ??
                  Theme.of(context)
                      .colorScheme
                      .onSurface
                      .withValues(alpha: 0.06),
              selectedColor:
                  color ?? Theme.of(context).colorScheme.onSurface,
              side: BorderSide.none,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(20),
              ),
              showCheckmark: false,
              padding:
                  const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            ),
          );
        }).toList(),
      ),
    );
  }
}
