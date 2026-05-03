import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../domain/entities/referral_entity.dart';
import '../providers/referral_provider.dart';
import '../../../../core/theme/app_palette.dart';

/// ReferralSystemMessageWidget renders a referral lifecycle event as an
/// interactive card inside a conversation. Role-aware accept / reject /
/// negotiate buttons appear based on the authoritative referral state
/// (fetched live so no stale action button can be tapped after another
/// device changed the status).
///
/// Metadata shape mirrors the backend payload:
///
///   - referral_id
///   - new_status (pending_provider / pending_client / active / ...)
///   - prev_status
///   - rate_pct (stripped when the client is a conv participant)
///   - referrer_id / provider_id / client_id
class ReferralSystemMessageWidget extends ConsumerWidget {
  const ReferralSystemMessageWidget({
    super.key,
    required this.type,
    required this.content,
    required this.metadata,
    required this.currentUserId,
  });

  final String type;
  final String content;
  final Map<String, dynamic> metadata;
  final String currentUserId;

  String? get _referralId => metadata['referral_id'] as String?;
  String? get _newStatus => metadata['new_status'] as String?;
  double? get _ratePct {
    final v = metadata['rate_pct'];
    if (v is num) return v.toDouble();
    return null;
  }

  String? _viewerRole() {
    if (metadata['referrer_id'] == currentUserId) return 'referrer';
    if (metadata['provider_id'] == currentUserId) return 'provider';
    if (metadata['client_id'] == currentUserId) return 'client';
    return null;
  }

