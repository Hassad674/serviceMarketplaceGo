import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/skill/data/skill_repository_impl.dart';

import '../../../helpers/fake_api_client.dart';

void main() {
  late FakeApiClient fakeApi;
  late SkillRepositoryImpl repo;

  setUp(() {
    fakeApi = FakeApiClient();
    repo = SkillRepositoryImpl(fakeApi);
  });

  group('getProfileSkills', () {
    test('parses the wrapped { data: [...] } envelope', () async {
      fakeApi.getHandlers['/api/v1/profile/skills'] = (_) async {
        return FakeApiClient.ok({
          'data': [
            {
              'skill_text': 'typescript',
              'display_text': 'TypeScript',
              'position': 0,
            },
            {
              'skill_text': 'figma',
              'display_text': 'Figma',
              'position': 1,
            },
          ],
        });
      };

      final skills = await repo.getProfileSkills();

      expect(skills.length, 2);
      expect(skills[0].skillText, 'typescript');
      expect(skills[0].position, 0);
      expect(skills[1].skillText, 'figma');
      expect(skills[1].position, 1);
    });

    test('parses a raw list payload', () async {
      fakeApi.getHandlers['/api/v1/profile/skills'] = (_) async {
        return FakeApiClient.ok([
          {
            'skill_text': 'rust',
            'display_text': 'Rust',
            'position': 0,
          },
        ]);
      };

      final skills = await repo.getProfileSkills();

      expect(skills.length, 1);
      expect(skills[0].skillText, 'rust');
    });

    test('returns an empty list when data is null', () async {
      fakeApi.getHandlers['/api/v1/profile/skills'] = (_) async {
        return FakeApiClient.ok({'data': null});
      };

      final skills = await repo.getProfileSkills();

      expect(skills, isEmpty);
    });
  });

  group('updateProfileSkills', () {
    test('sends the ordered skill_texts list', () async {
      Map<String, dynamic>? captured;
      fakeApi.putHandlers['/api/v1/profile/skills'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({'data': {'ok': true}});
      };

      await repo.updateProfileSkills(['figma', 'typescript']);

      expect(captured!['skill_texts'], ['figma', 'typescript']);
    });
  });

  group('fetchCatalog', () {
    test('parses { skills, total }', () async {
      Map<String, dynamic>? capturedQuery;
      fakeApi.getHandlers['/api/v1/skills/catalog'] = (q) async {
        capturedQuery = q;
        return FakeApiClient.ok({
          'data': {
            'skills': [
              {
                'skill_text': 'swift',
                'display_text': 'Swift',
                'expertise_keys': ['development'],
                'is_curated': true,
                'usage_count': 123,
              },
            ],
            'total': 1,
          },
        });
      };

      final page = await repo.fetchCatalog('development');

      expect(capturedQuery!['expertise'], 'development');
      expect(capturedQuery!['limit'], 50);
      expect(page.skills.length, 1);
      expect(page.skills.first.skillText, 'swift');
      expect(page.total, 1);
    });

    test('falls back to skills.length when total is missing', () async {
      fakeApi.getHandlers['/api/v1/skills/catalog'] = (_) async {
        return FakeApiClient.ok({
          'data': {
            'skills': [
              {'skill_text': 'go', 'display_text': 'Go'},
              {'skill_text': 'ruby', 'display_text': 'Ruby'},
            ],
          },
        });
      };

      final page = await repo.fetchCatalog('development');

      expect(page.skills.length, 2);
      expect(page.total, 2);
    });
  });

  group('searchAutocomplete', () {
    test('short-circuits on empty query without hitting the API', () async {
      final results = await repo.searchAutocomplete('   ');
      expect(results, isEmpty);
      // Absence of handler proves no HTTP call.
    });

    test('parses a list of catalog entries', () async {
      Map<String, dynamic>? capturedQuery;
      fakeApi.getHandlers['/api/v1/skills/autocomplete'] = (q) async {
        capturedQuery = q;
        return FakeApiClient.ok({
          'data': [
            {
              'skill_text': 'javascript',
              'display_text': 'JavaScript',
              'expertise_keys': ['development'],
              'is_curated': true,
              'usage_count': 5,
            },
            {
              'skill_text': 'java',
              'display_text': 'Java',
              'expertise_keys': ['development'],
              'is_curated': true,
              'usage_count': 3,
            },
          ],
        });
      };

      final results = await repo.searchAutocomplete('java');

      expect(capturedQuery!['q'], 'java');
      expect(capturedQuery!['limit'], 20);
      expect(results.length, 2);
      expect(results[0].skillText, 'javascript');
      expect(results[1].displayText, 'Java');
    });
  });

  group('createUserSkill', () {
    test('returns the echoed catalog entry on success', () async {
      Map<String, dynamic>? captured;
      fakeApi.postHandlers['/api/v1/skills'] = (data) async {
        captured = data as Map<String, dynamic>;
        return FakeApiClient.ok({
          'data': {
            'skill_text': 'niche_framework',
            'display_text': 'Niche Framework',
            'expertise_keys': ['development'],
            'is_curated': false,
            'usage_count': 1,
          },
        });
      };

      final entry = await repo.createUserSkill('Niche Framework');

      expect(captured!['display_text'], 'Niche Framework');
      expect(entry.skillText, 'niche_framework');
      expect(entry.displayText, 'Niche Framework');
      expect(entry.isCurated, false);
    });

    test('falls back to a local entry if the server returns no body', () async {
      fakeApi.postHandlers['/api/v1/skills'] = (_) async {
        return FakeApiClient.ok(null);
      };

      final entry = await repo.createUserSkill('  Homemade Skill  ');

      expect(entry.skillText, 'homemade skill');
      expect(entry.displayText, 'Homemade Skill');
      expect(entry.isCurated, false);
    });
  });
}
