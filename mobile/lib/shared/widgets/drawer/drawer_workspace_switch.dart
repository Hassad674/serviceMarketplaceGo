import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/router/app_router.dart';
import '../../../core/theme/app_theme.dart';
import '../../../l10n/app_localizations.dart';

const drawerWorkspacePref = 'workspace_mode';

/// Toggle pill switching providers between freelance and referrer
/// workspace modes (only visible to provider+referrer_enabled users).
class DrawerWorkspaceSwitch extends StatefulWidget {
  const DrawerWorkspaceSwitch({super.key, required this.l10n});

  final AppLocalizations l10n;

  @override
  State<DrawerWorkspaceSwitch> createState() => _DrawerWorkspaceSwitchState();
}

class _DrawerWorkspaceSwitchState extends State<DrawerWorkspaceSwitch> {
  bool _isReferrerMode = false;

  @override
  void initState() {
    super.initState();
    SharedPreferences.getInstance().then((prefs) {
      if (mounted) {
        setState(() {
          _isReferrerMode =
              prefs.getString(drawerWorkspacePref) == 'referrer';
        });
      }
    });
  }

  Future<void> _toggleWorkspace() async {
    final newMode = !_isReferrerMode;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(
      drawerWorkspacePref,
      newMode ? 'referrer' : 'freelance',
    );
    if (!mounted) return;
    setState(() => _isReferrerMode = newMode);
    Navigator.of(context).pop();
    GoRouter.of(context).go(
      newMode ? RoutePaths.dashboardReferrer : RoutePaths.dashboard,
    );
  }

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final isRef = _isReferrerMode;
    final label = isRef
        ? widget.l10n.drawerSwitchToFreelance
        : widget.l10n.drawerSwitchToReferrer;
    final icon = isRef ? Icons.swap_horiz : Icons.auto_awesome;
    final fgColor = isRef
        ? (isDark ? const Color(0xFF6EE7B7) : const Color(0xFF059669))
        : Colors.white;
    final bgDecor = isRef
        ? BoxDecoration(
            color: isDark
                ? const Color(0xFF065F46).withValues(alpha: 0.25)
                : const Color(0xFFECFDF5),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          )
        : BoxDecoration(
            gradient: const LinearGradient(
              colors: [Color(0xFFF43F5E), Color(0xFF8B5CF6)],
            ),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          );

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: GestureDetector(
        onTap: _toggleWorkspace,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: bgDecor,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(icon, size: 18, color: fgColor),
              const SizedBox(width: 8),
              Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: fgColor,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
