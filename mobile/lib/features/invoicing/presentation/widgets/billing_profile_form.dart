import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/billing_profile.dart';
import '../../domain/entities/billing_profile_snapshot.dart';
import '../providers/invoicing_providers.dart';
import 'billing_form_atoms.dart';
import 'billing_form_status.dart';
import 'billing_section_address.dart';
import 'billing_section_fiscal.dart';
import 'billing_section_legal_identity.dart';
import '../../../../core/theme/app_palette.dart';

// EU member states whose VAT numbers can be validated through VIES.
// Domain predicate kept separate from the country selector list — the
// selector renders every country Stripe Connect supports (43 entries),
// but only EU countries trigger the autoliquidation regime.
const Set<String> _kEuVatCountryCodes = <String>{
  'FR', 'BE', 'LU', 'DE', 'AT', 'NL', 'IT', 'ES', 'PT', 'IE',
  'DK', 'SE', 'FI', 'PL', 'CZ', 'GR', 'EE', 'LV', 'LT', 'SK',
  'SI', 'HU', 'BG', 'HR', 'RO', 'CY', 'MT',
};

bool _isEuCountry(String code) => _kEuVatCountryCodes.contains(code);

/// Editable form for the org's billing profile.
///
/// Reads the current snapshot through `billingProfileProvider`, hydrates
/// a local controlled state once, then pushes back through three
/// independent mutations:
///   - update (PUT)              — Save button
///   - sync from Stripe (POST)   — "Sync depuis Stripe" button (only
///     when never synced)
///   - validate VAT (POST)       — "Valider mon n° TVA" button
///
/// Each mutation has its own pending state so a VIES failure never
/// blocks a save and vice-versa.
class BillingProfileForm extends ConsumerStatefulWidget {
  const BillingProfileForm({super.key, this.onSaved});

  /// Fires once after a successful save AND only when the resulting
  /// profile passes server-side completeness. The screen wrapper uses
  /// it to route back to the page that triggered the gate (wallet,
  /// subscribe). Save attempts that leave the profile still incomplete
  /// keep the user on the form so they can fix the missing fields.
  final VoidCallback? onSaved;

  @override
  ConsumerState<BillingProfileForm> createState() =>
      _BillingProfileFormState();
}

class _BillingProfileFormState extends ConsumerState<BillingProfileForm> {
  final _formKey = GlobalKey<FormState>();

  ProfileType? _profileType;
  late final TextEditingController _legalName;
  late final TextEditingController _tradingName;
  late final TextEditingController _legalForm;
  late final TextEditingController _taxId;
  late final TextEditingController _vatNumber;
  late final TextEditingController _addressLine1;
  late final TextEditingController _addressLine2;
  late final TextEditingController _postalCode;
  late final TextEditingController _city;
  String _country = '';

  bool _hydrated = false;
  bool _saving = false;
  bool _syncing = false;
  bool _validatingVat = false;

  String? _saveError;
  String? _saveSuccess;
  String? _syncError;
  String? _vatError;
  DateTime? _vatValidatedAt;
  String? _vatRegisteredName;

  @override
  void initState() {
    super.initState();
    _legalName = TextEditingController();
    _tradingName = TextEditingController();
    _legalForm = TextEditingController();
    _taxId = TextEditingController();
    _vatNumber = TextEditingController();
    _addressLine1 = TextEditingController();
    _addressLine2 = TextEditingController();
    _postalCode = TextEditingController();
    _city = TextEditingController();
  }

  @override
  void dispose() {
    _legalName.dispose();
    _tradingName.dispose();
    _legalForm.dispose();
    _taxId.dispose();
    _vatNumber.dispose();
    _addressLine1.dispose();
    _addressLine2.dispose();
    _postalCode.dispose();
    _city.dispose();
    super.dispose();
  }

  void _hydrate(BillingProfile p) {
    _profileType = p.profileType;
    _legalName.text = p.legalName;
    _tradingName.text = p.tradingName;
    _legalForm.text = p.legalForm;
    _taxId.text = p.taxId;
    _vatNumber.text = p.vatNumber;
    _addressLine1.text = p.addressLine1;
    _addressLine2.text = p.addressLine2;
    _postalCode.text = p.postalCode;
    _city.text = p.city;
    _country = p.country;
    _vatValidatedAt = p.vatValidatedAt;
    _hydrated = true;
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(billingProfileProvider);
    return async.when(
      loading: () => const BillingFormLoader(),
      error: (_, __) => const BillingFormLoadError(),
      data: (snapshot) {
        if (!_hydrated) {
          _hydrate(snapshot.profile);
        }
        return _buildForm(snapshot);
      },
    );
  }

