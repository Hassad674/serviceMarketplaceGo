import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/client_profile.dart';
import '../providers/client_profile_provider.dart';
import '../widgets/client_profile_description_widget.dart';
import '../widgets/client_profile_header.dart';
import '../widgets/client_project_history_widget.dart';

/// Public (read-only) client-profile screen.
///
/// Route: `/clients/:orgId`.
///
/// The screen renders the exact same widgets as the private screen
/// but WITHOUT any editing affordance and WITHOUT a "Send message"
/// button (intentional — Contra/Upwork pattern: providers discover
/// clients via projects, not cold outreach).
class PublicClientProfileScreen extends ConsumerWidget {
  const PublicClientProfileScreen({
    super.key,
    required this.organizationId,
  });

  final String organizationId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final async = ref.watch(publicClientProfileProvider(organizationId));

    return Scaffold(
      appBar: AppBar(title: Text(l10n.clientProfilePublicTitle)),
      body: async.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => _ErrorBody(
          isNotFound: _isNotFound(error),
          onRetry: () =>
              ref.invalidate(publicClientProfileProvider(organizationId)),
        ),
        data: (profile) => _Content(profile: profile),
      ),
    );
  }

  bool _isNotFound(Object error) {
    if (error is DioException) {
      return error.response?.statusCode == 404;
    }
    return false;
  }
}

class _Content extends StatelessWidget {
  const _Content({required this.profile});

  final ClientProfile profile;

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          ClientProfileHeader(
            companyName: profile.companyName,
            avatarUrl: profile.avatarUrl,
            orgType: profile.type.isNotEmpty ? profile.type : null,
            totalSpentCents: profile.totalSpent,
            reviewCount: profile.reviewCount,
            averageRating: profile.averageRating,
            projectsCompleted: profile.projectsCompletedAsClient,
          ),
          const SizedBox(height: 16),
          ClientProfileDescriptionWidget(
            description: profile.clientDescription,
          ),
          const SizedBox(height: 16),
          ClientProjectHistoryWidget(projects: profile.projectHistory),
          const SizedBox(height: 24),
        ],
      ),
    );
  }
}

class _ErrorBody extends StatelessWidget {
  const _ErrorBody({required this.isNotFound, required this.onRetry});

  final bool isNotFound;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    if (isNotFound) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.search_off,
                size: 48,
                color: appColors?.mutedForeground,
              ),
              const SizedBox(height: 16),
              Text(
                l10n.clientProfileNotFound,
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyLarge,
              ),
            ],
          ),
        ),
      );
    }

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: theme.colorScheme.error,
            ),
            const SizedBox(height: 16),
            Text(
              l10n.couldNotLoadProfile,
              style: theme.textTheme.titleMedium,
            ),
            const SizedBox(height: 16),
            ElevatedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh, size: 18),
              label: Text(l10n.retry),
            ),
          ],
        ),
      ),
    );
  }
}
