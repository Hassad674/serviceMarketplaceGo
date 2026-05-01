import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:marketplace_mobile/shared/widgets/drawer/drawer_label_resolver.dart';

Future<AppLocalizations> _loadEn() async {
  return AppLocalizations.delegate.load(const Locale('en'));
}

void main() {
  test('resolveDrawerLabel returns the english label for known keys',
      () async {
    final l10n = await _loadEn();
    expect(resolveDrawerLabel(l10n, 'drawerDashboard'), l10n.drawerDashboard);
    expect(resolveDrawerLabel(l10n, 'drawerMessages'), l10n.drawerMessages);
    expect(resolveDrawerLabel(l10n, 'drawerProjects'), l10n.drawerProjects);
    expect(resolveDrawerLabel(l10n, 'drawerWallet'), l10n.drawerWallet);
    expect(
      resolveDrawerLabel(l10n, 'drawerNotifications'),
      l10n.drawerNotifications,
    );
  });

  test('resolveDrawerLabel returns hardcoded values for legacy keys',
      () async {
    final l10n = await _loadEn();
    expect(resolveDrawerLabel(l10n, 'drawerOpportunities'), 'Opportunités');
    expect(
      resolveDrawerLabel(l10n, 'drawerMyApplications'),
      'Mes candidatures',
    );
    expect(resolveDrawerLabel(l10n, 'drawerInvoices'), 'Mes factures');
  });

  test('resolveDrawerLabel falls back to the key for unknown values',
      () async {
    final l10n = await _loadEn();
    expect(resolveDrawerLabel(l10n, 'unknownKey'), 'unknownKey');
  });

  test('resolveDrawerRoleLabel returns localized role names', () async {
    final l10n = await _loadEn();
    expect(resolveDrawerRoleLabel(l10n, 'agency'), l10n.roleAgency);
    expect(resolveDrawerRoleLabel(l10n, 'enterprise'), l10n.roleEnterprise);
    expect(resolveDrawerRoleLabel(l10n, 'provider'), l10n.roleFreelance);
  });

  test('resolveDrawerRoleLabel falls back to raw role on unknown', () async {
    final l10n = await _loadEn();
    expect(resolveDrawerRoleLabel(l10n, 'mystery'), 'mystery');
  });
}

