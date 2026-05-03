import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/theme/app_palette.dart';

/// Green CTA used to accept a counter-proposal or cancellation request.
class DisputeAcceptButton extends StatelessWidget {
  const DisputeAcceptButton({
    super.key,
    required this.onPressed,
    required this.label,
  });

  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.check_circle, size: 16),
      label: Text(label),
      style: ElevatedButton.styleFrom(
        backgroundColor: AppPalette.green600,
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
      ),
    );
  }
}

/// Red outlined button used to reject a counter-proposal.
class DisputeRejectButton extends StatelessWidget {
  const DisputeRejectButton({
    super.key,
    required this.onPressed,
    required this.label,
  });

  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.cancel, size: 16),
      label: Text(label),
      style: OutlinedButton.styleFrom(
        foregroundColor: AppPalette.red600,
        side: const BorderSide(color: AppPalette.red300),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
      ),
    );
  }
}

/// Amber CTA used to launch a counter-proposal flow.
class DisputeCounterButton extends StatelessWidget {
  const DisputeCounterButton({
    super.key,
    required this.onPressed,
    required this.label,
  });

  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return ElevatedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.swap_horiz, size: 16),
      label: Text(label),
      style: ElevatedButton.styleFrom(
        backgroundColor: AppPalette.amber600,
        foregroundColor: Colors.white,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
      ),
    );
  }
}

/// Plain text button used to request dispute cancellation.
class DisputeCancelButton extends StatelessWidget {
  const DisputeCancelButton({
    super.key,
    required this.onPressed,
    required this.label,
  });

  final VoidCallback onPressed;
  final String label;

  @override
  Widget build(BuildContext context) {
    return TextButton(
      onPressed: onPressed,
      child: Text(label),
    );
  }
}
