import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';

/// Centered spinner shown while the freelance profile is loading.
class FreelanceLoadingState extends StatelessWidget {
  const FreelanceLoadingState({super.key});

  @override
  Widget build(BuildContext context) {
    return const Center(child: CircularProgressIndicator());
  }
}

/// Error placeholder rendered when the freelance profile fails to
/// load. The retry callback re-invalidates the provider.
class FreelanceErrorState extends StatelessWidget {
  const FreelanceErrorState({super.key, required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.error_outline,
            size: 48,
            color: theme.colorScheme.error,
          ),
          const SizedBox(height: 12),
          Text(l10n.couldNotLoadProfile),
          const SizedBox(height: 12),
          ElevatedButton(onPressed: onRetry, child: Text(l10n.retry)),
        ],
      ),
    );
  }
}
