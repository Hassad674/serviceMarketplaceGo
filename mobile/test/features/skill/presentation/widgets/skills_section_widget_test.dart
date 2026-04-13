import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/skill/domain/entities/catalog_entry.dart';
import 'package:marketplace_mobile/features/skill/domain/entities/profile_skill.dart';
import 'package:marketplace_mobile/features/skill/domain/repositories/skill_repository.dart';
import 'package:marketplace_mobile/features/skill/presentation/providers/skill_repository_provider.dart';
import 'package:marketplace_mobile/features/skill/presentation/widgets/skills_section_widget.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

/// Minimal in-memory [SkillRepository] so the test does not hit
/// the real [ApiClient] (which would try to reach the backend).
class _InMemorySkillRepository implements SkillRepository {
  _InMemorySkillRepository(this._skills);

  List<ProfileSkill> _skills;

  @override
  Future<List<ProfileSkill>> getProfileSkills() async => _skills;

  @override
  Future<void> updateProfileSkills(List<String> skillTexts) async {
    _skills = [
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

Widget _wrap({
  required Widget child,
  required SkillRepository repository,
}) {
  return ProviderScope(
    overrides: [
      skillRepositoryProvider.overrideWithValue(repository),
    ],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  testWidgets('renders nothing for enterprise orgs', (tester) async {
    await tester.pumpWidget(
      _wrap(
        repository: _InMemorySkillRepository(const <ProfileSkill>[]),
        child: const SkillsSectionWidget(
          orgType: 'enterprise',
          expertiseKeys: <String>[],
          canEdit: true,
        ),
      ),
    );
    // Should render a SizedBox.shrink — no title, no edit button.
    expect(find.text('Skills'), findsNothing);
    expect(find.text('Edit my skills'), findsNothing);
  });

  testWidgets('renders the card and edit button for provider_personal',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        repository: _InMemorySkillRepository(const <ProfileSkill>[
          ProfileSkill(
            skillText: 'typescript',
            displayText: 'TypeScript',
            position: 0,
          ),
        ]),
        child: const SkillsSectionWidget(
          orgType: 'provider_personal',
          expertiseKeys: <String>['development'],
          canEdit: true,
        ),
      ),
    );
    // Let the StateNotifier load complete.
    await tester.pump();

    expect(find.text('Skills'), findsOneWidget);
    expect(find.text('TypeScript'), findsOneWidget);
    expect(find.text('Edit my skills'), findsOneWidget);
  });

  testWidgets('shows the empty state when the operator has no skills',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        repository: _InMemorySkillRepository(const <ProfileSkill>[]),
        child: const SkillsSectionWidget(
          orgType: 'agency',
          expertiseKeys: <String>['development'],
          canEdit: true,
        ),
      ),
    );
    await tester.pump();

    expect(find.text('Skills'), findsOneWidget);
    expect(find.text('No skills added yet'), findsOneWidget);
  });

  testWidgets('hides the edit button when canEdit is false', (tester) async {
    await tester.pumpWidget(
      _wrap(
        repository: _InMemorySkillRepository(const <ProfileSkill>[
          ProfileSkill(
            skillText: 'figma',
            displayText: 'Figma',
            position: 0,
          ),
        ]),
        child: const SkillsSectionWidget(
          orgType: 'agency',
          expertiseKeys: <String>['design_ui_ux'],
          canEdit: false,
        ),
      ),
    );
    await tester.pump();

    expect(find.text('Figma'), findsOneWidget);
    expect(find.text('Edit my skills'), findsNothing);
  });
}
