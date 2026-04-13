import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/skill/domain/entities/catalog_entry.dart';
import 'package:marketplace_mobile/features/skill/domain/entities/profile_skill.dart';
import 'package:marketplace_mobile/features/skill/domain/entities/skill_limits.dart';

void main() {
  group('CatalogEntry.fromJson', () {
    test('parses a complete backend payload', () {
      final entry = CatalogEntry.fromJson(<String, dynamic>{
        'skill_text': 'typescript',
        'display_text': 'TypeScript',
        'expertise_keys': ['development', 'data_ai_ml'],
        'is_curated': true,
        'usage_count': 42,
      });
      expect(entry.skillText, 'typescript');
      expect(entry.displayText, 'TypeScript');
      expect(entry.expertiseKeys, ['development', 'data_ai_ml']);
      expect(entry.isCurated, true);
      expect(entry.usageCount, 42);
    });

    test('tolerates missing display_text by falling back to skill_text', () {
      final entry = CatalogEntry.fromJson(<String, dynamic>{
        'skill_text': 'rust',
      });
      expect(entry.displayText, 'rust');
      expect(entry.expertiseKeys, isEmpty);
      expect(entry.isCurated, false);
      expect(entry.usageCount, 0);
    });

    test('coerces numeric strings to int for usage_count', () {
      final entry = CatalogEntry.fromJson(<String, dynamic>{
        'skill_text': 'go',
        'usage_count': '17',
      });
      expect(entry.usageCount, 17);
    });

    test('drops non-string entries from expertise_keys', () {
      final entry = CatalogEntry.fromJson(<String, dynamic>{
        'skill_text': 'kotlin',
        'expertise_keys': ['development', 42, null],
      });
      expect(entry.expertiseKeys, ['development']);
    });

    test('equality and hashCode are based on skillText', () {
      const a = CatalogEntry(
        skillText: 'dart',
        displayText: 'Dart',
        expertiseKeys: <String>[],
        isCurated: true,
        usageCount: 5,
      );
      const b = CatalogEntry(
        skillText: 'dart',
        displayText: 'Dart (different label)',
        expertiseKeys: <String>['development'],
        isCurated: false,
        usageCount: 99,
      );
      expect(a, b);
      expect(a.hashCode, b.hashCode);
    });
  });

  group('ProfileSkill.fromJson', () {
    test('parses a standard payload', () {
      final skill = ProfileSkill.fromJson(<String, dynamic>{
        'skill_text': 'figma',
        'display_text': 'Figma',
        'position': 3,
      });
      expect(skill.skillText, 'figma');
      expect(skill.displayText, 'Figma');
      expect(skill.position, 3);
    });

    test('falls back to skill_text when display_text is missing', () {
      final skill = ProfileSkill.fromJson(<String, dynamic>{
        'skill_text': 'python',
      });
      expect(skill.displayText, 'python');
      expect(skill.position, 0);
    });
  });

  group('SkillLimits', () {
    test('agency has 40 max skills', () {
      expect(SkillLimits.maxForOrgType('agency'), 40);
      expect(SkillLimits.isFeatureEnabledForOrgType('agency'), true);
    });

    test('provider_personal has 25 max skills', () {
      expect(SkillLimits.maxForOrgType('provider_personal'), 25);
      expect(SkillLimits.isFeatureEnabledForOrgType('provider_personal'), true);
    });

    test('enterprise disables the feature', () {
      expect(SkillLimits.maxForOrgType('enterprise'), 0);
      expect(SkillLimits.isFeatureEnabledForOrgType('enterprise'), false);
    });

    test('unknown org types disable the feature', () {
      expect(SkillLimits.maxForOrgType(null), 0);
      expect(SkillLimits.maxForOrgType('random'), 0);
      expect(SkillLimits.isFeatureEnabledForOrgType(null), false);
    });
  });
}
