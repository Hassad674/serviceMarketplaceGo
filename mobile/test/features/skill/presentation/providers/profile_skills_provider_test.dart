import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/skill/domain/entities/catalog_entry.dart';
import 'package:marketplace_mobile/features/skill/domain/entities/profile_skill.dart';
import 'package:marketplace_mobile/features/skill/domain/repositories/skill_repository.dart';
import 'package:marketplace_mobile/features/skill/presentation/providers/profile_skills_provider.dart';

/// Hand-written fake repository. Lightweight — we control every
/// method's behaviour per-test. Mockito would be overkill for this.
class _FakeSkillRepository implements SkillRepository {
  _FakeSkillRepository({this.initial = const <ProfileSkill>[]});

  List<ProfileSkill> initial;
  List<String>? lastUpdatePayload;
  Exception? getError;
  Exception? updateError;
  int updateCalls = 0;

  @override
  Future<List<ProfileSkill>> getProfileSkills() async {
    if (getError != null) throw getError!;
    return initial;
  }

  @override
  Future<void> updateProfileSkills(List<String> skillTexts) async {
    updateCalls += 1;
    lastUpdatePayload = skillTexts;
    if (updateError != null) throw updateError!;
    // Persist the updated list so the refetch after save sees it.
    initial = [
      for (var i = 0; i < skillTexts.length; i++)
        ProfileSkill(
          skillText: skillTexts[i],
          displayText: skillTexts[i],
          position: i,
        ),
    ];
  }

  @override
  Future<SkillCatalogPage> fetchCatalog(
    String expertiseKey, {
    int limit = 50,
  }) async =>
      const SkillCatalogPage(skills: <CatalogEntry>[], total: 0);

  @override
  Future<List<CatalogEntry>> searchAutocomplete(
    String query, {
    int limit = 20,
  }) async =>
      const <CatalogEntry>[];

  @override
  Future<CatalogEntry> createUserSkill(String displayText) async =>
      CatalogEntry(
        skillText: displayText.toLowerCase(),
        displayText: displayText,
        expertiseKeys: const <String>[],
        isCurated: false,
        usageCount: 1,
      );
}

void main() {
  group('ProfileSkillsNotifier', () {
    test('transitions loading -> data on successful initial load', () async {
      final repo = _FakeSkillRepository(
        initial: [
          const ProfileSkill(
            skillText: 'typescript',
            displayText: 'TypeScript',
            position: 0,
          ),
        ],
      );
      final notifier = ProfileSkillsNotifier(repo);

      expect(notifier.state.skills.isLoading, isTrue);

      // Let the ctor's _load future settle.
      await Future<void>.delayed(Duration.zero);

      expect(notifier.state.skills.hasValue, isTrue);
      expect(notifier.state.skills.value!.length, 1);
      expect(notifier.state.skills.value!.first.skillText, 'typescript');
      expect(notifier.state.isSaving, isFalse);
      expect(notifier.state.error, isNull);

      notifier.dispose();
    });

    test('transitions to error state when the initial fetch fails', () async {
      final repo = _FakeSkillRepository()..getError = Exception('boom');
      final notifier = ProfileSkillsNotifier(repo);

      await Future<void>.delayed(Duration.zero);

      final async = notifier.state.skills;
      expect(async is AsyncError, isTrue);
      notifier.dispose();
    });

    test('save() returns true and refetches the list on success', () async {
      final repo = _FakeSkillRepository();
      final notifier = ProfileSkillsNotifier(repo);
      await Future<void>.delayed(Duration.zero);

      final ok = await notifier.save(['figma', 'typescript']);

      expect(ok, isTrue);
      expect(repo.lastUpdatePayload, ['figma', 'typescript']);
      expect(notifier.state.skills.value!.length, 2);
      expect(notifier.state.skills.value![0].skillText, 'figma');
      expect(notifier.state.skills.value![1].skillText, 'typescript');
      expect(notifier.state.isSaving, isFalse);
      expect(notifier.state.error, isNull);

      notifier.dispose();
    });

    test('save() returns false and surfaces error on failure', () async {
      final repo = _FakeSkillRepository()..updateError = Exception('bad');
      final notifier = ProfileSkillsNotifier(repo);
      await Future<void>.delayed(Duration.zero);

      final ok = await notifier.save(['figma']);

      expect(ok, isFalse);
      expect(notifier.state.error, 'generic');
      expect(notifier.state.isSaving, isFalse);

      notifier.dispose();
    });

    test('clearError() resets the error sentinel', () async {
      final repo = _FakeSkillRepository()..updateError = Exception('bad');
      final notifier = ProfileSkillsNotifier(repo);
      await Future<void>.delayed(Duration.zero);
      await notifier.save(['x']);
      expect(notifier.state.error, isNotNull);

      notifier.clearError();

      expect(notifier.state.error, isNull);
      notifier.dispose();
    });

    test('refresh() reloads from the repository', () async {
      final repo = _FakeSkillRepository();
      final notifier = ProfileSkillsNotifier(repo);
      await Future<void>.delayed(Duration.zero);
      expect(notifier.state.skills.value, isEmpty);

      repo.initial = [
        const ProfileSkill(
          skillText: 'rust',
          displayText: 'Rust',
          position: 0,
        ),
      ];
      await notifier.refresh();

      expect(notifier.state.skills.value!.single.skillText, 'rust');
      notifier.dispose();
    });
  });
}
