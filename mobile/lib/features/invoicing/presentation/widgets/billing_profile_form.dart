import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../domain/entities/billing_profile.dart';
import '../../domain/entities/billing_profile_snapshot.dart';
import '../../domain/entities/missing_field.dart';
import '../providers/invoicing_providers.dart';
import '_missing_fields_copy.dart';

// EU member states whose VAT numbers can be validated through VIES.
// 2-letter ISO codes — mirror `web/src/features/invoicing/components/eu-countries.ts`.
const List<({String code, String label})> _kEuCountries = [
  (code: 'FR', label: 'France'),
  (code: 'BE', label: 'Belgique'),
  (code: 'DE', label: 'Allemagne'),
  (code: 'ES', label: 'Espagne'),
  (code: 'IT', label: 'Italie'),
  (code: 'NL', label: 'Pays-Bas'),
  (code: 'PT', label: 'Portugal'),
  (code: 'LU', label: 'Luxembourg'),
  (code: 'IE', label: 'Irlande'),
  (code: 'AT', label: 'Autriche'),
  (code: 'PL', label: 'Pologne'),
  (code: 'RO', label: 'Roumanie'),
  (code: 'CZ', label: 'République tchèque'),
  (code: 'SE', label: 'Suède'),
  (code: 'FI', label: 'Finlande'),
  (code: 'DK', label: 'Danemark'),
  (code: 'EL', label: 'Grèce'),
  (code: 'HU', label: 'Hongrie'),
  (code: 'BG', label: 'Bulgarie'),
  (code: 'HR', label: 'Croatie'),
  (code: 'SI', label: 'Slovénie'),
  (code: 'SK', label: 'Slovaquie'),
  (code: 'EE', label: 'Estonie'),
  (code: 'LV', label: 'Lettonie'),
  (code: 'LT', label: 'Lituanie'),
  (code: 'MT', label: 'Malte'),
  (code: 'CY', label: 'Chypre'),
];

