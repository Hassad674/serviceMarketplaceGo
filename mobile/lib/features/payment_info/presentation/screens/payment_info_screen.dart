import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/country_field_spec.dart';
import 'package:marketplace_mobile/features/payment_info/lib/form_data_mapper.dart';
import '../../types/payment_info.dart';
import '../../domain/entities/identity_document_entity.dart';
import '../providers/country_fields_provider.dart';
import '../providers/identity_document_provider.dart';
import '../providers/payment_info_provider.dart';
import '../widgets/business_persons_section.dart';
import '../widgets/country_selector_section.dart';
import '../widgets/dynamic_section.dart';
import '../widgets/extra_fields_section.dart';
import '../widgets/identity_verification_section.dart';
import '../widgets/payment_info_widgets.dart';
import '../widgets/stripe_requirements_banner.dart';

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

/// Payment information form connected to the backend API.
///
/// Accessible to agency and provider roles. Loads existing data on mount,
/// renders dynamic sections based on the selected country.
class PaymentInfoScreen extends ConsumerStatefulWidget {
  const PaymentInfoScreen({super.key});

  @override
  ConsumerState<PaymentInfoScreen> createState() => _PaymentInfoScreenState();
}

class _PaymentInfoScreenState extends ConsumerState<PaymentInfoScreen> {
  var _data = const PaymentInfoFormData();
  bool _saved = false;
  bool _saving = false;
  bool _populated = false;
  String? _stripeError;
  String? _lastUserId;

  void _onValueChanged(String key, String value) {
    setState(() {
      _data = _data.copyWith(
        values: {..._data.values, key: value},
      );
      _saved = false;
    });
  }

  void _onCountryChanged(String country) {
    setState(() {
      _data = _data.copyWith(country: country, values: {}, extraFields: {});
      _saved = false;
    });
  }

  void _onBusinessToggled(bool isBusiness) {
    setState(() {
      _data = _data.copyWith(isBusiness: isBusiness, values: {});
      _saved = false;
    });
  }

  void _onFormDataChanged(PaymentInfoFormData Function(PaymentInfoFormData) fn) {
    setState(() {
      _data = fn(_data);
      _saved = false;
    });
  }

