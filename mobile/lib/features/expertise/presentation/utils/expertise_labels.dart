import 'package:flutter/widgets.dart';

import '../../../../l10n/app_localizations.dart';

/// Maps a canonical expertise domain key to its localized label.
///
/// The mapping lives in `presentation/utils/` (and not in
/// `domain/`) because it depends on [AppLocalizations] and therefore
/// on Flutter's widget layer. The domain layer stays pure Dart.
///
/// Unknown keys are returned as-is to keep the UI resilient to a
/// backend that ships new keys before the client catches up.
String localizedExpertiseLabel(BuildContext context, String key) {
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
