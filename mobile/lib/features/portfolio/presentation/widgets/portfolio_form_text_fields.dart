import 'package:flutter/material.dart';

const int kPortfolioMaxTitleLen = 200;
const int kPortfolioMaxDescLen = 2000;

/// Title input row with character counter for the portfolio form.
class PortfolioFormTitleField extends StatelessWidget {
  const PortfolioFormTitleField({
    super.key,
    required this.controller,
    required this.onChanged,
  });

  final TextEditingController controller;
  final VoidCallback onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              'Title *',
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            Text(
              '${controller.text.length}/$kPortfolioMaxTitleLen',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        TextField(
          controller: controller,
          maxLength: kPortfolioMaxTitleLen,
          onChanged: (_) => onChanged(),
          decoration: InputDecoration(
            hintText: 'e.g. E-commerce Redesign for Nike',
            counterText: '',
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 12,
            ),
          ),
        ),
      ],
    );
  }
}

/// Multi-line description input with character counter.
class PortfolioFormDescriptionField extends StatelessWidget {
  const PortfolioFormDescriptionField({
    super.key,
    required this.controller,
    required this.onChanged,
  });

  final TextEditingController controller;
  final VoidCallback onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(
              'Description',
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
            const Spacer(),
            Text(
              '${controller.text.length}/$kPortfolioMaxDescLen',
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        TextField(
          controller: controller,
          maxLength: kPortfolioMaxDescLen,
          maxLines: 4,
          onChanged: (_) => onChanged(),
          decoration: InputDecoration(
            hintText: 'Describe your role, the challenge and the results...',
            counterText: '',
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 12,
            ),
          ),
        ),
      ],
    );
  }
}

/// URL input for the project link with helper text.
class PortfolioFormLinkField extends StatelessWidget {
  const PortfolioFormLinkField({
    super.key,
    required this.controller,
  });

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Project link',
          style: theme.textTheme.labelLarge?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 6),
        TextField(
          controller: controller,
          keyboardType: TextInputType.url,
          decoration: InputDecoration(
            hintText: 'example.com',
            prefixIcon: const Icon(Icons.link, size: 18),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 14,
              vertical: 12,
            ),
          ),
        ),
        const SizedBox(height: 4),
        Text(
          "We'll automatically add https:// if you forget",
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}