  ({IconData icon, Color color, String headline}) _tone() {
    if (type == 'referral_intro_negotiated') {
      return (
        icon: Icons.swap_horiz_rounded,
        color: AppPalette.amber500,
        headline: 'Contre-proposition de taux',
      );
    }
    if (type == 'referral_intro_activated' || _newStatus == 'active') {
      return (
        icon: Icons.handshake_outlined,
        color: AppPalette.emerald500,
        headline: 'Mise en relation activée',
      );
    }
    if (type == 'referral_intro_closed') {
      if (_newStatus == 'expired') {
        return (
          icon: Icons.hourglass_bottom,
          color: AppPalette.slate500,
          headline: 'Mise en relation expirée',
        );
      }
      return (
        icon: Icons.cancel_outlined,
        color: AppPalette.slate500,
        headline: 'Mise en relation clôturée',
      );
    }
    return (
      icon: Icons.handshake_outlined,
      color: AppPalette.rose500,
      headline: "Nouvelle proposition d'apport d'affaires",
    );
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final referralId = _referralId;
    // Legacy fallback: no metadata → simple chip.
    if (referralId == null || referralId.isEmpty) {
      return _LegacyChip(content: content);
    }

    final tone = _tone();
    final liveAsync = ref.watch(referralByIdProvider(referralId));
    final viewerRole = _viewerRole();
    final rate = _ratePct;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
      child: Container(
        decoration: BoxDecoration(
          color: tone.color.withValues(alpha: 0.08),
          border: Border.all(color: tone.color.withValues(alpha: 0.24)),
          borderRadius: BorderRadius.circular(16),
        ),
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                CircleAvatar(
                  radius: 16,
                  backgroundColor: Colors.white,
                  child: Icon(tone.icon, size: 18, color: tone.color),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        tone.headline,
                        style: TextStyle(
                          fontSize: 13,
                          fontWeight: FontWeight.w700,
                          color: tone.color,
                        ),
                      ),
                      if (content.isNotEmpty)
                        Padding(
                          padding: const EdgeInsets.only(top: 2),
                          child: Text(
                            content,
                            style: TextStyle(
                              fontSize: 12,
                              color: tone.color.withValues(alpha: 0.9),
                            ),
                          ),
                        ),
                      if (rate != null)
                        Padding(
                          padding: const EdgeInsets.only(top: 6),
                          child: Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 2,
                            ),
                            decoration: BoxDecoration(
                              color: Colors.white,
                              borderRadius: BorderRadius.circular(999),
                            ),
                            child: Text(
                              'Commission ${_formatRate(rate)}',
                              style: TextStyle(
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                                color: tone.color,
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
            liveAsync.when(
              data: (live) => viewerRole == null
                  ? const SizedBox.shrink()
                  : Padding(
                      padding: const EdgeInsets.only(top: 10),
                      child: _ReferralActionsInline(
                        referral: live,
                        viewerRole: viewerRole,
                      ),
                    ),
              loading: () => const Padding(
                padding: EdgeInsets.only(top: 10),
                child: SizedBox(
                  height: 16,
                  width: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
              error: (_, __) => const SizedBox.shrink(),
            ),
            Padding(
              padding: const EdgeInsets.only(top: 10),
              child: Align(
                alignment: Alignment.centerRight,
                child: TextButton.icon(
                  onPressed: () => context.push('/referrals/$referralId'),
                  icon: const Icon(Icons.arrow_forward, size: 14),
                  label: const Text('Voir le détail'),
                  style: TextButton.styleFrom(
                    foregroundColor: tone.color,
                    textStyle: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  static String _formatRate(double rate) {
    if (rate == rate.truncate()) {
      return '${rate.toInt()}%';
    }
    return '${rate.toStringAsFixed(2)}%';
  }
}

class _LegacyChip extends StatelessWidget {
  const _LegacyChip({required this.content});
  final String content;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          decoration: BoxDecoration(
            color: AppPalette.rose500.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(20),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(
                Icons.handshake_outlined,
                size: 16,
                color: AppPalette.rose500,
              ),
              const SizedBox(width: 6),
              Flexible(
                child: Text(
                  content.isEmpty ? 'Mise en relation activée' : content,
                  style: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: AppPalette.rose500,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// _ReferralActionsInline renders role-specific accept / reject / negotiate
// buttons. Mirrors the web ReferralActions widget in behavior.
class _ReferralActionsInline extends ConsumerStatefulWidget {
  const _ReferralActionsInline({
    required this.referral,
    required this.viewerRole,
  });

  final Referral referral;
  final String viewerRole;

  @override
  ConsumerState<_ReferralActionsInline> createState() =>
      _ReferralActionsInlineState();
}

class _ReferralActionsInlineState
    extends ConsumerState<_ReferralActionsInline> {
  bool _loading = false;
  bool _showNegotiate = false;
  double _counterRate = 0;
  final TextEditingController _messageController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _counterRate = widget.referral.ratePct ?? 5;
  }

  @override
  void dispose() {
    _messageController.dispose();
    super.dispose();
  }

  List<_Action> _availableActions() {
    switch (widget.referral.status) {
      case 'pending_provider':
        if (widget.viewerRole == 'provider') {
          return const [
            _Action(kind: 'accept', label: 'Accepter', primary: true),
            _Action(kind: 'negotiate', label: 'Négocier'),
            _Action(kind: 'reject', label: 'Refuser', danger: true),
          ];
        }
        if (widget.viewerRole == 'referrer') {
          return const [
            _Action(kind: 'cancel', label: "Annuler l'intro", danger: true),
          ];
        }
        return const [];
      case 'pending_referrer':
        if (widget.viewerRole == 'referrer') {
          return const [
            _Action(
              kind: 'accept',
              label: 'Accepter le contre',
              primary: true,
            ),
            _Action(kind: 'negotiate', label: 'Contre-contrer'),
            _Action(kind: 'reject', label: 'Refuser', danger: true),
          ];
        }
        return const [];
      case 'pending_client':
        if (widget.viewerRole == 'client') {
          return const [
            _Action(
              kind: 'accept',
              label: 'Accepter la mise en relation',
              primary: true,
            ),
            _Action(kind: 'reject', label: 'Refuser', danger: true),
          ];
        }
        if (widget.viewerRole == 'referrer') {
          return const [
            _Action(kind: 'cancel', label: "Annuler l'intro", danger: true),
          ];
        }
        return const [];
      case 'active':
        if (widget.viewerRole == 'referrer') {
          return const [
            _Action(
              kind: 'terminate',
              label: "Terminer l'intro",
              danger: true,
            ),
          ];
        }
        return const [];
      default:
        return const [];
    }
  }

  Future<void> _run(String action, {double? rate, String? message}) async {
    setState(() => _loading = true);
    try {
      await respondToReferral(
        ref,
        id: widget.referral.id,
        action: action,
        newRatePct: rate,
        message: message,
      );
      if (mounted) {
        setState(() {
          _showNegotiate = false;
          _messageController.clear();
        });
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final actions = _availableActions();
    if (actions.isEmpty) return const SizedBox.shrink();

    if (_showNegotiate) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'Nouveau taux : ${_counterRate.toStringAsFixed(_counterRate % 1 == 0 ? 0 : 1)} %',
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
          Slider(
            min: 0,
            max: 30,
            divisions: 60,
            value: _counterRate.clamp(0, 30),
            onChanged: (v) => setState(() => _counterRate = v),
          ),
          TextField(
            controller: _messageController,
            decoration: const InputDecoration(
              labelText: 'Message (optionnel)',
              border: OutlineInputBorder(),
              isDense: true,
            ),
            maxLines: 2,
          ),
          const SizedBox(height: 8),
          Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              TextButton(
                onPressed: _loading
                    ? null
                    : () => setState(() => _showNegotiate = false),
                child: const Text('Annuler'),
              ),
              const SizedBox(width: 8),
              FilledButton(
                onPressed: _loading
                    ? null
                    : () => _run(
                          'negotiate',
                          rate: _counterRate,
                          message: _messageController.text,
                        ),
                child: _loading
                    ? const SizedBox(
                        height: 14,
                        width: 14,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Envoyer la contre-proposition'),
              ),
            ],
          ),
        ],
      );
    }

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: actions.map((a) {
        final onPressed = _loading
            ? null
            : () {
                if (a.kind == 'negotiate') {
                  setState(() => _showNegotiate = true);
                } else {
                  _run(a.kind);
                }
              };
        if (a.primary) {
          return FilledButton(
            onPressed: onPressed,
            style: FilledButton.styleFrom(
              backgroundColor: AppPalette.rose500,
              padding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              textStyle: const TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
            child: Text(a.label),
          );
        }
        if (a.danger) {
          return OutlinedButton(
            onPressed: onPressed,
            style: OutlinedButton.styleFrom(
              foregroundColor: AppPalette.rose600,
              side: const BorderSide(color: AppPalette.red200),
              padding:
                  const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              textStyle: const TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
            ),
            child: Text(a.label),
          );
        }
        return OutlinedButton(
          onPressed: onPressed,
          style: OutlinedButton.styleFrom(
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            textStyle: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
          child: Text(a.label),
        );
      }).toList(),
    );
  }
}

class _Action {
  const _Action({
    required this.kind,
    required this.label,
    this.primary = false,
    this.danger = false,
  });

  final String kind;
  final String label;
  final bool primary;
  final bool danger;
}
