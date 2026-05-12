import 'package:flutter/material.dart';

import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';
import 'system_message_palette.dart';

/// Centered pill rendered for proposal/call/dispute lifecycle events.
///
/// When [showPayCta] is true and [onPay] is provided, renders a
/// "Payer maintenant" CTA underneath the pill. Used by
/// `MessageBubble` to restore the client-facing pay action on the
/// `proposal_accepted` system message — the proposal card has its own
/// Pay button, but once it scrolls out of view the client loses the
/// quick path to pay.
class SystemMessageBubble extends StatelessWidget {
  const SystemMessageBubble({
    super.key,
    required this.message,
    this.showPayCta = false,
    this.onPay,
  });

  final MessageEntity message;
  final bool showPayCta;
  final VoidCallback? onPay;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    final visuals = systemMessageVisualsFor(
      context: context,
      message: message,
      l10n: l10n,
    );

    final pill = Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      decoration: BoxDecoration(
        color: visuals.color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(visuals.icon, size: 16, color: visuals.color),
          const SizedBox(width: 6),
          Flexible(
            child: Text(
              visuals.label,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
                color: visuals.color,
              ),
            ),
          ),
        ],
      ),
    );

    final shouldRenderCta = showPayCta && onPay != null;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: shouldRenderCta
            ? Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  pill,
                  const SizedBox(height: 8),
                  FilledButton.icon(
                    key: const ValueKey(
                      'proposal-accepted-pay-cta',
                    ),
                    onPressed: onPay,
                    icon: const Icon(Icons.payment_outlined, size: 16),
                    label: Text(l10n.payNow),
                  ),
                ],
              )
            : pill,
      ),
    );
  }
}
