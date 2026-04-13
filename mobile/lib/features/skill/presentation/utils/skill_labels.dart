import 'package:flutter/widgets.dart';

import '../../../../l10n/app_localizations.dart';

/// Maps a canonical expertise domain key to its localized label for
/// the skills browser panels.
///
/// This duplicates the expertise feature's label map on purpose:
/// the skills feature is strictly independent (rule #3 of the
/// modularity charter — features never import each other) and this
/// file is the only touch point with the AppLocalizations keys. If
/// a new expertise domain ever ships, both files must be updated in
/// lockstep — a small price for keeping the feature removable.
///
/// Unknown keys fall back to the raw key so a rolling backend
/// deploy never shows a blank header.
String localizedDomainLabel(BuildContext context, String key) {
  final loc = AppLocalizations.of(context)!;
  switch (key) {
    case 'development':
      return loc.expertiseDomainDevelopment;
    case 'data_ai_ml':
      return loc.expertiseDomainDataAiMl;
    case 'design_ui_ux':
      return loc.expertiseDomainDesignUiUx;
    case 'design_3d_animation':
      return loc.expertiseDomainDesign3dAnimation;
    case 'video_motion':
      return loc.expertiseDomainVideoMotion;
    case 'photo_audiovisual':
      return loc.expertiseDomainPhotoAudiovisual;
    case 'marketing_growth':
      return loc.expertiseDomainMarketingGrowth;
    case 'writing_translation':
      return loc.expertiseDomainWritingTranslation;
    case 'business_dev_sales':
      return loc.expertiseDomainBusinessDevSales;
    case 'consulting_strategy':
      return loc.expertiseDomainConsultingStrategy;
    case 'product_ux_research':
      return loc.expertiseDomainProductUxResearch;
    case 'ops_admin_support':
      return loc.expertiseDomainOpsAdminSupport;
    case 'legal':
      return loc.expertiseDomainLegal;
    case 'finance_accounting':
      return loc.expertiseDomainFinanceAccounting;
    case 'hr_recruitment':
      return loc.expertiseDomainHrRecruitment;
    default:
      return key;
  }
}
