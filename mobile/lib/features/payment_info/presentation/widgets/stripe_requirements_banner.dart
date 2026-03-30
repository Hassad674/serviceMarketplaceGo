import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

// ---------------------------------------------------------------------------
// Stripe requirements data
// ---------------------------------------------------------------------------

class _StripeRequirements {
  final bool hasRequirements;
  final List<String> requirements;

  const _StripeRequirements({
    required this.hasRequirements,
    required this.requirements,
  });
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

final _stripeRequirementsProvider =
    FutureProvider.family<_StripeRequirements, String>((ref, lang) async {
  final api = ref.watch(apiClientProvider);
  try {
    final response = await api.get(
      '/api/v1/payment-info/requirements',
      queryParameters: {'lang': lang},
    );
    final data = response.data as Map<String, dynamic>?;
    if (data == null) {
      return const _StripeRequirements(
        hasRequirements: false,
        requirements: [],
      );
    }
    final hasReq = data['has_requirements'] as bool? ?? false;
    final reqList = (data['requirements'] as List<dynamic>?)
            ?.map((e) => e.toString())
            .toList() ??
        [];
    return _StripeRequirements(
      hasRequirements: hasReq,
      requirements: reqList,
    );
  } catch (_) {
    return const _StripeRequirements(
      hasRequirements: false,
      requirements: [],
    );
  }
});

// ---------------------------------------------------------------------------
// Widget
// ---------------------------------------------------------------------------

/// Banner that shows pending Stripe requirements with an Account Link button.
///
/// Calls GET /api/v1/payment-info/requirements to check for pending items.
/// When requirements exist, shows an amber banner with a button to open
/// Stripe's Account Link in an external browser.
class StripeRequirementsBanner extends ConsumerStatefulWidget {
  const StripeRequirementsBanner({super.key});

  @override
  ConsumerState<StripeRequirementsBanner> createState() =>
      _StripeRequirementsBannerState();
}

class _StripeRequirementsBannerState
    extends ConsumerState<StripeRequirementsBanner> {
  bool _opening = false;

  Future<void> _openAccountLink() async {
    setState(() => _opening = true);
    try {
      final api = ref.read(apiClientProvider);
      final response =
          await api.post('/api/v1/payment-info/account-link');
      final data = response.data as Map<String, dynamic>?;
      final url = data?['url'] as String?;
      if (url != null && url.isNotEmpty) {
        await launchUrl(
          Uri.parse(url),
          mode: LaunchMode.externalApplication,
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to open Stripe: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _opening = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final lang = Localizations.localeOf(context).languageCode;
    final asyncReqs = ref.watch(_stripeRequirementsProvider(lang));

    return asyncReqs.when(
      loading: () => const SizedBox.shrink(),
      error: (_, __) => const SizedBox.shrink(),
      data: (reqs) {
        if (!reqs.hasRequirements) return const SizedBox.shrink();
        return _buildBanner(context, l10n, reqs);
      },
    );
  }

  Widget _buildBanner(
    BuildContext context,
    AppLocalizations l10n,
    _StripeRequirements reqs,
  ) {
    final isDark = Theme.of(context).brightness == Brightness.dark;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: isDark
            ? const Color(0xFFF59E0B).withValues(alpha: 0.1)
            : const Color(0xFFFFFBEB),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: isDark
              ? const Color(0xFFF59E0B).withValues(alpha: 0.3)
              : const Color(0xFFFDE68A),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.warning_amber_outlined,
                size: 20,
                color: isDark
                    ? const Color(0xFFFBBF24)
                    : const Color(0xFFD97706),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.stripeRequirementsTitle,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: isDark
                        ? const Color(0xFFFBBF24)
                        : const Color(0xFF92400E),
                  ),
                ),
              ),
            ],
          ),
          if (reqs.requirements.isNotEmpty) ...[
            const SizedBox(height: 8),
            ...reqs.requirements.map(
              (r) => Padding(
                padding: const EdgeInsets.only(left: 28, bottom: 2),
                child: Text(
                  '\u2022 $r',
                  style: TextStyle(
                    fontSize: 12,
                    color: isDark
                        ? const Color(0xFFFBBF24)
                        : const Color(0xFF92400E),
                  ),
                ),
              ),
            ),
          ],
          const SizedBox(height: 8),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton(
              onPressed: _opening ? null : _openAccountLink,
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFFF59E0B),
                foregroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                  borderRadius:
                      BorderRadius.circular(AppTheme.radiusMd),
                ),
                padding: const EdgeInsets.symmetric(vertical: 10),
              ),
              child: _opening
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor:
                            AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : Text(
                      l10n.stripeCompleteOnStripe,
                      style: const TextStyle(
                        fontWeight: FontWeight.w600,
                        fontSize: 13,
                      ),
                    ),
            ),
          ),
        ],
      ),
    );
  }
}
