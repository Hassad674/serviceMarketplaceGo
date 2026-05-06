import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/search/presentation/widgets/public_profile/public_profile_helpers.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Future<AppLocalizations> _loadEn() async {
  return AppLocalizations.delegate.load(const Locale('en'));
}

void main() {
  group('pickDirectPricing', () {
    test('returns the direct row when present', () {
      final pricing = pickDirectPricing({
        'pricing': [
          {
            'kind': 'direct',
            'type': 'hourly',
            'min_amount': 10000,
            'currency': 'EUR',
            'negotiable': false,
            'note': '',
          },
        ],
      });
      expect(pricing, isNotNull);
    });

    test('returns null when pricing is null', () {
      expect(pickDirectPricing({}), isNull);
    });

    test('returns null when no direct row exists', () {
      final pricing = pickDirectPricing({
        'pricing': [
          {
            'kind': 'commission',
            'type': 'commission_flat',
            'min_amount': 10000,
            'currency': 'EUR',
            'negotiable': false,
            'note': '',
          },
        ],
      });
      expect(pricing, isNull);
    });

    test('skips malformed rows without crashing', () {
      final pricing = pickDirectPricing({
        'pricing': [
          {'kind': 'direct'}, // missing required fields
        ],
      });
      expect(pricing, isNull);
    });
  });

  group('readIntField', () {
    test('parses int values', () {
      expect(readIntField(42), 42);
    });

    test('truncates double values', () {
      expect(readIntField(3.7), 3);
    });

    test('parses string values', () {
      expect(readIntField('123'), 123);
    });

    test('returns null for invalid strings', () {
      expect(readIntField('abc'), isNull);
    });

    test('returns null for null input', () {
      expect(readIntField(null), isNull);
    });
  });

  group('workModeLabel', () {
    test('localizes known work modes', () async {
      final l10n = await _loadEn();
      expect(workModeLabel('remote', l10n), l10n.tier1LocationWorkModeRemote);
      expect(workModeLabel('on_site', l10n), l10n.tier1LocationWorkModeOnSite);
      expect(workModeLabel('hybrid', l10n), l10n.tier1LocationWorkModeHybrid);
    });

    test('returns the raw key for unknown modes', () async {
      final l10n = await _loadEn();
      expect(workModeLabel('unknown', l10n), 'unknown');
    });
  });

  group('publicProfileRoleColor', () {
    testWidgets('returns the right color for each org type',
        (tester) async {
      late Color agency;
      late Color enterprise;
      late Color provider;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.light,
          home: Builder(
            builder: (context) {
              agency = publicProfileRoleColor(context, 'agency');
              enterprise = publicProfileRoleColor(context, 'enterprise');
              provider = publicProfileRoleColor(context, 'provider_personal');
              return const SizedBox.shrink();
            },
          ),
        ),
      );
      // Soleil v2 collapses cool tones onto corail.
      final corail = AppTheme.light.colorScheme.primary;
      expect(agency, corail);
      expect(enterprise, corail);
      expect(provider, corail);
    });

    testWidgets('falls back to onSurfaceVariant for null or unknown',
        (tester) async {
      late Color nullColor;
      late Color mysteryColor;
      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.light,
          home: Builder(
            builder: (context) {
              nullColor = publicProfileRoleColor(context, null);
              mysteryColor = publicProfileRoleColor(context, 'mystery');
              return const SizedBox.shrink();
            },
          ),
        ),
      );
      final mute = AppTheme.light.colorScheme.onSurfaceVariant;
      expect(nullColor, mute);
      expect(mysteryColor, mute);
    });
  });

  group('buildInitialsFromName', () {
    test('returns ? for empty', () {
      expect(buildInitialsFromName(''), '?');
    });

    test('returns ? for placeholder org names', () {
      expect(buildInitialsFromName('Org abc12345'), '?');
    });

    test('returns first letter for single-word names', () {
      expect(buildInitialsFromName('Alice'), 'A');
    });

    test('returns first+last initial for multi-word names', () {
      expect(buildInitialsFromName('Alice Bob'), 'AB');
      expect(buildInitialsFromName('Alice Bob Charles'), 'AC');
    });

    test('uppercases output', () {
      expect(buildInitialsFromName('alice bob'), 'AB');
    });
  });

  group('resolvePublicDisplayName', () {
    test('prefers nav-supplied name', () {
      expect(resolvePublicDisplayName({}, 'NavName'), 'NavName');
    });

    test('falls back to profile name', () {
      expect(
        resolvePublicDisplayName({'name': 'Profile name'}, null),
        'Profile name',
      );
    });

    test('falls back to org id stub', () {
      expect(
        resolvePublicDisplayName(
          {'organization_id': 'abcdefgh-1234'},
          null,
        ),
        'Org abcdefgh',
      );
    });

    test('ultimate fallback returns "Organization"', () {
      expect(resolvePublicDisplayName({}, null), 'Organization');
      expect(resolvePublicDisplayName({}, ''), 'Organization');
    });
  });
}
