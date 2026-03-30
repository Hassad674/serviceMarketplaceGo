import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/payment_info_entity.dart';
import '../../types/payment_info.dart';
import '../providers/payment_info_provider.dart';
import '../widgets/business_persons_section.dart';
import '../widgets/identity_verification_section.dart';
import '../widgets/payment_info_widgets.dart';
import '../widgets/stripe_requirements_banner.dart';

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

/// Payment information form connected to the backend API.
///
/// Accessible to agency and provider roles. Loads existing data on mount,
/// displays personal info, optional business info, and bank account fields.
class PaymentInfoScreen extends ConsumerStatefulWidget {
  const PaymentInfoScreen({super.key});

  @override
  ConsumerState<PaymentInfoScreen> createState() => _PaymentInfoScreenState();
}

class _PaymentInfoScreenState extends ConsumerState<PaymentInfoScreen> {
  var _data = const PaymentInfoFormData();
  bool _saved = false;
  bool _saving = false;
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    // Force refresh payment info on screen mount
    Future.microtask(() => ref.invalidate(paymentInfoProvider));
  }

  void _update(PaymentInfoFormData Function(PaymentInfoFormData) updater) {
    setState(() {
      _data = updater(_data);
      _saved = false;
    });
  }

  bool get _isValid {
    final personalOk = _data.firstName.trim().isNotEmpty &&
        _data.lastName.trim().isNotEmpty &&
        _data.dateOfBirth.isNotEmpty &&
        _data.nationality.isNotEmpty &&
        _data.address.trim().isNotEmpty &&
        _data.city.trim().isNotEmpty &&
        _data.postalCode.trim().isNotEmpty;
    if (!personalOk) return false;

    if (_data.isBusiness) {
      final bizOk = _data.businessRole != null &&
          _data.businessName.trim().isNotEmpty &&
          _data.businessAddress.trim().isNotEmpty &&
          _data.businessCity.trim().isNotEmpty &&
          _data.businessPostalCode.trim().isNotEmpty &&
          _data.businessCountry.isNotEmpty &&
          _data.taxId.trim().isNotEmpty;
      if (!bizOk) return false;
    }

    final bankOk = _data.accountHolder.trim().isNotEmpty &&
        _data.bankCountry.isNotEmpty &&
        (_data.bankMode == BankAccountMode.iban
            ? _data.iban.trim().isNotEmpty
            : _data.accountNumber.trim().isNotEmpty &&
                _data.routingNumber.trim().isNotEmpty);
    return bankOk;
  }

  void _populateFromEntity(PaymentInfo info) {
    final hasIban = info.iban.isNotEmpty;
    _data = PaymentInfoFormData(
      isBusiness: info.isBusiness,
      firstName: info.firstName,
      lastName: info.lastName,
      dateOfBirth: info.dateOfBirth,
      nationality: info.nationality,
      address: info.address,
      city: info.city,
      postalCode: info.postalCode,
      phone: info.phone,
      activitySector: info.activitySector,
      businessRole: _parseBusinessRole(info.roleInCompany),
      businessName: info.businessName,
      businessAddress: info.businessAddress,
      businessCity: info.businessCity,
      businessPostalCode: info.businessPostalCode,
      businessCountry: info.businessCountry,
      taxId: info.taxId,
      vatNumber: info.vatNumber,
      isSelfRepresentative: info.isSelfRepresentative,
      isSelfDirector: info.isSelfDirector,
      noMajorOwners: info.noMajorOwners,
      isSelfExecutive: info.isSelfExecutive,
      bankMode: hasIban ? BankAccountMode.iban : BankAccountMode.local,
      iban: info.iban,
      bic: info.bic,
      accountNumber: info.accountNumber,
      routingNumber: info.routingNumber,
      accountHolder: info.accountHolder,
      bankCountry: info.bankCountry,
    );
    _saved = true;
  }

  BusinessRole? _parseBusinessRole(String value) {
    const mapping = {
      'owner': BusinessRole.owner,
      'ceo': BusinessRole.ceo,
      'director': BusinessRole.director,
      'partner': BusinessRole.partner,
      'other': BusinessRole.other,
    };
    return mapping[value.toLowerCase()];
  }

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final repo = ref.read(paymentInfoRepositoryProvider);
      await repo.savePaymentInfo(_formDataToJson());
      ref.invalidate(paymentInfoProvider);
      ref.invalidate(paymentInfoStatusProvider);
      if (mounted) setState(() => _saved = true);
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

  Map<String, dynamic> _formDataToJson() {
    final authState = ref.read(authProvider);
    final userEmail = authState.user?['email'] as String? ?? '';

    final json = <String, dynamic>{
      'first_name': _data.firstName,
      'last_name': _data.lastName,
      'date_of_birth': _data.dateOfBirth,
      'nationality': _data.nationality,
      'address': _data.address,
      'city': _data.city,
      'postal_code': _data.postalCode,
      'phone': _data.phone,
      'activity_sector': _data.activitySector,
      'email': userEmail,
      'is_business': _data.isBusiness,
      'business_name': _data.businessName,
      'business_address': _data.businessAddress,
      'business_city': _data.businessCity,
      'business_postal_code': _data.businessPostalCode,
      'business_country': _data.businessCountry,
      'tax_id': _data.taxId,
      'vat_number': _data.vatNumber,
      'role_in_company': _data.businessRole?.name ?? '',
      'is_self_representative': _data.isSelfRepresentative,
      'is_self_director': _data.isSelfDirector,
      'no_major_owners': _data.noMajorOwners,
      'is_self_executive': _data.isSelfExecutive,
      'iban': _data.iban,
      'bic': _data.bic,
      'account_number': _data.accountNumber,
      'routing_number': _data.routingNumber,
      'account_holder': _data.accountHolder,
      'bank_country': _data.bankCountry,
    };

    if (_data.businessPersons.isNotEmpty) {
      json['business_persons'] =
          _data.businessPersons.map((p) => p.toJson()).toList();
    }

    return json;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final asyncInfo = ref.watch(paymentInfoProvider);

    // Populate form from existing data once
    asyncInfo.whenData((info) {
      if (!_initialized && info != null) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && !_initialized) {
            setState(() {
              _populateFromEntity(info);
              _initialized = true;
            });
          }
        });
      } else if (!_initialized && !asyncInfo.isLoading) {
        _initialized = true;
      }
    });

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
        data: (_) => _buildForm(l10n, theme),
      ),
    );
  }

  Widget _buildForm(AppLocalizations l10n, ThemeData theme) {
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
            const StripeRequirementsBanner(),
            const SizedBox(height: 16),

            // Status banner
            PaymentStatusBanner(saved: _saved),
            const SizedBox(height: 16),

            // Business toggle
            PaymentBusinessToggle(
              value: _data.isBusiness,
              onChanged: (v) => _update((d) => d.copyWith(isBusiness: v)),
            ),
            const SizedBox(height: 16),

            // Personal info (includes phone)
            _PersonalInfoSection(data: _data, onUpdate: _update),
            const SizedBox(height: 16),

            // Activity sector
            ActivitySectorSection(data: _data, onUpdate: _update),
            const SizedBox(height: 16),

            // Business info (animated)
            AnimatedSize(
              duration: const Duration(milliseconds: 300),
              curve: Curves.easeOut,
              alignment: Alignment.topCenter,
              child: _data.isBusiness
                  ? Padding(
                      padding: const EdgeInsets.only(bottom: 16),
                      child: Column(
                        children: [
                          _BusinessInfoSection(
                            data: _data,
                            onUpdate: _update,
                          ),
                          const SizedBox(height: 16),
                          BusinessPersonsSection(
                            data: _data,
                            onUpdate: _update,
                          ),
                        ],
                      ),
                    )
                  : const SizedBox.shrink(),
            ),

            // Bank account
            _BankAccountSection(data: _data, onUpdate: _update),
            const SizedBox(height: 24),

            // Identity verification
            const IdentityVerificationSection(),
            const SizedBox(height: 24),

            // Save button
            PaymentInfoSaveButton(
              isValid: _isValid,
              isSaving: _saving,
              onSave: _save,
            ),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Personal info section
// ---------------------------------------------------------------------------

class _PersonalInfoSection extends StatelessWidget {
  const _PersonalInfoSection({
    required this.data,
    required this.onUpdate,
  });

  final PaymentInfoFormData data;
  final FormUpdater onUpdate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final sectionTitle = data.isBusiness
        ? l10n.paymentInfoLegalRep
        : l10n.paymentInfoPersonalInfo;

    return PaymentSectionCard(
      title: sectionTitle,
      children: [
        PaymentFormField(
          label: l10n.paymentInfoFirstName,
          value: data.firstName,
          onChanged: (v) => onUpdate((d) => d.copyWith(firstName: v)),
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoLastName,
          value: data.lastName,
          onChanged: (v) => onUpdate((d) => d.copyWith(lastName: v)),
          required: true,
        ),
        PaymentDateField(
          label: l10n.paymentInfoDob,
          value: data.dateOfBirth,
          onChanged: (v) => onUpdate((d) => d.copyWith(dateOfBirth: v)),
        ),
        PaymentCountryDropdown(
          label: l10n.paymentInfoNationality,
          value: data.nationality,
          onChanged: (code) {
            final mode = ibanCountryCodes.contains(code)
                ? BankAccountMode.iban
                : BankAccountMode.local;
            onUpdate((d) => d.copyWith(nationality: code, bankMode: mode));
          },
        ),
        PaymentFormField(
          label: l10n.paymentInfoAddress,
          value: data.address,
          onChanged: (v) => onUpdate((d) => d.copyWith(address: v)),
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoCity,
          value: data.city,
          onChanged: (v) => onUpdate((d) => d.copyWith(city: v)),
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoPostalCode,
          value: data.postalCode,
          onChanged: (v) => onUpdate((d) => d.copyWith(postalCode: v)),
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoPhone,
          value: data.phone,
          onChanged: (v) => onUpdate((d) => d.copyWith(phone: v)),
          keyboardType: TextInputType.phone,
          placeholder: '+33 6 12 34 56 78',
          required: true,
        ),
        if (data.isBusiness)
          PaymentRoleDropdown(
            value: data.businessRole,
            onChanged: (r) => onUpdate((d) => d.copyWith(businessRole: r)),
          ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Business info section
// ---------------------------------------------------------------------------

class _BusinessInfoSection extends StatelessWidget {
  const _BusinessInfoSection({
    required this.data,
    required this.onUpdate,
  });

  final PaymentInfoFormData data;
  final FormUpdater onUpdate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return PaymentSectionCard(
      title: l10n.paymentInfoBusinessInfo,
      children: [
        PaymentFormField(
          label: l10n.paymentInfoBusinessName,
          value: data.businessName,
          onChanged: (v) => onUpdate((d) => d.copyWith(businessName: v)),
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoBusinessAddress,
          value: data.businessAddress,
          onChanged: (v) => onUpdate((d) => d.copyWith(businessAddress: v)),
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoBusinessCity,
          value: data.businessCity,
          onChanged: (v) => onUpdate((d) => d.copyWith(businessCity: v)),
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoBusinessPostalCode,
          value: data.businessPostalCode,
          onChanged: (v) =>
              onUpdate((d) => d.copyWith(businessPostalCode: v)),
          required: true,
        ),
        PaymentCountryDropdown(
          label: l10n.paymentInfoBusinessCountry,
          value: data.businessCountry,
          onChanged: (code) =>
              onUpdate((d) => d.copyWith(businessCountry: code)),
        ),
        PaymentFormField(
          label: l10n.paymentInfoTaxId,
          value: data.taxId,
          onChanged: (v) => onUpdate((d) => d.copyWith(taxId: v)),
          placeholder: l10n.paymentInfoTaxIdHint,
          required: true,
        ),
        PaymentFormField(
          label: l10n.paymentInfoVatNumber,
          value: data.vatNumber,
          onChanged: (v) => onUpdate((d) => d.copyWith(vatNumber: v)),
          placeholder: l10n.paymentInfoVatNumberHint,
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Bank account section
// ---------------------------------------------------------------------------

class _BankAccountSection extends StatelessWidget {
  const _BankAccountSection({
    required this.data,
    required this.onUpdate,
  });

  final PaymentInfoFormData data;
  final FormUpdater onUpdate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final isIban = data.bankMode == BankAccountMode.iban;

    return PaymentSectionCard(
      title: l10n.paymentInfoBankAccount,
      children: [
        if (isIban) ...[
          PaymentFormField(
            label: l10n.paymentInfoIban,
            value: data.iban,
            onChanged: (v) => onUpdate((d) => d.copyWith(iban: v)),
            placeholder: l10n.paymentInfoIbanHint,
            required: true,
          ),
          PaymentFormField(
            label: l10n.paymentInfoBic,
            value: data.bic,
            onChanged: (v) => onUpdate((d) => d.copyWith(bic: v)),
            placeholder: l10n.paymentInfoBicHint,
          ),
          PaymentIbanHelpText(helpText: l10n.paymentInfoIbanHelp),
        ] else ...[
          PaymentFormField(
            label: l10n.paymentInfoAccountNumber,
            value: data.accountNumber,
            onChanged: (v) => onUpdate((d) => d.copyWith(accountNumber: v)),
            required: true,
          ),
          PaymentFormField(
            label: l10n.paymentInfoRoutingNumber,
            value: data.routingNumber,
            onChanged: (v) => onUpdate((d) => d.copyWith(routingNumber: v)),
            required: true,
          ),
        ],
        PaymentNoIbanCheckbox(
          value: !isIban,
          label: l10n.paymentInfoNoIban,
          onChanged: (checked) => onUpdate(
            (d) => d.copyWith(
              bankMode:
                  checked ? BankAccountMode.local : BankAccountMode.iban,
            ),
          ),
        ),
        PaymentCountryDropdown(
          label: l10n.paymentInfoBankCountry,
          value: data.bankCountry,
          onChanged: (code) =>
              onUpdate((d) => d.copyWith(bankCountry: code)),
        ),
        PaymentFormField(
          label: l10n.paymentInfoAccountHolder,
          value: data.accountHolder,
          onChanged: (v) => onUpdate((d) => d.copyWith(accountHolder: v)),
          required: true,
        ),
      ],
    );
  }
}

