import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/client_profile/domain/entities/client_profile.dart';

void main() {
  group('ClientProjectProvider', () {
    test('parses a full payload', () {
      final entity = ClientProjectProvider.fromJson({
        'organization_id': 'org-p-1',
        'display_name': 'Pixel Studio',
        'avatar_url': 'https://cdn/avatar.png',
      });

      expect(entity.organizationId, 'org-p-1');
      expect(entity.displayName, 'Pixel Studio');
      expect(entity.avatarUrl, 'https://cdn/avatar.png');
    });

    test('tolerates missing fields with safe defaults', () {
      final entity = ClientProjectProvider.fromJson(const {});
      expect(entity.organizationId, '');
      expect(entity.displayName, '');
      expect(entity.avatarUrl, isNull);
    });

    test('implements value equality', () {
      final a = ClientProjectProvider.fromJson({
        'organization_id': 'a',
        'display_name': 'Alice',
      });
      final b = ClientProjectProvider.fromJson({
        'organization_id': 'a',
        'display_name': 'Alice',
      });
      expect(a, equals(b));
      expect(a.hashCode, equals(b.hashCode));
    });
  });

  group('ClientProjectEntry', () {
    test('parses amount and timestamp correctly', () {
      final entry = ClientProjectEntry.fromJson({
        'proposal_id': 'prop-1',
        'title': 'Redesign landing page',
        'amount': 150000, // cents
        'completed_at': '2025-11-03T09:15:00Z',
        'provider': {
          'organization_id': 'org-p-1',
          'display_name': 'Alice',
        },
      });

      expect(entry.proposalId, 'prop-1');
      expect(entry.title, 'Redesign landing page');
      expect(entry.amount, 150000);
      expect(entry.completedAt.year, 2025);
      expect(entry.provider.displayName, 'Alice');
    });

    test('accepts amount as a numeric string', () {
      final entry = ClientProjectEntry.fromJson({
        'proposal_id': 'p',
        'title': 't',
        'amount': '9900',
        'completed_at': '2026-01-01T00:00:00Z',
        'provider': const <String, dynamic>{},
      });
      expect(entry.amount, 9900);
    });
  });

  group('ClientProfile.fromJson', () {
    test('parses the complete contract payload', () {
      final profile = ClientProfile.fromJson({
        'organization_id': 'org-e-1',
        'type': 'enterprise',
        'company_name': 'Acme Corp',
        'avatar_url': 'https://cdn/logo.png',
        'client_description': 'We buy pixels.',
        'total_spent': 1234567,
        'review_count': 3,
        'average_rating': 4.8,
        'projects_completed_as_client': 5,
        'project_history': [
          {
            'proposal_id': 'p1',
            'title': 'Logo',
            'amount': 50000,
            'completed_at': '2026-02-01T10:00:00Z',
            'provider': {
              'organization_id': 'org-p-1',
              'display_name': 'Alice',
            },
          },
        ],
        'reviews': [
          {
            'id': 'r1',
            'proposal_id': 'p1',
            'reviewer_id': 'u1',
            'reviewed_id': 'u2',
            'global_rating': 5,
            'created_at': '2026-02-03T10:00:00Z',
          },
        ],
      });

      expect(profile.organizationId, 'org-e-1');
      expect(profile.type, 'enterprise');
      expect(profile.companyName, 'Acme Corp');
      expect(profile.totalSpent, 1234567);
      expect(profile.reviewCount, 3);
      expect(profile.averageRating, 4.8);
      expect(profile.projectsCompletedAsClient, 5);
      expect(profile.projectHistory, hasLength(1));
      expect(profile.projectHistory.first.title, 'Logo');
      expect(profile.reviews, hasLength(1));
      expect(profile.reviews.first.globalRating, 5);
      expect(profile.hasReviews, isTrue);
    });

    test('handles missing optional fields with zero defaults', () {
      final profile = ClientProfile.fromJson({
        'organization_id': 'org-a-1',
        'type': 'agency',
        'company_name': 'Studio',
      });

      expect(profile.totalSpent, 0);
      expect(profile.reviewCount, 0);
      expect(profile.averageRating, 0);
      expect(profile.projectsCompletedAsClient, 0);
      expect(profile.clientDescription, '');
      expect(profile.projectHistory, isEmpty);
      expect(profile.reviews, isEmpty);
      expect(profile.hasReviews, isFalse);
    });

    test('ignores non-list project_history/reviews safely', () {
      final profile = ClientProfile.fromJson({
        'organization_id': 'org-a-1',
        'type': 'agency',
        'company_name': 'Studio',
        'project_history': 'not-a-list',
        'reviews': 42,
      });

      expect(profile.projectHistory, isEmpty);
      expect(profile.reviews, isEmpty);
    });

    test('average_rating accepts int', () {
      final profile = ClientProfile.fromJson({
        'organization_id': 'org-a-1',
        'type': 'agency',
        'company_name': 'Studio',
        'average_rating': 4,
      });
      expect(profile.averageRating, 4.0);
    });
  });
}
