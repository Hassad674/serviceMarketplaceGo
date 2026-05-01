import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../../core/theme/app_theme.dart';
import '../../../../../l10n/app_localizations.dart';
import '../../../../review/presentation/providers/review_provider.dart';

/// Vertical 16dp gap that hides itself when the previous block is
/// not visible. Keeps the column tight when the profile collapses.
class PublicProfileSpacerIfVisible extends StatelessWidget {
  const PublicProfileSpacerIfVisible({super.key, required this.visible});

  final bool visible;

  @override
  Widget build(BuildContext context) {
    if (!visible) return const SizedBox.shrink();
    return const SizedBox(height: 16);
  }
}

/// "Send Message" button shown above the public profile body when
/// the viewer is authenticated and not on their own org page.
class PublicProfileSendMessageButton extends StatelessWidget {
  const PublicProfileSendMessageButton({
    super.key,
    required this.sending,
    required this.onPressed,
  });

  final bool sending;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return SizedBox(
      width: double.infinity,
      child: ElevatedButton.icon(
        onPressed: sending ? null : onPressed,
        icon: sending
            ? const SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  color: Colors.white,
                ),
              )
            : const Icon(Icons.chat_outlined, size: 20),
        label: Text(l10n.messagingSendMessage),
        style: ElevatedButton.styleFrom(
          backgroundColor: const Color(0xFFF43F5E),
          foregroundColor: Colors.white,
          minimumSize: const Size(double.infinity, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
        ),
      ),
    );
  }
}

/// Pill rendered next to the public profile header showing the
/// org type (agency / freelance / enterprise).
class PublicProfileOrgTypeBadge extends StatelessWidget {
  const PublicProfileOrgTypeBadge({super.key, required this.orgType});

  final String orgType;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
      decoration: BoxDecoration(
        color: _color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Text(
        _label,
        style: TextStyle(
          color: _color,
          fontWeight: FontWeight.w600,
          fontSize: 13,
        ),
      ),
    );
  }

  String get _label {
    switch (orgType) {
      case 'agency':
        return 'Agency';
      case 'enterprise':
        return 'Enterprise';
      case 'provider_personal':
        return 'Freelance';
      default:
        return orgType;
    }
  }

  Color get _color {
    switch (orgType) {
      case 'agency':
        return const Color(0xFF2563EB);
      case 'enterprise':
        return const Color(0xFF8B5CF6);
      case 'provider_personal':
        return const Color(0xFFF43F5E);
      default:
        return const Color(0xFF64748B);
    }
  }
}

/// Star rating pill shown under the public profile header.
class PublicProfileAverageRating extends ConsumerWidget {
  const PublicProfileAverageRating({super.key, required this.orgId});

  final String orgId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncAvg = ref.watch(averageRatingProvider(orgId));
    return asyncAvg.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (avg) {
        if (avg.count == 0) return const SizedBox.shrink();
        return Row(
          mainAxisAlignment: MainAxisAlignment.center,
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.star, color: Color(0xFFFBBF24), size: 16),
            const SizedBox(width: 4),
            Text(
              avg.average.toStringAsFixed(1),
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(width: 4),
            Text(
              '(${avg.count})',
              style: TextStyle(
                fontSize: 12,
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        );
      },
    );
  }
}
