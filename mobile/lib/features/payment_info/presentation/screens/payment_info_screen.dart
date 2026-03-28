import 'dart:developer' show log;

import 'package:flutter/material.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/payment_info.dart';
import '../widgets/payment_info_widgets.dart';

// ---------------------------------------------------------------------------
// Screen
// ---------------------------------------------------------------------------

/// Payment information form — UI only, no backend call.
///
/// Accessible to agency and provider roles. Displays personal info,
/// optional business info, and bank account fields.
class PaymentInfoScreen extends StatefulWidget {
  const PaymentInfoScreen({super.key});

  @override
  State<PaymentInfoScreen> createState() => _PaymentInfoScreenState();
}

class _PaymentInfoScreenState extends State<PaymentInfoScreen> {
  var _data = const PaymentInfoFormData();
  bool _saved = false;

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
        _data.email.trim().isNotEmpty &&
        _data.country.isNotEmpty &&
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
          _data.taxId.trim().isNotEmpty;
      if (!bizOk) return false;
    }

    final bankOk = _data.accountHolder.trim().isNotEmpty &&
        (_data.bankMode == BankAccountMode.iban
            ? _data.iban.trim().isNotEmpty
            : _data.accountNumber.trim().isNotEmpty &&
                _data.routingNumber.trim().isNotEmpty);
    return bankOk;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.paymentInfoTitle),
      ),
      body: SafeArea(
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

              // Status banner
              PaymentStatusBanner(saved: _saved),
              const SizedBox(height: 16),

              // Business toggle
              PaymentBusinessToggle(
                value: _data.isBusiness,
                onChanged: (v) => _update((d) => d.copyWith(isBusiness: v)),
              ),
              const SizedBox(height: 16),

              // Personal info
              _PersonalInfoSection(data: _data, onUpdate: _update),
              const SizedBox(height: 16),

              // Business info (animated)
              AnimatedSize(
                duration: const Duration(milliseconds: 300),
                curve: Curves.easeOut,
                alignment: Alignment.topCenter,
                child: _data.isBusiness
                    ? Padding(
                        padding: const EdgeInsets.only(bottom: 16),
                        child: _BusinessInfoSection(
                          data: _data,
                          onUpdate: _update,
                        ),
                      )
                    : const SizedBox.shrink(),
              ),

              // Bank account
              _BankAccountSection(data: _data, onUpdate: _update),
              const SizedBox(height: 24),

              // Save button
              SizedBox(
                width: double.infinity,
                height: 48,
                child: ElevatedButton(
                  onPressed: _isValid
                      ? () {
                          log('Payment info: $_data');
                          setState(() => _saved = true);
                        }
                      : null,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: const Color(0xFFF43F5E),
                    foregroundColor: Colors.white,
                    disabledBackgroundColor:
                        theme.colorScheme.onSurface.withValues(alpha: 0.12),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(AppTheme.radiusLg),
                    ),
                  ),
                  child: Text(
                    l10n.paymentInfoSave,
                    style: const TextStyle(
                      fontWeight: FontWeight.w600,
                      fontSize: 15,
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 24),
            ],
          ),
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
        PaymentFormField(
          label: l10n.paymentInfoEmail,
          value: data.email,
          onChanged: (v) => onUpdate((d) => d.copyWith(email: v)),
          keyboardType: TextInputType.emailAddress,
          required: true,
        ),
        PaymentCountryDropdown(
          value: data.country,
          onChanged: (code) {
            final mode = ibanCountryCodes.contains(code)
                ? BankAccountMode.iban
                : BankAccountMode.local;
            onUpdate((d) => d.copyWith(country: code, bankMode: mode));
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
          PaymentBankModeToggle(
            label: l10n.paymentInfoNoIban,
            onTap: () =>
                onUpdate((d) => d.copyWith(bankMode: BankAccountMode.local)),
          ),
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
          PaymentBankModeToggle(
            label: l10n.paymentInfoUseIban,
            onTap: () =>
                onUpdate((d) => d.copyWith(bankMode: BankAccountMode.iban)),
          ),
        ],
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
