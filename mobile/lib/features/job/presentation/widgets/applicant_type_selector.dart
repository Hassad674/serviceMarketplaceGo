import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';

/// A segmented button selector for choosing who can apply to the job.
///
/// Options: All / Freelancers / Agencies.
class ApplicantTypeSelector extends StatelessWidget {
  const ApplicantTypeSelector({
    super.key,
    required this.selected,
    required this.onChanged,
  });

  final ApplicantType selected;
  final ValueChanged<ApplicantType> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.jobApplicantType, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        SizedBox(
          width: double.infinity,
          child: SegmentedButton<ApplicantType>(
            segments: [
              ButtonSegment(
                value: ApplicantType.all,
                label: Text(l10n.jobApplicantAll),
                icon: const Icon(Icons.groups_outlined, size: 18),
              ),
              ButtonSegment(
                value: ApplicantType.freelancers,
                label: Text(l10n.jobApplicantFreelancers),
                icon: const Icon(Icons.person_outline, size: 18),
              ),
              ButtonSegment(
                value: ApplicantType.agencies,
                label: Text(l10n.jobApplicantAgencies),
                icon: const Icon(Icons.business_outlined, size: 18),
              ),
            ],
            selected: {selected},
            onSelectionChanged: (set) => onChanged(set.first),
            style: ButtonStyle(
              shape: WidgetStatePropertyAll(
                RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}
