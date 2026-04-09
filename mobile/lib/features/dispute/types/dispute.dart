import 'dart:io';

/// Reasons a dispute can be opened. Values match the backend enum.
enum DisputeReason {
  workNotConforming('work_not_conforming'),
  nonDelivery('non_delivery'),
  insufficientQuality('insufficient_quality'),
  clientGhosting('client_ghosting'),
  scopeCreep('scope_creep'),
  refusalToValidate('refusal_to_validate'),
  harassment('harassment'),
  other('other');

  const DisputeReason(this.value);

  final String value;

  /// Returns the reasons available for the given role.
  /// Client can report: non-conforming work, non-delivery, insufficient quality, other.
  /// Provider can report: client ghosting, scope creep, refusal to validate, harassment, other.
  static List<DisputeReason> forRole(String role) {
    if (role == 'client') {
      return const [
        DisputeReason.workNotConforming,
        DisputeReason.nonDelivery,
        DisputeReason.insufficientQuality,
        DisputeReason.other,
      ];
    }
    return const [
      DisputeReason.clientGhosting,
      DisputeReason.scopeCreep,
      DisputeReason.refusalToValidate,
      DisputeReason.harassment,
      DisputeReason.other,
    ];
  }
}

/// Mutable form state for opening a dispute.
class DisputeFormData {
  DisputeReason? reason;
  String messageToParty = '';
  String description = '';
  AmountType amountType = AmountType.total;
  int partialAmount = 0;
  List<File> attachments = [];

  DisputeFormData();
}

/// Mutable form state for a counter-proposal.
class CounterProposalFormData {
  int amountClient = 0;
  String message = '';
  List<File> attachments = [];

  CounterProposalFormData();
}

enum AmountType { total, partial }
