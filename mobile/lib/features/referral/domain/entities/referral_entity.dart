/// Referral domain entity for the apport d'affaires (business referral)
/// feature. Mirrors the backend `ReferralResponse` shape (snake_case JSON).
///
/// IMPORTANT: ratePct is intentionally nullable. The backend redacts it
/// when the viewer is the client and the referral is in a pre-active
/// state (Modèle A: the client never sees the commission rate). Widgets
/// must handle the null case gracefully.
///
/// Plain Dart class — build_runner is broken in this repo so Freezed
/// is not used (same convention as DisputeEntity).
class Referral {
  const Referral({
    required this.id,
    required this.referrerId,
    required this.providerId,
    required this.clientId,
    required this.durationMonths,
    required this.status,
    required this.version,
    required this.introSnapshot,
    required this.lastActionAt,
    required this.createdAt,
    required this.updatedAt,
    this.ratePct,
    this.introMessageForMe,
    this.activatedAt,
    this.expiresAt,
    this.rejectionReason,
  });

  final String id;
  final String referrerId;
  final String providerId;
  final String clientId;

  /// Commission percentage. NULL when the viewer is the client and the
  /// referral has not been activated yet (Modèle A confidentiality).
  final double? ratePct;

  final int durationMonths;

  /// One of: pending_provider, pending_referrer, pending_client, active,
  /// rejected, expired, cancelled, terminated.
  final String status;

  final int version;
  final IntroSnapshot introSnapshot;
  final String? introMessageForMe;
  final String? activatedAt;
  final String? expiresAt;
  final String lastActionAt;
  final String? rejectionReason;
  final String createdAt;
  final String updatedAt;

  bool get isPending =>
      status == 'pending_provider' ||
      status == 'pending_referrer' ||
      status == 'pending_client';

  bool get isTerminal =>
      status == 'rejected' ||
      status == 'expired' ||
      status == 'cancelled' ||
      status == 'terminated';

  factory Referral.fromJson(Map<String, dynamic> json) {
    return Referral(
      id: json['id'] as String,
      referrerId: json['referrer_id'] as String,
      providerId: json['provider_id'] as String,
      clientId: json['client_id'] as String,
      ratePct: (json['rate_pct'] as num?)?.toDouble(),
      durationMonths: json['duration_months'] as int,
      status: json['status'] as String,
      version: json['version'] as int,
      introSnapshot: IntroSnapshot.fromJson(
        (json['intro_snapshot'] as Map<String, dynamic>?) ?? const {},
      ),
      introMessageForMe: json['intro_message_for_me'] as String?,
      activatedAt: json['activated_at'] as String?,
      expiresAt: json['expires_at'] as String?,
      lastActionAt: json['last_action_at'] as String,
      rejectionReason: json['rejection_reason'] as String?,
      createdAt: json['created_at'] as String,
      updatedAt: json['updated_at'] as String,
    );
  }
}

class IntroSnapshot {
  const IntroSnapshot({
    required this.provider,
    required this.client,
  });

  final ProviderSnapshot provider;
  final ClientSnapshot client;

  factory IntroSnapshot.fromJson(Map<String, dynamic> json) {
    return IntroSnapshot(
      provider: ProviderSnapshot.fromJson(
        (json['provider'] as Map<String, dynamic>?) ?? const {},
      ),
      client: ClientSnapshot.fromJson(
        (json['client'] as Map<String, dynamic>?) ?? const {},
      ),
    );
  }
}

class ProviderSnapshot {
  const ProviderSnapshot({
    this.expertiseDomains = const [],
    this.yearsExperience,
    this.averageRating,
    this.reviewCount,
    this.pricingMinCents,
    this.pricingMaxCents,
    this.pricingCurrency,
    this.pricingType,
    this.region,
    this.languages = const [],
    this.availabilityState,
  });

  final List<String> expertiseDomains;
  final int? yearsExperience;
  final double? averageRating;
  final int? reviewCount;
  final int? pricingMinCents;
  final int? pricingMaxCents;
  final String? pricingCurrency;
  final String? pricingType;
  final String? region;
  final List<String> languages;
  final String? availabilityState;

  bool get isEmpty =>
      expertiseDomains.isEmpty &&
      yearsExperience == null &&
      averageRating == null &&
      pricingMinCents == null &&
      region == null &&
      languages.isEmpty &&
      availabilityState == null;

  factory ProviderSnapshot.fromJson(Map<String, dynamic> json) {
    return ProviderSnapshot(
      expertiseDomains: ((json['expertise_domains'] as List<dynamic>?) ?? const [])
          .map((e) => e as String)
          .toList(),
      yearsExperience: json['years_experience'] as int?,
      averageRating: (json['average_rating'] as num?)?.toDouble(),
      reviewCount: json['review_count'] as int?,
      pricingMinCents: json['pricing_min_cents'] as int?,
      pricingMaxCents: json['pricing_max_cents'] as int?,
      pricingCurrency: json['pricing_currency'] as String?,
      pricingType: json['pricing_type'] as String?,
      region: json['region'] as String?,
      languages: ((json['languages'] as List<dynamic>?) ?? const [])
          .map((e) => e as String)
          .toList(),
      availabilityState: json['availability_state'] as String?,
    );
  }
}

class ClientSnapshot {
  const ClientSnapshot({
    this.industry,
    this.sizeBucket,
    this.region,
    this.budgetEstimateMinCents,
    this.budgetEstimateMaxCents,
    this.budgetCurrency,
    this.needSummary,
    this.timeline,
  });

  final String? industry;
  final String? sizeBucket;
  final String? region;
  final int? budgetEstimateMinCents;
  final int? budgetEstimateMaxCents;
  final String? budgetCurrency;
  final String? needSummary;
  final String? timeline;

  bool get isEmpty =>
      industry == null &&
      sizeBucket == null &&
      region == null &&
      budgetEstimateMinCents == null &&
      needSummary == null &&
      timeline == null;

  factory ClientSnapshot.fromJson(Map<String, dynamic> json) {
    return ClientSnapshot(
      industry: json['industry'] as String?,
      sizeBucket: json['size_bucket'] as String?,
      region: json['region'] as String?,
      budgetEstimateMinCents: json['budget_estimate_min_cents'] as int?,
      budgetEstimateMaxCents: json['budget_estimate_max_cents'] as int?,
      budgetCurrency: json['budget_currency'] as String?,
      needSummary: json['need_summary'] as String?,
      timeline: json['timeline'] as String?,
    );
  }
}

/// One row in the bilateral negotiation audit trail. Used by the detail
/// page timeline widget.
class ReferralNegotiation {
  const ReferralNegotiation({
    required this.id,
    required this.version,
    required this.actorId,
    required this.actorRole,
    required this.action,
    required this.ratePct,
    required this.message,
    required this.createdAt,
  });

  final String id;
  final int version;
  final String actorId;

  /// One of: referrer, provider, client.
  final String actorRole;

  /// One of: proposed, countered, accepted, rejected.
  final String action;

  final double ratePct;
  final String message;
  final String createdAt;

  factory ReferralNegotiation.fromJson(Map<String, dynamic> json) {
    return ReferralNegotiation(
      id: json['id'] as String,
      version: json['version'] as int,
      actorId: json['actor_id'] as String,
      actorRole: json['actor_role'] as String,
      action: json['action'] as String,
      ratePct: (json['rate_pct'] as num).toDouble(),
      message: (json['message'] as String?) ?? '',
      createdAt: json['created_at'] as String,
    );
  }
}
