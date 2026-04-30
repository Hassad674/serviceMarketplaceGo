import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referral/domain/entities/referral_entity.dart';

void main() {
  Map<String, dynamic> baseJson() => {
        'id': 'r-1',
        'referrer_id': 'u-1',
        'provider_id': 'u-2',
        'client_id': 'u-3',
        'rate_pct': 10.0,
        'duration_months': 6,
        'status': 'pending_provider',
        'version': 1,
        'intro_snapshot': {
          'provider': {},
          'client': {},
        },
        'last_action_at': '2026-04-01T00:00:00Z',
        'created_at': '2026-04-01T00:00:00Z',
        'updated_at': '2026-04-01T00:00:00Z',
      };

  group('Referral.fromJson', () {
    test('parses required fields', () {
      final r = Referral.fromJson(baseJson());
      expect(r.id, 'r-1');
      expect(r.referrerId, 'u-1');
      expect(r.providerId, 'u-2');
      expect(r.clientId, 'u-3');
      expect(r.ratePct, 10.0);
      expect(r.durationMonths, 6);
      expect(r.status, 'pending_provider');
    });

    test('handles null ratePct (Modèle A redaction)', () {
      final json = baseJson()..remove('rate_pct');
      final r = Referral.fromJson(json);
      expect(r.ratePct, isNull);
    });

    test('parses optional fields when present', () {
      final json = baseJson()
        ..['intro_message_for_me'] = 'hi'
        ..['activated_at'] = '2026-04-15T00:00:00Z'
        ..['expires_at'] = '2026-10-15T00:00:00Z'
        ..['rejection_reason'] = 'busy';
      final r = Referral.fromJson(json);
      expect(r.introMessageForMe, 'hi');
      expect(r.activatedAt, '2026-04-15T00:00:00Z');
      expect(r.expiresAt, '2026-10-15T00:00:00Z');
      expect(r.rejectionReason, 'busy');
    });
  });

  group('Referral.isPending', () {
    test('returns true for pending_provider', () {
      final r = Referral.fromJson(baseJson());
      expect(r.isPending, isTrue);
    });

    test('returns true for pending_referrer', () {
      final json = baseJson()..['status'] = 'pending_referrer';
      expect(Referral.fromJson(json).isPending, isTrue);
    });

    test('returns true for pending_client', () {
      final json = baseJson()..['status'] = 'pending_client';
      expect(Referral.fromJson(json).isPending, isTrue);
    });

    test('returns false for active', () {
      final json = baseJson()..['status'] = 'active';
      expect(Referral.fromJson(json).isPending, isFalse);
    });
  });

  group('Referral.isTerminal', () {
    test('returns true for rejected', () {
      final json = baseJson()..['status'] = 'rejected';
      expect(Referral.fromJson(json).isTerminal, isTrue);
    });

    test('returns true for expired', () {
      final json = baseJson()..['status'] = 'expired';
      expect(Referral.fromJson(json).isTerminal, isTrue);
    });

    test('returns true for cancelled', () {
      final json = baseJson()..['status'] = 'cancelled';
      expect(Referral.fromJson(json).isTerminal, isTrue);
    });

    test('returns true for terminated', () {
      final json = baseJson()..['status'] = 'terminated';
      expect(Referral.fromJson(json).isTerminal, isTrue);
    });

    test('returns false for active', () {
      final json = baseJson()..['status'] = 'active';
      expect(Referral.fromJson(json).isTerminal, isFalse);
    });
  });

  group('IntroSnapshot', () {
    test('parses with empty payload', () {
      final s = IntroSnapshot.fromJson({});
      expect(s.provider.isEmpty, isTrue);
    });

    test('parses provider details', () {
      final s = IntroSnapshot.fromJson({
        'provider': {
          'expertise_domains': ['mobile', 'backend'],
          'years_experience': 5,
          'average_rating': 4.5,
          'review_count': 12,
          'languages': ['fr', 'en'],
          'pricing_min_cents': 50000,
          'pricing_max_cents': 200000,
          'pricing_currency': 'EUR',
          'pricing_type': 'hourly',
          'region': 'EU',
          'availability_state': 'available',
        },
      });
      expect(s.provider.expertiseDomains, ['mobile', 'backend']);
      expect(s.provider.yearsExperience, 5);
      expect(s.provider.averageRating, 4.5);
      expect(s.provider.languages, ['fr', 'en']);
      expect(s.provider.pricingMinCents, 50000);
      expect(s.provider.region, 'EU');
      expect(s.provider.isEmpty, isFalse);
    });
  });

  group('ProviderSnapshot.isEmpty', () {
    test('returns true when nothing is set', () {
      const p = ProviderSnapshot();
      expect(p.isEmpty, isTrue);
    });

    test('returns false when at least one field is set', () {
      const p = ProviderSnapshot(yearsExperience: 1);
      expect(p.isEmpty, isFalse);
    });
  });
}
