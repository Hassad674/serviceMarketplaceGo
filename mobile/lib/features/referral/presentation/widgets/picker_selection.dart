/// Small value classes the referral creation flow passes between the
/// picker sheets and the form. Kept here rather than on the entity
/// because they are presentation-layer artifacts that the domain layer
/// does not need to know about.
class ProviderPickerSelection {
  const ProviderPickerSelection({
    required this.userId,
    required this.orgId,
    required this.name,
    required this.orgType,
  });

  final String userId;
  final String orgId;
  final String name;
  final String orgType;
}

class ClientPickerSelection {
  const ClientPickerSelection({
    required this.userId,
    required this.orgId,
    required this.name,
  });

  final String userId;
  final String orgId;
  final String name;
}

/// Human labels for the three party org types surfaced in the pickers.
String orgTypeLabel(String orgType) {
  switch (orgType) {
    case 'provider_personal':
      return 'Freelance';
    case 'agency':
      return 'Agence';
    case 'enterprise':
      return 'Entreprise';
    default:
      return orgType;
  }
}
