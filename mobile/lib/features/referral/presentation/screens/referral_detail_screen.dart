import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../auth/presentation/providers/auth_provider.dart';
import '../../domain/entities/referral_entity.dart';
import '../providers/referral_provider.dart';
import '../widgets/anonymized_client_card.dart';
import '../widgets/anonymized_provider_card.dart';
import '../widgets/negotiation_timeline_widget.dart';
import '../widgets/referral_missions_section.dart';
import '../widgets/referral_status_chip.dart';

/// ReferralDetailScreen — smart container for a single referral. Resolves
/// the viewer role (referrer / provider / client) and dispatches to the
/// right rendering branch with the right action buttons.
class ReferralDetailScreen extends ConsumerWidget {
  const ReferralDetailScreen({super.key, required this.referralId});

  final String referralId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncReferral = ref.watch(referralByIdProvider(referralId));
    final authState = ref.watch(authProvider);
    final viewerId = authState.user?['id'] as String?;

    return Scaffold(
      appBar: AppBar(title: const Text('Business referral')),
      body: asyncReferral.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'Could not load this referral: $e',
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
        data: (referral) {
          final role = _resolveRole(referral, viewerId);
          if (role == null) {
            return const Padding(
              padding: EdgeInsets.all(24),
              child: Text('You are not a party to this referral.'),
            );
          }
          return _Body(referral: referral, viewerRole: role);
        },
      ),
    );
  }

  static String? _resolveRole(Referral r, String? viewerId) {
    if (viewerId == null) return null;
    if (viewerId == r.referrerId) return 'referrer';
    if (viewerId == r.providerId) return 'provider';
    if (viewerId == r.clientId) return 'client';
    return null;
  }
}

class _Body extends ConsumerWidget {
  const _Body({required this.referral, required this.viewerRole});

  final Referral referral;
  final String viewerRole;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final showRate = viewerRole != 'client' || referral.status == 'active';

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _Header(referral: referral),
        const SizedBox(height: 16),
        if (viewerRole == 'client' || viewerRole == 'referrer')
          AnonymizedProviderCard(snapshot: referral.introSnapshot.provider),
        if (viewerRole == 'client' || viewerRole == 'referrer')
          const SizedBox(height: 16),
        if (viewerRole == 'provider' || viewerRole == 'referrer')
          AnonymizedClientCard(snapshot: referral.introSnapshot.client),
        if (referral.introMessageForMe != null &&
            referral.introMessageForMe!.isNotEmpty) ...[
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(
                      Icons.format_quote,
                      size: 18,
                      color: theme.colorScheme.primary,
                    ),
                    const SizedBox(width: 8),
                    Text(
                      'A word from the referrer',
                      style: theme.textTheme.labelMedium?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Text('"${referral.introMessageForMe}"'),
              ],
            ),
          ),
        ],
        const SizedBox(height: 16),
        _ActionPanel(referral: referral, viewerRole: viewerRole),
        // Attributed proposals — visible once the intro is active.
        // Apporteur + provider see the commission amounts; the client
        // sees only proposal/milestone status (Modèle A).
        if (referral.status == 'active')
          ReferralMissionsSection(
            referralId: referral.id,
            viewerIsClient: viewerRole == 'client',
          ),
        if (showRate) ...[
          const SizedBox(height: 24),
          Text(
            'Negotiation history',
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 8),
          NegotiationTimelineWidget(
            referralId: referral.id,
            showRate: showRate,
          ),
        ],
        const SizedBox(height: 24),
      ],
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({required this.referral});

  final Referral referral;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final rateLabel = referral.ratePct == null
        ? '—'
        : '${referral.ratePct!.toStringAsFixed(referral.ratePct! % 1 == 0 ? 0 : 1)}%';
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(color: theme.colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              ReferralStatusChip(status: referral.status),
              const SizedBox(width: 8),
              Text('v${referral.version}', style: theme.textTheme.bodySmall),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              _Metric(label: 'Commission', value: rateLabel),
              const SizedBox(width: 16),
              _Metric(label: 'Duration', value: '${referral.durationMonths}mo'),
            ],
          ),
          if (referral.activatedAt != null && referral.expiresAt != null) ...[
            const SizedBox(height: 12),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: Colors.green.shade50,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                'Active until ${referral.expiresAt!.substring(0, 10)}',
                style: theme.textTheme.labelSmall?.copyWith(
                  color: Colors.green.shade800,
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _Metric extends StatelessWidget {
  const _Metric({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label.toUpperCase(),
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
            fontWeight: FontWeight.w600,
          ),
        ),
        Text(value, style: theme.textTheme.titleMedium),
      ],
    );
  }
}

