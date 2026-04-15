import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../providers/referral_provider.dart';
import '../widgets/client_picker_sheet.dart';
import '../widgets/picker_selection.dart';
import '../widgets/provider_picker_sheet.dart';

/// ReferralCreationScreen — single-page form to create a new business
/// referral. Provider and client parties are picked via dedicated modal
/// bottom sheets (searchable for providers, conversation-restricted for
/// clients) — the apporteur never types a raw UUID.
class ReferralCreationScreen extends ConsumerStatefulWidget {
  const ReferralCreationScreen({super.key});

  @override
  ConsumerState<ReferralCreationScreen> createState() =>
      _ReferralCreationScreenState();
}

class _ReferralCreationScreenState extends ConsumerState<ReferralCreationScreen> {
  final _formKey = GlobalKey<FormState>();
  final _pitchProviderCtrl = TextEditingController();
  final _pitchClientCtrl = TextEditingController();

  ProviderPickerSelection? _provider;
  ClientPickerSelection? _client;
  double _ratePct = 5;
  int _durationMonths = 6;
  bool _submitting = false;
  String? _error;

  // Snapshot toggles — V1 reveals everything by default. The user can
  // tweak in a follow-up if they want to mask specific fields.
  final Map<String, bool> _toggles = {
    'include_expertise': true,
    'include_experience': true,
    'include_rating': true,
    'include_pricing': true,
    'include_region': true,
    'include_languages': true,
    'include_availability': true,
  };

  @override
  void dispose() {
    _pitchProviderCtrl.dispose();
    _pitchClientCtrl.dispose();
    super.dispose();
  }

  Future<void> _pickProvider() async {
    final selection = await showProviderPickerSheet(context);
    if (!mounted || selection == null) return;
    setState(() => _provider = selection);
  }

  Future<void> _pickClient() async {
    final selection = await showClientPickerSheet(context);
    if (!mounted || selection == null) return;
    setState(() => _client = selection);
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    if (_provider == null) {
      setState(() => _error = 'Sélectionnez un prestataire.');
      return;
    }
    if (_client == null) {
      setState(() => _error = 'Sélectionnez un client parmi vos conversations.');
      return;
    }
    setState(() {
      _submitting = true;
      _error = null;
    });
    final created = await createReferral(
      ref,
      providerId: _provider!.userId,
      clientId: _client!.userId,
      ratePct: _ratePct,
      durationMonths: _durationMonths,
      introMessageProvider: _pitchProviderCtrl.text.trim(),
      introMessageClient: _pitchClientCtrl.text.trim(),
      snapshotToggles: _toggles,
    );
    if (!mounted) return;
    if (created == null) {
      setState(() {
        _submitting = false;
        _error = 'Could not create the intro. Please try again.';
      });
      return;
    }
    context.go('/referrals/${created.id}');
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: const Text('New referral')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'Introduce a provider to a client. The client never sees the commission rate — you negotiate it privately with the provider.',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 24),
              _SectionTitle('1 — Parties'),
              _PartyTile(
                label: 'Prestataire',
                placeholder: 'Rechercher ou choisir depuis une conversation…',
                selectedName: _provider?.name,
                selectedBadge: _provider != null ? orgTypeLabel(_provider!.orgType) : null,
                leadingIcon: Icons.person_outline,
                onTap: _pickProvider,
                onClear: _provider != null ? () => setState(() => _provider = null) : null,
              ),
              const SizedBox(height: 12),
              _PartyTile(
                label: 'Client',
                placeholder: 'Choisir depuis une conversation…',
                selectedName: _client?.name,
                selectedBadge: _client != null ? 'Entreprise' : null,
                leadingIcon: Icons.business_outlined,
                onTap: _pickClient,
                onClear: _client != null ? () => setState(() => _client = null) : null,
              ),
              const SizedBox(height: 24),
              _SectionTitle('2 — Terms'),
              Text(
                'Commission: ${_ratePct.toStringAsFixed(_ratePct % 1 == 0 ? 0 : 1)}%',
                style: theme.textTheme.bodyMedium,
              ),
              Slider(
                value: _ratePct,
                min: 0,
                max: 30,
                divisions: 60,
                label: '${_ratePct.toStringAsFixed(1)}%',
                onChanged: (v) => setState(() => _ratePct = v),
              ),
              const SizedBox(height: 8),
              DropdownButtonFormField<int>(
                initialValue: _durationMonths,
                decoration: const InputDecoration(
                  labelText: 'Exclusivity duration',
                  border: OutlineInputBorder(),
                ),
                items: const [3, 6, 9, 12, 18, 24]
                    .map((n) => DropdownMenuItem(value: n, child: Text('$n months')))
                    .toList(),
                onChanged: (v) => setState(() => _durationMonths = v ?? 6),
              ),
              const SizedBox(height: 24),
              _SectionTitle('3 — Snapshot fields'),
              Text(
                'Pick the provider attributes the client will see before accepting.',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              for (final entry in _toggleLabels.entries)
                CheckboxListTile(
                  contentPadding: EdgeInsets.zero,
                  dense: true,
                  title: Text(entry.value),
                  value: _toggles[entry.key] ?? false,
                  onChanged: (v) => setState(() => _toggles[entry.key] = v ?? false),
                ),
              const SizedBox(height: 24),
              _SectionTitle('4 — Your messages'),
              TextFormField(
                controller: _pitchProviderCtrl,
                maxLines: 3,
                maxLength: 2000,
                decoration: const InputDecoration(
                  labelText: 'Pitch for the provider',
                  border: OutlineInputBorder(),
                ),
                validator: (v) => (v == null || v.trim().isEmpty)
                    ? 'Required'
                    : null,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _pitchClientCtrl,
                maxLines: 3,
                maxLength: 2000,
                decoration: const InputDecoration(
                  labelText: 'Pitch for the client',
                  border: OutlineInputBorder(),
                ),
                validator: (v) => (v == null || v.trim().isEmpty)
                    ? 'Required'
                    : null,
              ),
              if (_error != null) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.errorContainer,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    _error!,
                    style: TextStyle(color: theme.colorScheme.onErrorContainer),
                  ),
                ),
              ],
              const SizedBox(height: 24),
              FilledButton.icon(
                onPressed: _submitting ? null : _submit,
                icon: _submitting
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.send),
                label: const Text('Send introduction'),
              ),
              const SizedBox(height: 32),
            ],
          ),
        ),
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle(this.title);
  final String title;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Text(
        title,
        style: Theme.of(context).textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w700,
            ),
      ),
    );
  }
}

