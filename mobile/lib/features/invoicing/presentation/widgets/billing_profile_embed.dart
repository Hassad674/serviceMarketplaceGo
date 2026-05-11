import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../providers/invoicing_providers.dart';
import 'billing_profile_form.dart';
import 'billing_profile_summary.dart';

/// Inline embed for the billing-profile, mirroring the web
/// `BillingProfileEmbed`. Two modes:
///
///   - [BillingEmbedMode.summary]  — compact read-only summary card,
///     used when the profile is already complete.
///   - [BillingEmbedMode.form]     — the full [BillingProfileForm], used
///     when the user opts to edit OR when the profile is incomplete.
///
/// The parent owns the mode state so the payment screen can gate the
/// confirm button on `mode === summary && profile.is_complete`.
///
/// Reads the snapshot via `billingProfileProvider`. Loading / error
/// states are rendered as compact placeholders so the page layout
/// stays stable.
enum BillingEmbedMode { summary, form }

class BillingProfileEmbed extends ConsumerWidget {
  const BillingProfileEmbed({
    super.key,
    required this.mode,
    required this.onEdit,
    required this.onSaved,
  });

  /// Controlled rendering mode.
  final BillingEmbedMode mode;

  /// Fired when the user clicks "Modifier" inside the summary view.
  /// The parent must flip its mode state to [BillingEmbedMode.form].
  final VoidCallback onEdit;

  /// Fired after a successful save where the resulting profile passes
  /// server-side completeness. The parent must flip its mode state
  /// back to [BillingEmbedMode.summary].
  final VoidCallback onSaved;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final snapshotAsync = ref.watch(billingProfileProvider);
    final l10n = AppLocalizations.of(context)!;
    return snapshotAsync.when(
      loading: () => const _LoadingPlaceholder(),
      error: (_, __) => _ErrorPlaceholder(
        message: l10n.unexpectedError,
        onRetry: () => ref.invalidate(billingProfileProvider),
      ),
      data: (snapshot) {
        if (mode == BillingEmbedMode.summary) {
          return BillingProfileSummary(
            profile: snapshot.profile,
            onEdit: onEdit,
          );
        }
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (!snapshot.isComplete)
              _CompletePromptBanner(l10n: l10n),
            if (!snapshot.isComplete) const SizedBox(height: 12),
            BillingProfileForm(onSaved: onSaved),
          ],
        );
      },
    );
  }
}

class _LoadingPlaceholder extends StatelessWidget {
  const _LoadingPlaceholder();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Center(
        child: SizedBox(
          width: 18,
          height: 18,
          child: CircularProgressIndicator(
            strokeWidth: 2,
            color: theme.colorScheme.primary,
          ),
        ),
      ),
    );
  }
}

class _ErrorPlaceholder extends StatelessWidget {
  const _ErrorPlaceholder({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Row(
        children: [
          Icon(
            Icons.error_outline_rounded,
            size: 20,
            color: theme.colorScheme.error,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: SoleilTextStyles.body.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                fontSize: 13,
              ),
            ),
          ),
          TextButton(
            onPressed: onRetry,
            child: Text(AppLocalizations.of(context)!.retry),
          ),
        ],
      ),
    );
  }
}

class _CompletePromptBanner extends StatelessWidget {
  const _CompletePromptBanner({required this.l10n});

  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final warningSoft =
        appColors?.subtleForeground ?? theme.colorScheme.onSurfaceVariant;
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: theme.colorScheme.errorContainer.withValues(alpha: 0.25),
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: theme.colorScheme.error.withValues(alpha: 0.35),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.billingEmbedCompletePromptTitle,
            style: SoleilTextStyles.bodyEmphasis.copyWith(
              color: theme.colorScheme.onSurface,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            l10n.billingEmbedCompletePromptBody,
            style: SoleilTextStyles.body.copyWith(
              color: warningSoft,
              fontSize: 13,
            ),
          ),
        ],
      ),
    );
  }
}