  Widget _buildForm(BillingProfileSnapshot snapshot) {
    final theme = Theme.of(context);
    final isFr = _country == 'FR';
    final isEu = _isEuCountry(_country);
    final isBusiness = _profileType == ProfileType.business;

    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (!snapshot.isComplete && snapshot.missingFields.isNotEmpty) ...[
            BillingMissingBanner(fields: snapshot.missingFields),
            const SizedBox(height: 16),
          ],
          BillingStripeSyncRow(
            syncedAt: snapshot.profile.syncedFromKycAt,
            syncing: _syncing,
            onSync: _onSync,
            error: _syncError,
          ),
          const SizedBox(height: 16),
          BillingSection(
            title: 'Pays',
            subtitle:
                "Choisis d'abord ton pays — les autres champs s'adaptent en conséquence (SIRET pour la France, n° TVA intracom pour l'UE, adresse seule ailleurs).",
            child: BillingCountryDropdown(
              value: _country,
              onChanged: (v) => setState(() => _country = v ?? ''),
            ),
          ),
          const SizedBox(height: 12),
          BillingSection(
            title: 'Type de profil',
            child: BillingProfileTypeRadio(
              value: _profileType,
              onChanged: (v) => setState(() => _profileType = v),
            ),
          ),
          const SizedBox(height: 12),
          BillingLegalIdentitySection(
            isBusiness: isBusiness,
            legalName: _legalName,
            tradingName: _tradingName,
            legalForm: _legalForm,
          ),
          const SizedBox(height: 12),
          BillingFiscalSection(
            isFr: isFr,
            isEu: isEu,
            taxId: _taxId,
            vatNumber: _vatNumber,
            vatValidatedAt: _vatValidatedAt,
            vatRegisteredName: _vatRegisteredName,
            validatingVat: _validatingVat,
            vatError: _vatError,
            onValidateVat: _onValidateVat,
          ),
          const SizedBox(height: 12),
          BillingAddressSection(
            addressLine1: _addressLine1,
            addressLine2: _addressLine2,
            postalCode: _postalCode,
            city: _city,
          ),
          // The "Email de facturation" section was removed — invoices
          // default to the org owner's account email server-side.
          const SizedBox(height: 20),
          if (_saveError != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                _saveError!,
                style: TextStyle(
                  color: theme.colorScheme.error,
                  fontSize: 13,
                ),
              ),
            ),
          if (_saveSuccess != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                _saveSuccess!,
                style:
                    const TextStyle(color: AppPalette.green500, fontSize: 13),
              ),
            ),
          ElevatedButton(
            onPressed: _saving ? null : _onSave,
            style: ElevatedButton.styleFrom(
              backgroundColor: AppPalette.rose500,
              foregroundColor: Colors.white,
              minimumSize: const Size(double.infinity, 48),
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              ),
            ),
            child: _saving
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      valueColor: AlwaysStoppedAnimation(Colors.white),
                    ),
                  )
                : const Text(
                    'Enregistrer',
                    style: TextStyle(fontWeight: FontWeight.w600),
                  ),
          ),
        ],
      ),
    );
  }

  Future<void> _onSave() async {
    if (_profileType == null) {
      setState(() => _saveError = 'Sélectionne un type de profil');
      return;
    }
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    setState(() {
      _saving = true;
      _saveError = null;
      _saveSuccess = null;
    });
    try {
      final useCase = ref.read(updateBillingProfileUseCaseProvider);
      final snapshot = await useCase(
        UpdateBillingProfileInput(
          profileType: _profileType!,
          legalName: _legalName.text.trim(),
          tradingName: _tradingName.text.trim(),
          legalForm: _legalForm.text.trim(),
          taxId: _taxId.text.trim(),
          vatNumber: _vatNumber.text.trim(),
          addressLine1: _addressLine1.text.trim(),
          addressLine2: _addressLine2.text.trim(),
          postalCode: _postalCode.text.trim(),
          city: _city.text.trim(),
          country: _country,
          // Empty: the backend defaults invoicing_email to the org
          // owner's account email when the row's value is empty.
          invoicingEmail: '',
        ),
      );
      ref.invalidate(billingProfileProvider);
      if (!mounted) return;
      setState(() {
        _saveSuccess = 'Profil enregistré.';
      });
      if (snapshot.isComplete) {
        widget.onSaved?.call();
      }
    } catch (e) {
      debugPrint('billing profile save failed: $e');
      if (!mounted) return;
      setState(() {
        _saveError = "L'enregistrement a échoué. Réessaie.";
      });
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _onSync() async {
    setState(() {
      _syncing = true;
      _syncError = null;
    });
    try {
      final useCase = ref.read(syncBillingProfileUseCaseProvider);
      final snapshot = await useCase();
      _hydrate(snapshot.profile);
      ref.invalidate(billingProfileProvider);
      if (!mounted) return;
      setState(() {});
    } catch (e) {
      debugPrint('stripe sync failed: $e');
      if (!mounted) return;
      setState(() {
        _syncError = 'La synchronisation Stripe a échoué. Réessaie ou '
            'complète manuellement.';
      });
    } finally {
      if (mounted) setState(() => _syncing = false);
    }
  }

  Future<void> _onValidateVat() async {
    setState(() {
      _validatingVat = true;
      _vatError = null;
      _vatRegisteredName = null;
    });
    try {
      final useCase = ref.read(validateVATUseCaseProvider);
      final result = await useCase();
      ref.invalidate(billingProfileProvider);
      if (!mounted) return;
      setState(() {
        if (result.valid) {
          _vatValidatedAt = result.checkedAt;
          _vatRegisteredName = result.registeredName;
        } else {
          _vatError = 'Numéro non reconnu par VIES';
        }
      });
    } catch (e) {
      debugPrint('vat validation failed: $e');
      if (!mounted) return;
      setState(() {
        _vatError = 'Numéro non reconnu par VIES';
      });
    } finally {
      if (mounted) setState(() => _validatingVat = false);
    }
  }
}