/// _PartyTile renders a Material input-shaped trigger that opens one of
/// the picker bottom sheets on tap. Keeps the form visually aligned with
/// the other TextFormField entries while avoiding any raw UUID input.
class _PartyTile extends StatelessWidget {
  const _PartyTile({
    required this.label,
    required this.placeholder,
    required this.selectedName,
    required this.selectedBadge,
    required this.leadingIcon,
    required this.onTap,
    required this.onClear,
  });

  final String label;
  final String placeholder;
  final String? selectedName;
  final String? selectedBadge;
  final IconData leadingIcon;
  final VoidCallback onTap;
  final VoidCallback? onClear;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasValue = selectedName != null && selectedName!.isNotEmpty;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: InputDecorator(
        decoration: InputDecoration(
          labelText: label,
          border: const OutlineInputBorder(),
          prefixIcon: Icon(leadingIcon, size: 20),
          suffixIcon: hasValue && onClear != null
              ? IconButton(
                  icon: const Icon(Icons.close, size: 18),
                  onPressed: onClear,
                  tooltip: 'Effacer',
                )
              : const Icon(Icons.chevron_right),
        ),
        child: hasValue
            ? Row(
                children: [
                  Expanded(
                    child: Text(
                      selectedName!,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontWeight: FontWeight.w600),
                    ),
                  ),
                  if (selectedBadge != null) ...[
                    const SizedBox(width: 8),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(
                        color: theme.colorScheme.surfaceContainerHighest,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        selectedBadge!,
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                  ],
                ],
              )
            : Text(
                placeholder,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
      ),
    );
  }
}

const Map<String, String> _toggleLabels = {
  'include_expertise': 'Expertise domains',
  'include_experience': 'Years of experience',
  'include_rating': 'Average rating',
  'include_pricing': 'Pricing range',
  'include_region': 'Region',
  'include_languages': 'Languages',
  'include_availability': 'Availability',
};