bool _isEuCountry(String code) =>
    _kEuCountries.any((c) => c.code == code);

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
  const BillingProfileForm({super.key});

  @override
  ConsumerState<BillingProfileForm> createState() => _BillingProfileFormState();
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
  late final TextEditingController _invoicingEmail;

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
    _invoicingEmail = TextEditingController();
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
    _invoicingEmail.dispose();
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
    _invoicingEmail.text = p.invoicingEmail;
    _vatValidatedAt = p.vatValidatedAt;
    _hydrated = true;
  }

  @override
  Widget build(BuildContext context) {
    final async = ref.watch(billingProfileProvider);
    return async.when(
      loading: () => const _Loader(),
      error: (_, __) => const _LoadError(),
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
          if (!snapshot.isComplete && snapshot.missingFields.isNotEmpty)
            _MissingBanner(fields: snapshot.missingFields),
          if (!snapshot.isComplete && snapshot.missingFields.isNotEmpty)
            const SizedBox(height: 16),
          _StripeSyncRow(
            syncedAt: snapshot.profile.syncedFromKycAt,
            syncing: _syncing,
            onSync: _onSync,
            error: _syncError,
          ),
          const SizedBox(height: 16),
          _Section(
            title: 'Type de profil',
            child: _ProfileTypeRadio(
              value: _profileType,
              onChanged: (v) => setState(() => _profileType = v),
            ),
          ),
          const SizedBox(height: 12),
          _Section(
            title: 'Identité légale',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _LabeledField(
                  label: 'Raison sociale ou nom légal',
                  controller: _legalName,
                  validator: _required,
                ),
                if (isBusiness) ...[
                  const SizedBox(height: 12),
                  _LabeledField(
                    label: 'Nom commercial (optionnel)',
                    controller: _tradingName,
                  ),
                  const SizedBox(height: 12),
                  _LabeledField(
                    label: 'Forme juridique',
                    controller: _legalForm,
                    hint: 'SAS, SARL, EURL, etc.',
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: 12),
          _Section(
            title: 'Identifiants fiscaux',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (isFr)
                  _LabeledField(
                    label: 'Numéro SIRET',
                    controller: _taxId,
                    hint: '14 chiffres, sans espace',
                    keyboardType: TextInputType.number,
                    maxLength: 14,
                    validator: _siret,
                  )
                else
                  _LabeledField(
                    label: 'Identifiant fiscal',
                    controller: _taxId,
                  ),
                if (isEu) ...[
                  const SizedBox(height: 12),
                  _VatRow(
                    controller: _vatNumber,
                    validatedAt: _vatValidatedAt,
                    registeredName: _vatRegisteredName,
                    validating: _validatingVat,
                    error: _vatError,
                    onValidate: _onValidateVat,
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: 12),
          _Section(
            title: 'Adresse',
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _LabeledField(
                  label: 'Adresse',
                  controller: _addressLine1,
                  validator: _required,
                ),
                const SizedBox(height: 12),
                _LabeledField(
                  label: "Complément d'adresse (optionnel)",
                  controller: _addressLine2,
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: _LabeledField(
                        label: 'Code postal',
                        controller: _postalCode,
                        validator: _required,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _LabeledField(
                        label: 'Ville',
                        controller: _city,
                        validator: _required,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                _CountryDropdown(
                  value: _country,
                  onChanged: (v) =>
                      setState(() => _country = v ?? ''),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          _Section(
            title: 'Email de facturation',
            child: _LabeledField(
              label: 'Email où envoyer les factures',
              controller: _invoicingEmail,
              keyboardType: TextInputType.emailAddress,
              validator: _email,
            ),
          ),
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
                style: const TextStyle(
                  color: Color(0xFF22C55E),
                  fontSize: 13,
                ),
              ),
            ),
          ElevatedButton(
            onPressed: _saving ? null : _onSave,
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFFF43F5E),
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

  // ---------------------------------------------------------------------------
  // Validators
  // ---------------------------------------------------------------------------

  String? _required(String? v) {
    if (v == null || v.trim().isEmpty) return 'Champ obligatoire';
    return null;
  }

  String? _siret(String? v) {
    if (v == null || v.trim().isEmpty) return 'Champ obligatoire';
    final digits = v.trim();
    if (digits.length != 14 ||
        !RegExp(r'^[0-9]{14}$').hasMatch(digits)) {
      return 'Le SIRET doit comporter 14 chiffres';
    }
    return null;
  }

  String? _email(String? v) {
    if (v == null || v.trim().isEmpty) return 'Champ obligatoire';
    final value = v.trim();
    if (!RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$').hasMatch(value)) {
      return 'Email invalide';
    }
    return null;
  }

  // ---------------------------------------------------------------------------
  // Actions
  // ---------------------------------------------------------------------------

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
      await useCase(
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
          invoicingEmail: _invoicingEmail.text.trim(),
        ),
      );
      ref.invalidate(billingProfileProvider);
      if (!mounted) return;
      setState(() {
        _saveSuccess = 'Profil enregistré.';
      });
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
      // Re-hydrate locally so the user sees the synced values without
      // having to reload the screen — invalidate also so other consumers
      // refresh.
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

// ---------------------------------------------------------------------------
// Section + field building blocks
// ---------------------------------------------------------------------------

class _Section extends StatelessWidget {
  const _Section({required this.title, required this.child});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            title,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _LabeledField extends StatelessWidget {
  const _LabeledField({
    required this.label,
    required this.controller,
    this.hint,
    this.validator,
    this.keyboardType,
    this.maxLength,
  });

  final String label;
  final TextEditingController controller;
  final String? hint;
  final String? Function(String?)? validator;
  final TextInputType? keyboardType;
  final int? maxLength;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: theme.textTheme.bodySmall?.copyWith(
            fontWeight: FontWeight.w500,
          ),
        ),
        const SizedBox(height: 6),
        TextFormField(
          controller: controller,
          keyboardType: keyboardType,
          maxLength: maxLength,
          validator: validator,
          decoration: InputDecoration(
            isDense: true,
            counterText: '',
            hintText: hint,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
          ),
        ),
      ],
    );
  }
}

class _ProfileTypeRadio extends StatelessWidget {
  const _ProfileTypeRadio({required this.value, required this.onChanged});

  final ProfileType? value;
  final ValueChanged<ProfileType> onChanged;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        _Tile(
          label: 'Particulier',
          selected: value == ProfileType.individual,
          onTap: () => onChanged(ProfileType.individual),
        ),
        const SizedBox(height: 8),
        _Tile(
          label: 'Entreprise',
          selected: value == ProfileType.business,
          onTap: () => onChanged(ProfileType.business),
        ),
      ],
    );
  }
}

class _Tile extends StatelessWidget {
  const _Tile({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: selected
              ? const Color(0xFFFFE4E6)
              : theme.colorScheme.surface,
          border: Border.all(
            color: selected
                ? const Color(0xFFF43F5E)
                : theme.dividerColor,
          ),
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        ),
        child: Row(
          children: [
            Icon(
              selected
                  ? Icons.radio_button_checked
                  : Icons.radio_button_unchecked,
              size: 18,
              color: selected
                  ? const Color(0xFFF43F5E)
                  : theme.colorScheme.onSurface.withValues(alpha: 0.5),
            ),
            const SizedBox(width: 10),
            Text(
              label,
              style: TextStyle(
                fontSize: 14,
                fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                color: selected
                    ? const Color(0xFFBE123C)
                    : theme.colorScheme.onSurface,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _CountryDropdown extends StatelessWidget {
  const _CountryDropdown({required this.value, required this.onChanged});

  final String value;
  final ValueChanged<String?> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final entries = _kEuCountries
        .map(
          (c) => DropdownMenuItem<String>(
            value: c.code,
            child: Text(c.label),
          ),
        )
        .toList();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Pays',
          style: theme.textTheme.bodySmall?.copyWith(
            fontWeight: FontWeight.w500,
          ),
        ),
        const SizedBox(height: 6),
        DropdownButtonFormField<String>(
          initialValue: value.isEmpty ? null : value,
          isExpanded: true,
          decoration: InputDecoration(
            isDense: true,
            hintText: '— Sélectionne —',
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
          ),
          items: entries,
          onChanged: onChanged,
          validator: (v) =>
              v == null || v.isEmpty ? 'Champ obligatoire' : null,
        ),
      ],
    );
  }
}

class _VatRow extends StatelessWidget {
  const _VatRow({
    required this.controller,
    required this.validatedAt,
    required this.registeredName,
    required this.validating,
    required this.error,
    required this.onValidate,
  });

  final TextEditingController controller;
  final DateTime? validatedAt;
  final String? registeredName;
  final bool validating;
  final String? error;
  final VoidCallback onValidate;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _LabeledField(
          label: 'Numéro de TVA intracommunautaire',
          controller: controller,
          hint: 'FR12345678901',
        ),
        const SizedBox(height: 8),
        Align(
          alignment: Alignment.centerLeft,
          child: OutlinedButton.icon(
            onPressed: validating || controller.text.trim().isEmpty
                ? null
                : onValidate,
            icon: validating
                ? const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.check_circle_outline, size: 16),
            label: const Text('Valider mon n° TVA'),
          ),
        ),
        if (validatedAt != null && error == null) ...[
          const SizedBox(height: 6),
          Row(
            children: [
              const Icon(
                Icons.check_circle,
                size: 14,
                color: Color(0xFF22C55E),
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  registeredName != null && registeredName!.isNotEmpty
                      ? '$registeredName · validé le '
                          '${DateFormat('dd/MM/yyyy').format(validatedAt!)}'
                      : 'Validé le ${DateFormat('dd/MM/yyyy').format(validatedAt!)}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: const Color(0xFF15803D),
                  ),
                ),
              ),
            ],
          ),
        ],
        if (error != null) ...[
          const SizedBox(height: 6),
          Row(
            children: [
              Icon(
                Icons.cancel,
                size: 14,
                color: theme.colorScheme.error,
              ),
              const SizedBox(width: 6),
              Text(
                error!,
                style: TextStyle(
                  color: theme.colorScheme.error,
                  fontSize: 12,
                ),
              ),
            ],
          ),
        ],
      ],
    );
  }
}

