import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referral/domain/entities/referral_entity.dart';
import 'package:marketplace_mobile/features/referral/presentation/providers/referral_provider.dart';

/// CreateReferralResult is the discriminated outcome returned by the
/// mobile createReferral() helper. It surfaces the backend's
/// "already_in_relation" anti-fraud rejection as a typed flag the
/// screen can switch on without inspecting raw exceptions.
void main() {
  Referral fixture() => Referral.fromJson(<String, dynamic>{
        'id': 'r-1',
        'referrer_id': 'u-1',
        'provider_id': 'u-2',
        'client_id': 'u-3',
        'rate_pct': 10.0,
        'duration_months': 6,
        'status': 'pending_provider',
        'version': 1,
        'intro_snapshot': {
          'provider': <String, dynamic>{},
          'client': <String, dynamic>{},
        },
        'last_action_at': '2026-04-01T00:00:00Z',
        'created_at': '2026-04-01T00:00:00Z',
        'updated_at': '2026-04-01T00:00:00Z',
      });

  group('CreateReferralResult', () {
    test('success carries the referral and reads as success', () {
      final outcome = CreateReferralResult.success(fixture());
      expect(outcome.isSuccess, isTrue);
      expect(outcome.isAlreadyInRelation, isFalse);
      expect(outcome.referral?.id, 'r-1');
      expect(outcome.errorCode, isNull);
    });

    test('failure with already_in_relation flags the anti-fraud branch', () {
      final outcome = CreateReferralResult.failure(
        code: referralAlreadyInRelationCode,
      );
      expect(outcome.isSuccess, isFalse);
      expect(outcome.isAlreadyInRelation, isTrue);
      expect(outcome.errorCode, referralAlreadyInRelationCode);
    });

    test('failure with another code does NOT flag the anti-fraud branch', () {
      final outcome = CreateReferralResult.failure(code: 'validation_error');
      expect(outcome.isSuccess, isFalse);
      expect(outcome.isAlreadyInRelation, isFalse);
      expect(outcome.errorCode, 'validation_error');
    });

    test('failure with no code does not crash the discriminator', () {
      final outcome = CreateReferralResult.failure();
      expect(outcome.isSuccess, isFalse);
      expect(outcome.isAlreadyInRelation, isFalse);
      expect(outcome.errorCode, isNull);
    });

    test('referralAlreadyInRelationCode matches the backend sentinel', () {
      // The backend's handler maps referral.ErrPartiesAlreadyInRelation to
      // exactly this code (handler/referral_handler.go). Pinning the
      // constant prevents an accidental rename from breaking the gate.
      expect(referralAlreadyInRelationCode, 'already_in_relation');
    });
  });
}
