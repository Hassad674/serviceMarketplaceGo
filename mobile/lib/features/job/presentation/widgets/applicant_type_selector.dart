import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';

/// M-09 — Soleil v2 segmented selector for the "Qui peut postuler ?" question.
///
/// Ports the picker to ivoire/corail tokens (no `Color(0xFF...)` literals):
/// active option = corail-soft fill + corail border + corail-deep label,
/// off option = white surface + sable-light border + tabac label.
/// Behaviour and prop interface are unchanged.
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
    final appColors = theme.extension<AppColors>();
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.jobApplicantType.toUpperCase(),
          style: SoleilTextStyles.mono.copyWith(
            color: mute,
            fontSize: 11,
            fontWeight: FontWeight.w700,
            letterSpacing: 0.8,
          ),
        ),
        const SizedBox(height: 10),
        _OptionTile(
          icon: Icons.groups_outlined,
          label: l10n.jobApplicantAll,
          selected: selected == ApplicantType.all,
          onTap: () => onChanged(ApplicantType.all),
        ),
        const SizedBox(height: 8),
        _OptionTile(
          icon: Icons.person_outline,
          label: l10n.jobApplicantFreelancers,
          selected: selected == ApplicantType.freelancers,
          onTap: () => onChanged(ApplicantType.freelancers),
        ),
        const SizedBox(height: 8),
        _OptionTile(
          icon: Icons.business_outlined,
          label: l10n.jobApplicantAgencies,
          selected: selected == ApplicantType.agencies,
          onTap: () => onChanged(ApplicantType.agencies),
        ),
      ],
    );
  }
}

class _OptionTile extends StatelessWidget {
  const _OptionTile({
    required this.icon,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final primaryDeep = appColors?.primaryDeep ?? primary;
    final accentSoft = appColors?.accentSoft ?? theme.colorScheme.primaryContainer;
    final border = appColors?.border ?? theme.colorScheme.outline;
    final foreground = theme.colorScheme.onSurface;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOut,
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          decoration: BoxDecoration(
            color: selected ? accentSoft : theme.colorScheme.surfaceContainerLowest,
            border: Border.all(
              color: selected ? primary : border,
              width: 1.5,
            ),
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          ),
          child: Row(
            children: [
              Icon(
                icon,
                size: 20,
                color: selected ? primaryDeep : foreground,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  label,
                  style: SoleilTextStyles.bodyEmphasis.copyWith(
                    color: selected ? primaryDeep : foreground,
                  ),
                ),
              ),
              _Radio(selected: selected),
            ],
          ),
        ),
      ),
    );
  }
}

class _Radio extends StatelessWidget {
  const _Radio({required this.selected});

  final bool selected;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final borderStrong = appColors?.borderStrong ?? theme.colorScheme.outline;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      width: 18,
      height: 18,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: selected ? primary : theme.colorScheme.surfaceContainerLowest,
        border: Border.all(
          color: selected ? primary : borderStrong,
          width: 1.5,
        ),
      ),
      child: selected
          ? Center(
              child: Container(
                width: 6,
                height: 6,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: theme.colorScheme.onPrimary,
                ),
              ),
            )
          : null,
    );
  }
}