/// _ActionPanel renders the action buttons available to the viewer based
/// on the current status. Tapping an action calls respondToReferral and
/// the screen rebuilds via provider invalidation.
class _ActionPanel extends ConsumerStatefulWidget {
  const _ActionPanel({required this.referral, required this.viewerRole});

  final Referral referral;
  final String viewerRole;

  @override
  ConsumerState<_ActionPanel> createState() => _ActionPanelState();
}

class _ActionPanelState extends ConsumerState<_ActionPanel> {
  bool _showNegotiate = false;
  late double _counterRate;
  String _message = '';
  bool _submitting = false;

  @override
  void initState() {
    super.initState();
    _counterRate = widget.referral.ratePct ?? 5;
  }

  Future<void> _send(String action, {double? newRate}) async {
    setState(() => _submitting = true);
    await respondToReferral(
      ref,
      id: widget.referral.id,
      action: action,
      newRatePct: newRate,
      message: _message.isNotEmpty ? _message : null,
    );
    if (mounted) {
      setState(() {
        _submitting = false;
        _showNegotiate = false;
        _message = '';
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final actions = _availableActions(widget.referral.status, widget.viewerRole);
    if (actions.isEmpty) return const SizedBox.shrink();

    final theme = Theme.of(context);

    if (_showNegotiate) {
      return Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          border: Border.all(color: theme.colorScheme.outlineVariant),
          borderRadius: BorderRadius.circular(16),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'New rate: ${_counterRate.toStringAsFixed(_counterRate % 1 == 0 ? 0 : 1)}%',
              style: theme.textTheme.bodyMedium,
            ),
            Slider(
              value: _counterRate,
              min: 0,
              max: 30,
              divisions: 60,
              onChanged: (v) => setState(() => _counterRate = v),
            ),
            TextField(
              decoration: const InputDecoration(
                labelText: 'Message (optional)',
                border: OutlineInputBorder(),
              ),
              maxLines: 2,
              onChanged: (v) => _message = v,
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => setState(() => _showNegotiate = false),
                    child: const Text('Cancel'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: _submitting
                        ? null
                        : () => _send('negotiate', newRate: _counterRate),
                    child: _submitting
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Text('Send counter'),
                  ),
                ),
              ],
            ),
          ],
        ),
      );
    }

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: actions.map((a) => _buildButton(a)).toList(),
    );
  }

  Widget _buildButton(_Action action) {
    final onPressed = _submitting
        ? null
        : () {
            switch (action.kind) {
              case 'accept':
                _send('accept');
                break;
              case 'reject':
                _send('reject');
                break;
              case 'cancel':
                _send('cancel');
                break;
              case 'terminate':
                _send('terminate');
                break;
              case 'negotiate':
                setState(() => _showNegotiate = true);
                break;
            }
          };

    if (action.variant == 'primary') {
      return FilledButton(onPressed: onPressed, child: Text(action.label));
    }
    if (action.variant == 'danger') {
      return OutlinedButton(
        onPressed: onPressed,
        style: OutlinedButton.styleFrom(foregroundColor: Theme.of(context).colorScheme.error),
        child: Text(action.label),
      );
    }
    return OutlinedButton(onPressed: onPressed, child: Text(action.label));
  }

  static List<_Action> _availableActions(String status, String role) {
    switch (status) {
      case 'pending_provider':
        if (role == 'provider') {
          return const [
            _Action('accept', 'Accept', 'primary'),
            _Action('negotiate', 'Negotiate', 'secondary'),
            _Action('reject', 'Reject', 'danger'),
          ];
        }
        if (role == 'referrer') {
          return const [_Action('cancel', 'Cancel intro', 'danger')];
        }
        return const [];

      case 'pending_referrer':
        if (role == 'referrer') {
          return const [
            _Action('accept', 'Accept counter', 'primary'),
            _Action('negotiate', 'Counter back', 'secondary'),
            _Action('reject', 'Reject', 'danger'),
          ];
        }
        return const [];

      case 'pending_client':
        if (role == 'client') {
          return const [
            _Action('accept', 'Accept introduction', 'primary'),
            _Action('reject', 'Reject', 'danger'),
          ];
        }
        if (role == 'referrer') {
          return const [_Action('cancel', 'Cancel intro', 'danger')];
        }
        return const [];

      case 'active':
        if (role == 'referrer') {
          return const [_Action('terminate', 'End referral', 'danger')];
        }
        return const [];

      default:
        return const [];
    }
  }
}

class _Action {
  const _Action(this.kind, this.label, this.variant);

  final String kind;
  final String label;
  final String variant; // primary | secondary | danger
}
