import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/router/app_router.dart';
import '../../features/auth/presentation/providers/auth_provider.dart';
import '../../l10n/app_localizations.dart';

/// Persistent KYC warning banner for providers/agencies with pending funds.
/// Shows a warning (amber) when KYC deadline is running, and a critical
/// alert (red) when the account is restricted after 14 days.
class KYCBanner extends ConsumerWidget {
  const KYCBanner({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final user = authState.user;
    if (user == null) return const SizedBox.shrink();

    final role = user['role'] as String? ?? '';
    if (role == 'enterprise') return const SizedBox.shrink();

    final kycStatus = user['kyc_status'] as String? ?? 'none';
    if (kycStatus == 'none' || kycStatus == 'completed') {
      return const SizedBox.shrink();
    }

    final isRestricted = kycStatus == 'restricted';
    final deadline = user['kyc_deadline'] as String?;
    final daysLeft = deadline != null ? _daysUntil(deadline) : null;

    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
      child: Material(
        borderRadius: BorderRadius.circular(12),
        color: isRestricted
            ? Colors.red.shade50
            : Colors.amber.shade50,
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () => context.go(RoutePaths.paymentInfo),
          child: Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: isRestricted
                    ? Colors.red.shade200
                    : Colors.amber.shade200,
              ),
            ),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(
                  isRestricted ? Icons.shield_outlined : Icons.warning_amber,
                  color: isRestricted ? Colors.red.shade600 : Colors.amber.shade700,
                  size: 22,
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        isRestricted
                            ? l10n.kycBannerRestrictedTitle
                            : l10n.kycBannerPendingTitle,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                          color: isRestricted
                              ? Colors.red.shade900
                              : Colors.amber.shade900,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        isRestricted
                            ? l10n.kycBannerRestrictedBody
                            : l10n.kycBannerPendingBody(daysLeft ?? 0),
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: isRestricted
                              ? Colors.red.shade700
                              : Colors.amber.shade700,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                Icon(
                  Icons.arrow_forward_ios,
                  size: 14,
                  color: isRestricted
                      ? Colors.red.shade400
                      : Colors.amber.shade600,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  int _daysUntil(String deadline) {
    final dt = DateTime.tryParse(deadline);
    if (dt == null) return 0;
    final diff = dt.difference(DateTime.now()).inDays;
    return diff < 0 ? 0 : diff;
  }
}
