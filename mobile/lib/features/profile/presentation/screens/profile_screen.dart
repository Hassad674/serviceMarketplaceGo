import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../auth/presentation/providers/auth_provider.dart';

/// Profile screen showing user info and a logout button.
class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  String _roleLabelFrench(String? role) {
    switch (role) {
      case 'agency':
        return 'Agence';
      case 'enterprise':
        return 'Entreprise';
      case 'provider':
        return 'Freelance';
      default:
        return role ?? 'Inconnu';
    }
  }

  Color _roleColor(String? role) {
    switch (role) {
      case 'agency':
        return const Color(0xFF2563EB);
      case 'enterprise':
        return const Color(0xFF8B5CF6);
      case 'provider':
        return const Color(0xFFF43F5E);
      default:
        return const Color(0xFF64748B);
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    final user = authState.user;
    final displayName =
        user?['display_name'] as String? ?? user?['first_name'] as String? ?? '';
    final email = user?['email'] as String? ?? '';
    final role = user?['role'] as String?;
    final initials = _buildInitials(displayName);

    return Scaffold(
      appBar: AppBar(title: const Text('Mon Profil')),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            children: [
              const SizedBox(height: 16),

              // Avatar with initials
              CircleAvatar(
                radius: 40,
                backgroundColor: theme.colorScheme.primary.withValues(alpha: 0.1),
                child: Text(
                  initials,
                  style: TextStyle(
                    fontSize: 24,
                    fontWeight: FontWeight.bold,
                    color: theme.colorScheme.primary,
                  ),
                ),
              ),
              const SizedBox(height: 16),

              // Name
              Text(
                displayName.isNotEmpty ? displayName : 'Utilisateur',
                style: theme.textTheme.headlineMedium,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 4),

              // Email
              Text(
                email,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                ),
              ),
              const SizedBox(height: 12),

              // Role badge
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(
                  color: _roleColor(role).withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(16),
                ),
                child: Text(
                  _roleLabelFrench(role),
                  style: TextStyle(
                    color: _roleColor(role),
                    fontWeight: FontWeight.w600,
                    fontSize: 13,
                  ),
                ),
              ),

              const Spacer(),

              // Logout button
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  onPressed: () async {
                    await ref.read(authProvider.notifier).logout();
                    if (context.mounted) {
                      context.go(RoutePaths.login);
                    }
                  },
                  icon: Icon(
                    Icons.logout,
                    color: theme.colorScheme.error,
                  ),
                  label: Text(
                    'Se deconnecter',
                    style: TextStyle(color: theme.colorScheme.error),
                  ),
                  style: OutlinedButton.styleFrom(
                    side: BorderSide(
                      color: theme.colorScheme.error.withValues(alpha: 0.3),
                    ),
                    minimumSize: const Size(double.infinity, 48),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ),
    );
  }

  String _buildInitials(String name) {
    if (name.isEmpty) return '?';
    final parts = name.trim().split(RegExp(r'\s+'));
    if (parts.length == 1) return parts[0][0].toUpperCase();
    return '${parts[0][0]}${parts.last[0]}'.toUpperCase();
  }
}
