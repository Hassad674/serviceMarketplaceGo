import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';

/// Role selection screen for registration.
///
/// Presents three cards: Agency, Freelance/Business Referrer, Enterprise.
/// Tapping a card navigates to the role-specific register form.
class RoleSelectionScreen extends StatelessWidget {
  const RoleSelectionScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.go(RoutePaths.login),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const SizedBox(height: 16),
              Text(
                'Join the marketplace',
                textAlign: TextAlign.center,
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 8),
              Text(
                'Choose your profile to get started',
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
                ),
              ),
              const SizedBox(height: 32),

              // Agency card
              _RoleCard(
                icon: Icons.business,
                title: 'Agency / IT Services',
                description:
                    'Manage your agency, providers, and missions',
                accentColor: const Color(0xFF2563EB), // blue-600
                onTap: () => context.go(RoutePaths.registerAgency),
              ),
              const SizedBox(height: 16),

              // Provider / Freelance card
              _RoleCard(
                icon: Icons.person,
                title: 'Freelance / Business Referrer',
                description:
                    'Offer your services or connect professionals together',
                accentColor: const Color(0xFFF43F5E), // rose-500
                onTap: () => context.go(RoutePaths.registerProvider),
              ),
              const SizedBox(height: 16),

              // Enterprise card
              _RoleCard(
                icon: Icons.corporate_fare,
                title: 'Enterprise',
                description:
                    'Post your projects and find the best providers',
                accentColor: const Color(0xFF10B981), // emerald-500
                onTap: () => context.go(RoutePaths.registerEnterprise),
              ),
              const SizedBox(height: 32),

              // Already have an account
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    'Already registered?',
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color:
                          theme.colorScheme.onSurface.withValues(alpha: 0.6),
                    ),
                  ),
                  TextButton(
                    onPressed: () => context.go(RoutePaths.login),
                    child: const Text('Sign In'),
                  ),
                ],
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ),
    );
  }
}

/// A tappable card representing a role option.
class _RoleCard extends StatelessWidget {
  const _RoleCard({
    required this.icon,
    required this.title,
    required this.description,
    required this.accentColor,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String description;
  final Color accentColor;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: theme.dividerColor,
          ),
        ),
        child: Row(
          children: [
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: accentColor.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(icon, color: accentColor, size: 24),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    description,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
                    ),
                  ),
                ],
              ),
            ),
            Icon(
              Icons.chevron_right,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ],
        ),
      ),
    );
  }
}