  Future<void> _save(List<FieldSection>? sections, String email) async {
    setState(() => _saving = true);
    try {
      final json = valuesToFlatData(_data, sections, email: email);

      final repo = ref.read(paymentInfoRepositoryProvider);
      final savedInfo = await repo.savePaymentInfo(json);
      ref.invalidate(paymentInfoProvider);
      ref.invalidate(paymentInfoStatusProvider);
      if (mounted) {
        setState(() {
          _saved = true;
          _stripeError = savedInfo.stripeError.isNotEmpty
              ? savedInfo.stripeError
              : null;
        });
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to save: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final asyncInfo = ref.watch(paymentInfoProvider);

    final authState = ref.watch(authProvider);
    final userEmail = authState.user?['email'] as String? ?? '';

    // Reset form when user changes (logout → new account)
    final currentUserId = userEmail;
    if (_lastUserId != null && _lastUserId != currentUserId) {
      _populated = false;
      _data = const PaymentInfoFormData();
      _saved = false;
      _stripeError = null;
    }
    _lastUserId = currentUserId;

    // Populate form when data arrives (only once per user).
    if (!_populated && asyncInfo.hasValue && asyncInfo.value != null) {
      _populated = true;
      _data = responseToFormData(asyncInfo.value!);
      // Pre-fill email from auth user (not stored in entity)
      if (_data.values['individual.email']?.isEmpty ?? true) {
        _data = _data.copyWith(
          values: {..._data.values, 'individual.email': userEmail},
        );
      }
      _saved = true;
    }

    // Show stripe error from loaded entity (if any)
    final loadedStripeError =
        asyncInfo.valueOrNull?.stripeError ?? '';
    final effectiveStripeError = _stripeError ??
        (loadedStripeError.isNotEmpty ? loadedStripeError : null);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.paymentInfoTitle),
      ),
      body: asyncInfo.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Error: $error'),
              const SizedBox(height: 8),
              ElevatedButton(
                onPressed: () => ref.invalidate(paymentInfoProvider),
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
        data: (_) => _buildForm(l10n, theme, effectiveStripeError, userEmail),
      ),
    );
  }

  Widget _buildForm(AppLocalizations l10n, ThemeData theme, String? stripeErr, String userEmail) {
    final businessType = _data.isBusiness ? 'company' : 'individual';
    final hasCountry = _data.country.length == 2;

    // Fetch dynamic sections for the selected country
    final asyncFields = hasCountry
        ? ref.watch(countryFieldsProvider(
            CountryFieldsKey(_data.country, businessType)))
        : null;

    final countryFields = asyncFields?.valueOrNull;
    final allSections = countryFields?.sections ?? <FieldSection>[];
    final entitySections =
        allSections.where((s) => s.id != 'bank').toList();
    final bankSection =
        allSections.where((s) => s.id == 'bank').firstOrNull;
    final extraFields = countryFields?.extraFields ?? <FieldSpec>[];
    final personRoles = countryFields?.personRoles ?? <String>[];
    final docsRequired = countryFields?.individualDocRequired ?? false;

    // Build field errors and warnings from Stripe requirements
    final asyncReqs = ref.watch(stripeRequirementsProvider);
    final reqs = asyncReqs.valueOrNull;
    final fieldErrors = buildFieldErrors(reqs);
    final fieldWarnings = buildFieldWarnings(reqs);

    // Fetch existing identity documents for status display
    final asyncDocs = ref.watch(identityDocumentsProvider);
    final existingDocs = asyncDocs.valueOrNull ?? <IdentityDocument>[];

    final valid = hasCountry && isFormValid(_data, allSections);

    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Subtitle
            Text(
              l10n.paymentInfoSubtitle,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
              ),
            ),
            const SizedBox(height: 16),

            // Stripe requirements banner
            if (_saved) const StripeRequirementsBanner(),
            if (_saved) const SizedBox(height: 16),

            // Status banner + stripe error
            PaymentStatusBanner(saved: _saved),
            const SizedBox(height: 16),

            // Country selector
            CountrySelectorSection(
              value: _data.country,
              onChanged: _onCountryChanged,
            ),
            const SizedBox(height: 16),

            // Business toggle
            PaymentBusinessToggle(
              value: _data.isBusiness,
              onChanged: _onBusinessToggled,
            ),
            const SizedBox(height: 16),

            // Activity sector (always visible, not from country specs)
            ActivitySectorSection(data: _data, onUpdate: _onFormDataChanged),
            const SizedBox(height: 16),

            // Dynamic sections or placeholder
            if (!hasCountry)
              _Placeholder(text: 'Select your country to see required fields')
            else if (asyncFields != null && asyncFields.isLoading)
              const Center(child: Padding(
                padding: EdgeInsets.all(24),
                child: CircularProgressIndicator(),
              ))
            else ...[
              // Entity sections (personal info, company, etc.)
              ...entitySections.map((section) => Padding(
                    padding: const EdgeInsets.only(bottom: 16),
                    child: DynamicSection(
                      section: section,
                      values: _data.values,
                      onChanged: _onValueChanged,
                      fieldErrors: fieldErrors,
                      fieldWarnings: fieldWarnings,
                      documents: existingDocs,
                      countryCode: _data.country,
                    ),
                  )),

              // Business persons (when business + roles beyond representative)
              if (_data.isBusiness &&
                  personRoles.any((r) => r != 'representative'))
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: BusinessPersonsSection(
                    data: _data,
                    onUpdate: _onFormDataChanged,
                  ),
                ),

              // Bank section
              if (bankSection != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: DynamicSection(
                    section: bankSection,
                    values: _data.values,
                    onChanged: _onValueChanged,
                    fieldErrors: fieldErrors,
                    fieldWarnings: fieldWarnings,
                    documents: existingDocs,
                    countryCode: _data.country,
                  ),
                ),

              // Extra fields (country-specific)
              if (extraFields.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: ExtraFieldsSection(
                    fields: extraFields,
                    values: _data.values,
                    onChanged: _onValueChanged,
                    fieldErrors: fieldErrors,
                    countryCode: _data.country,
                  ),
                ),

              // Identity verification
              if (docsRequired) ...[
                const IdentityVerificationSection(),
                const SizedBox(height: 8),
              ],
            ],

            // Stripe error just above save
            if (stripeErr != null) ...[
              _StripeErrorBanner(message: stripeErr),
              const SizedBox(height: 12),
            ],

            // Save button
            PaymentInfoSaveButton(
              isValid: valid,
              isSaving: _saving,
              onSave: () => _save(allSections, userEmail),
            ),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Small helper widgets
// ---------------------------------------------------------------------------

class _Placeholder extends StatelessWidget {
  const _Placeholder({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: Theme.of(context).dividerColor),
      ),
      child: Text(
        text,
        textAlign: TextAlign.center,
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.5),
            ),
      ),
    );
  }
}

class _StripeErrorBanner extends StatelessWidget {
  const _StripeErrorBanner({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.error.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: theme.colorScheme.error.withValues(alpha: 0.3),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.warning_amber_rounded,
              size: 20, color: theme.colorScheme.error),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Stripe error',
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: theme.colorScheme.error,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  message,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.error.withValues(alpha: 0.8),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
