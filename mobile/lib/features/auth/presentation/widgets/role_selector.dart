import 'package:flutter/material.dart';

class RoleSelector extends StatelessWidget {
  final String selectedRole;
  final ValueChanged<String> onRoleChanged;

  const RoleSelector({super.key, required this.selectedRole, required this.onRoleChanged});

  static const _roles = [
    ('agency', 'Agence', Icons.business, 'Gérez votre agence et vos prestataires'),
    ('enterprise', 'Entreprise', Icons.corporate_fare, 'Publiez des projets et recrutez'),
    ('provider', 'Freelance', Icons.person, 'Proposez vos services et compétences'),
  ];

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Row(
      children: _roles.map((role) {
        final (value, label, icon, description) = role;
        final isSelected = selectedRole == value;

        return Expanded(
          child: Padding(
            padding: EdgeInsets.only(right: value == 'provider' ? 0 : 8),
            child: GestureDetector(
              onTap: () => onRoleChanged(value),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(
                    color: isSelected ? theme.colorScheme.primary : theme.colorScheme.outline,
                    width: isSelected ? 2 : 1,
                  ),
                  color: isSelected ? theme.colorScheme.primaryContainer.withOpacity(0.3) : null,
                ),
                child: Column(
                  children: [
                    Icon(icon, size: 28, color: isSelected ? theme.colorScheme.primary : theme.colorScheme.onSurfaceVariant),
                    const SizedBox(height: 6),
                    Text(label, style: theme.textTheme.labelLarge?.copyWith(
                      fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
                      color: isSelected ? theme.colorScheme.primary : null,
                    )),
                    const SizedBox(height: 4),
                    Text(description, style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.onSurfaceVariant),
                      textAlign: TextAlign.center, maxLines: 2, overflow: TextOverflow.ellipsis),
                  ],
                ),
              ),
            ),
          ),
        );
      }).toList(),
    );
  }
}
