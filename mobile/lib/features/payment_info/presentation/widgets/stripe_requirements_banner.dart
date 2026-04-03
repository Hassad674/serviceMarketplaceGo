import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

// ---------------------------------------------------------------------------
// Stripe requirements data (public for use by the form screen)
// ---------------------------------------------------------------------------

class RequirementField {
  final String key;
  final String labelKey;

  const RequirementField({required this.key, required this.labelKey});
}

class RequirementSection {
  final String id;
  final String titleKey;
  final List<RequirementField> fields;

  const RequirementSection({
    required this.id,
    required this.titleKey,
    required this.fields,
  });
}

class StripeRequirements {
  final bool hasRequirements;
  final List<RequirementSection> sections;

  const StripeRequirements({
    required this.hasRequirements,
    required this.sections,
  });
}

// ---------------------------------------------------------------------------
// Provider
// ---------------------------------------------------------------------------

final stripeRequirementsProvider =
    FutureProvider<StripeRequirements>((ref) async {
  final api = ref.watch(apiClientProvider);
  try {
    final response = await api.get('/api/v1/payment-info/requirements');
    final data = response.data as Map<String, dynamic>?;
    if (data == null) {
      return const StripeRequirements(
        hasRequirements: false,
        sections: [],
      );
    }
    final hasReq = data['has_requirements'] as bool? ?? false;
    final rawSections = data['sections'] as List<dynamic>? ?? [];
    final sections = rawSections.map((s) {
      final sMap = s as Map<String, dynamic>;
      final rawFields = sMap['fields'] as List<dynamic>? ?? [];
      final fields = rawFields.map((f) {
        final fMap = f as Map<String, dynamic>;
        return RequirementField(
          key: fMap['key'] as String? ?? '',
          labelKey: fMap['label_key'] as String? ?? '',
        );
      }).toList();
      return RequirementSection(
        id: sMap['id'] as String? ?? '',
        titleKey: sMap['title_key'] as String? ?? '',
        fields: fields,
      );
    }).toList();

    return StripeRequirements(
      hasRequirements: hasReq,
      sections: sections,
    );
  } catch (_) {
    return const StripeRequirements(
      hasRequirements: false,
      sections: [],
    );
  }
});

// ---------------------------------------------------------------------------
// Widget
// ---------------------------------------------------------------------------

/// Banner that shows pending Stripe requirements with a list of missing fields.
///
/// Calls GET /api/v1/payment-info/requirements to check for pending items.
/// When requirements exist, shows an amber banner listing the required fields.
class StripeRequirementsBanner extends ConsumerWidget {
  const StripeRequirementsBanner({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final asyncReqs = ref.watch(stripeRequirementsProvider);

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
    StripeRequirements reqs,
  ) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final fieldNames = _collectFieldNames(reqs.sections);

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
          const SizedBox(height: 4),
          Padding(
            padding: const EdgeInsets.only(left: 28),
            child: Text(
              l10n.stripeRequirementsDesc,
              style: TextStyle(
                fontSize: 12,
                color: isDark
                    ? const Color(0xFFFBBF24).withValues(alpha: 0.8)
                    : const Color(0xFF92400E).withValues(alpha: 0.8),
              ),
            ),
          ),
          if (fieldNames.isNotEmpty) ...[
            const SizedBox(height: 8),
            ...fieldNames.map(
              (name) => Padding(
                padding: const EdgeInsets.only(left: 28, bottom: 2),
                child: Text(
                  '\u2022 $name',
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
        ],
      ),
    );
  }

  /// Collect human-readable field names from requirement sections.
  List<String> _collectFieldNames(List<RequirementSection> sections) {
    final names = <String>[];
    for (final section in sections) {
      for (final field in section.fields) {
        names.add(_humanizeKey(field.labelKey));
      }
    }
    return names;
  }

  /// Convert a camelCase key to a readable label.
  String _humanizeKey(String key) {
    if (key.isEmpty) return key;
    // Insert space before capitals, replace underscores
    final spaced = key
        .replaceAllMapped(
          RegExp(r'([A-Z])'),
          (m) => ' ${m.group(0)}',
        )
        .replaceAll('_', ' ')
        .trim();
    // Capitalize first letter of each word
    return spaced
        .split(' ')
        .where((w) => w.isNotEmpty)
        .map((w) => '${w[0].toUpperCase()}${w.substring(1)}')
        .join(' ');
  }
}

/// Builds a map of field key -> error message from Stripe requirements.
///
/// Used by the payment info screen to highlight fields that need attention.
Map<String, String> buildFieldErrors(StripeRequirements? reqs) {
  if (reqs == null || !reqs.hasRequirements) return const {};
  final errors = <String, String>{};
  for (final section in reqs.sections) {
    for (final field in section.fields) {
      errors[field.key] = 'This field is required by Stripe';
    }
  }
  return errors;
}
