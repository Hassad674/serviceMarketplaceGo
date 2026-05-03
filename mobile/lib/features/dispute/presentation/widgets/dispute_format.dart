import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../../../l10n/app_localizations.dart';
import '../../../../core/theme/app_palette.dart';

/// Pure formatting helpers for the dispute banner. Extracted so a
/// new dispute reason or status string requires only a single edit
/// without touching the banner widget itself.

Color disputeStatusColor(String status) {
  switch (status) {
    case 'open':
    case 'negotiation':
      return AppPalette.orange600; // orange-600
    case 'escalated':
      return AppPalette.red600; // red-600
    case 'resolved':
      return AppPalette.green600; // green-600
    case 'cancelled':
      return AppPalette.slate500; // slate-500
    default:
      return AppPalette.orange600;
  }
}

IconData disputeStatusIcon(String status) {
  switch (status) {
    case 'resolved':
      return Icons.check_circle_outline;
    case 'cancelled':
      return Icons.cancel_outlined;
    case 'escalated':
      return Icons.shield_outlined;
    default:
      return Icons.warning_amber_rounded;
  }
}

String disputeStatusLabel(AppLocalizations l10n, String status) {
  switch (status) {
    case 'open':
      return l10n.disputeStatusOpen;
    case 'negotiation':
      return l10n.disputeStatusNegotiation;
    case 'escalated':
      return l10n.disputeStatusEscalated;
    case 'resolved':
      return l10n.disputeStatusResolved;
    case 'cancelled':
      return l10n.disputeStatusCancelled;
    default:
      return status;
  }
}

String disputeReasonLabel(AppLocalizations l10n, String reason) {
  switch (reason) {
    case 'work_not_conforming':
      return l10n.disputeReasonWorkNotConforming;
    case 'non_delivery':
      return l10n.disputeReasonNonDelivery;
    case 'insufficient_quality':
      return l10n.disputeReasonInsufficientQuality;
    case 'client_ghosting':
      return l10n.disputeReasonClientGhosting;
    case 'scope_creep':
      return l10n.disputeReasonScopeCreep;
    case 'refusal_to_validate':
      return l10n.disputeReasonRefusalToValidate;
    case 'harassment':
      return l10n.disputeReasonHarassment;
    default:
      return l10n.disputeReasonOther;
  }
}

String formatEur(int centimes) {
  return NumberFormat.currency(
    locale: 'fr_FR',
    symbol: '€',
    decimalDigits: 2,
  ).format(centimes / 100);
}

int daysSinceCreation(String createdAt) {
  try {
    final dt = DateTime.parse(createdAt);
    return DateTime.now().difference(dt).inDays;
  } catch (_) {
    return 0;
  }
}
