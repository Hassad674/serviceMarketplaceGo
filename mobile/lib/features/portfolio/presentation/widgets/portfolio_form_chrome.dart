import 'package:flutter/material.dart';

/// Top chrome of the portfolio form: a small drag handle bar and a
/// header row with title, helper subtitle and a close button.
class PortfolioFormChrome extends StatelessWidget {
  const PortfolioFormChrome({
    super.key,
    required this.isEdit,
    required this.onClose,
  });

  final bool isEdit;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Center(
          child: Container(
            margin: const EdgeInsets.only(top: 12, bottom: 8),
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: theme.dividerColor,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 4, 12, 8),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      isEdit ? 'Edit project' : 'Add project',
                      style: theme.textTheme.titleLarge?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    Text(
                      isEdit
                          ? 'Update your project details'
                          : 'Showcase a project with images, videos and a link',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: onClose,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

/// Bottom sheet content for choosing whether to add an image or a video.
///
/// Exposed as a standalone widget so the parent can `showModalBottomSheet`
/// with `(_) => PortfolioAddMediaSheet(...)`. The widget pops itself before
/// invoking the chosen handler.
class PortfolioAddMediaSheet extends StatelessWidget {
  const PortfolioAddMediaSheet({
    super.key,
    required this.onPickImage,
    required this.onPickVideo,
  });

  final VoidCallback onPickImage;
  final VoidCallback onPickVideo;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ListTile(
            leading: const Icon(Icons.image_outlined),
            title: const Text('Add an image'),
            onTap: () {
              Navigator.of(context).pop();
              onPickImage();
            },
          ),
          ListTile(
            leading: const Icon(Icons.videocam_outlined),
            title: const Text('Add a video'),
            onTap: () {
              Navigator.of(context).pop();
              onPickVideo();
            },
          ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}
