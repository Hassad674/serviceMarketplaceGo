import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/theme/theme_provider.dart';
import '../../../../l10n/app_localizations.dart';

/// Dark / light mode toggle row reused on the freelance and referrer
/// profile screens.
class FreelanceDarkModeToggle extends ConsumerWidget {
  const FreelanceDarkModeToggle({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final themeMode = ref.watch(themeModeProvider);
    final isDark = themeMode == ThemeMode.dark;
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: ListTile(
        leading: Icon(
          isDark ? Icons.dark_mode : Icons.light_mode,
          color: theme.colorScheme.primary,
        ),
        title: Text(l10n.darkMode),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        ),
        trailing: Switch(
          value: isDark,
          activeTrackColor: theme.colorScheme.primary,
          onChanged: (_) => ref.read(themeModeProvider.notifier).toggle(),
        ),
      ),
    );
  }
}
