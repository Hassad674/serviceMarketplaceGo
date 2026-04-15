import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../providers/referral_provider.dart';

/// ReferralCreationScreen — single-page form to create a new business
/// referral. Mirrors the web wizard but flat (no multi-step) since the
/// mobile keyboard makes pagination painful.
///
/// V1 takes provider/client UUIDs as raw text inputs. A picker integrated
/// with the search feature is on the V2 backlog.
class ReferralCreationScreen extends ConsumerStatefulWidget {
  const ReferralCreationScreen({super.key});

  @override
  ConsumerState<ReferralCreationScreen> createState() =>
      _ReferralCreationScreenState();
}

class _ReferralCreationScreenState extends ConsumerState<ReferralCreationScreen> {
  final _formKey = GlobalKey<FormState>();
  final _providerCtrl = TextEditingController();
  final _clientCtrl = TextEditingController();
  final _pitchProviderCtrl = TextEditingController();
  final _pitchClientCtrl = TextEditingController();

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
    _providerCtrl.dispose();
    _clientCtrl.dispose();
    _pitchProviderCtrl.dispose();
    _pitchClientCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _submitting = true;
      _error = null;
    });
    final created = await createReferral(
      ref,
      providerId: _providerCtrl.text.trim(),
      clientId: _clientCtrl.text.trim(),
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
        _error = 'Could not create the intro. Check the IDs and try again.';
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
              TextFormField(
                controller: _providerCtrl,
                decoration: const InputDecoration(
                  labelText: 'Provider ID',
                  hintText: 'UUID of the provider you want to recommend',
                  border: OutlineInputBorder(),
                ),
                validator: (v) => (v == null || v.trim().isEmpty)
                    ? 'Required'
                    : null,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _clientCtrl,
                decoration: const InputDecoration(
                  labelText: 'Client ID',
                  hintText: 'UUID of the enterprise or agency',
                  border: OutlineInputBorder(),
                ),
                validator: (v) => (v == null || v.trim().isEmpty)
                    ? 'Required'
                    : null,
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

const Map<String, String> _toggleLabels = {
  'include_expertise': 'Expertise domains',
  'include_experience': 'Years of experience',
  'include_rating': 'Average rating',
  'include_pricing': 'Pricing range',
  'include_region': 'Region',
  'include_languages': 'Languages',
  'include_availability': 'Availability',
};