class _MissingBanner extends StatelessWidget {
  const _MissingBanner({required this.fields});

  final List<MissingField> fields;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: const Color(0xFFFFF7ED),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: const Color(0xFFFCD34D),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(
            Icons.warning_amber_rounded,
            size: 18,
            color: Color(0xFF92400E),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Quelques informations restent à compléter',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                    color: const Color(0xFF92400E),
                  ),
                ),
                const SizedBox(height: 4),
                ...fields.map(
                  (f) => Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Text(
                      '• ${describeMissing(f)}',
                      style: TextStyle(
                        fontSize: 12,
                        color: const Color(0xFF92400E).withValues(alpha: 0.9),
                      ),
                    ),
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

class _StripeSyncRow extends StatelessWidget {
  const _StripeSyncRow({
    required this.syncedAt,
    required this.syncing,
    required this.onSync,
    required this.error,
  });

  final DateTime? syncedAt;
  final bool syncing;
  final VoidCallback onSync;
  final String? error;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: syncedAt == null
                  ? Text(
                      'Profil non synchronisé depuis Stripe',
                      style: theme.textTheme.bodySmall,
                    )
                  : Row(
                      children: [
                        const Icon(
                          Icons.check_circle,
                          size: 14,
                          color: Color(0xFF22C55E),
                        ),
                        const SizedBox(width: 6),
                        Expanded(
                          child: Text(
                            'Synchronisé le '
                            '${DateFormat('dd/MM/yyyy').format(syncedAt!)}',
                            style: theme.textTheme.bodySmall,
                          ),
                        ),
                      ],
                    ),
            ),
            if (syncedAt == null)
              OutlinedButton.icon(
                onPressed: syncing ? null : onSync,
                icon: syncing
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.sync, size: 16),
                label: const Text('Sync depuis Stripe'),
              ),
          ],
        ),
        if (error != null) ...[
          const SizedBox(height: 6),
          Text(
            error!,
            style: TextStyle(
              color: theme.colorScheme.error,
              fontSize: 12,
            ),
          ),
        ],
      ],
    );
  }
}

class _Loader extends StatelessWidget {
  const _Loader();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 48),
      child: Center(child: CircularProgressIndicator()),
    );
  }
}

class _LoadError extends StatelessWidget {
  const _LoadError();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Text(
        'Impossible de charger le profil de facturation. Réessaie dans '
        'un instant.',
        style: theme.textTheme.bodyMedium,
      ),
    );
  }
}
