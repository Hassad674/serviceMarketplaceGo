import 'package:flutter/material.dart';

import '../../../../../../l10n/app_localizations.dart';
import '../../../../domain/entities/message_entity.dart';

/// Card rendered for `evaluation_request` system messages.
///
/// Provides a "Leave a review" CTA when both the client and provider
/// organization ids are present in the metadata. Pre-fix legacy
/// messages without org ids leave the CTA disabled.
class EvaluationRequestBubble extends StatelessWidget {
  const EvaluationRequestBubble({
    super.key,
    required this.message,
    this.onReview,
  });

  final MessageEntity message;
  final void Function(
    String proposalId,
    String proposalTitle,
    String clientOrganizationId,
    String providerOrganizationId,
  )? onReview;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    const color = Color(0xFF10B981); // emerald-500

    final meta = message.metadata;
    final proposalId = meta?['proposal_id'] as String? ?? '';
    final proposalTitle = meta?['proposal_title'] as String? ?? '';
    final clientOrgId =
        meta?['proposal_client_organization_id'] as String? ?? '';
    final providerOrgId =
        meta?['proposal_provider_organization_id'] as String? ?? '';
    final ctaEnabled = clientOrgId.isNotEmpty && providerOrgId.isNotEmpty;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(Icons.star_outline, size: 16, color: color),
                  const SizedBox(width: 6),
                  Flexible(
                    child: Text(
                      l10n.evaluationRequestMessage,
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                        color: color,
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              SizedBox(
                height: 32,
                child: FilledButton(
                  onPressed: (onReview != null && ctaEnabled)
                      ? () => onReview!(
                            proposalId,
                            proposalTitle,
                            clientOrgId,
                            providerOrgId,
                          )
                      : null,
                  style: FilledButton.styleFrom(
                    backgroundColor: const Color(0xFFF43F5E),
                    foregroundColor: Colors.white,
                    textStyle: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  child: Text(l10n.leaveReview),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
