import '../../../l10n/app_localizations.dart';

/// Resolves the localized label for a given drawer item key.
///
/// Mirrors the web sidebar's i18n strategy. Keeps the navigation
/// tile decoupled from the `AppLocalizations` schema.
String resolveDrawerLabel(AppLocalizations l10n, String key) {
  switch (key) {
    case 'drawerDashboard':
      return l10n.drawerDashboard;
    case 'drawerMessages':
      return l10n.drawerMessages;
    case 'drawerProjects':
      return l10n.drawerProjects;
    case 'drawerJobs':
      return l10n.drawerJobs;
    case 'drawerOpportunities':
      return 'Opportunités';
    case 'drawerMyApplications':
      return 'Mes candidatures';
    case 'drawerTeam':
      return l10n.drawerTeam;
    case 'drawerProfile':
      return l10n.drawerProfile;
    case 'navClientProfile':
      return l10n.navClientProfile;
    case 'navProviderProfile':
      return l10n.navProviderProfile;
    case 'drawerFindFreelancers':
      return l10n.drawerFindFreelancers;
    case 'drawerFindAgencies':
      return l10n.drawerFindAgencies;
    case 'drawerFindReferrers':
      return l10n.drawerFindReferrers;
    case 'drawerPaymentInfo':
      return l10n.drawerPaymentInfo;
    case 'drawerWallet':
      return l10n.drawerWallet;
    case 'drawerNotifications':
      return l10n.drawerNotifications;
    case 'drawerInvoices':
      return 'Mes factures';
    default:
      return key;
  }
}

/// Returns the localized role label for the drawer header badge.
String resolveDrawerRoleLabel(AppLocalizations l10n, String role) {
  switch (role) {
    case 'agency':
      return l10n.roleAgency;
    case 'enterprise':
      return l10n.roleEnterprise;
    case 'provider':
      return l10n.roleFreelance;
    default:
      return role;
  }
}
