import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_exception.dart';
import '../../../../core/theme/app_palette.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/team_repository_impl.dart';
import '../../domain/entities/role_permissions_matrix.dart';
import '../providers/team_provider.dart';

part 'role_permissions_editor_parts.dart';
part 'role_permissions_editor_atoms.dart';

/// Editable roles shown as expandable cards in the editor. Owner is
/// excluded on purpose — it is rendered in read-only fashion above the
/// editable list, and the backend forbids any override on it anyway.
const List<String> _editableRoles = <String>['admin', 'member', 'viewer'];

/// Permissions that must never appear in the editor grid because
/// they are locked at the domain layer. We still show them in the
/// "Owner-exclusive" footer so users understand what the Owner can
/// do that no other role ever will.
const Set<String> _nonOverridablePermissionKeys = <String>{
  'org.delete',
  'team.transfer_ownership',
  'wallet.withdraw',
  'kyc.manage',
  'team.manage_role_permissions',
};

/// Full role permissions panel shown on the mobile team screen.
///
/// Modes:
///   - Owner: editable — one expandable card per editable role with
///     a Material Switch per permission + a save bar that appears as
///     soon as the Owner makes a change.
///   - Non-Owner: read-only — the exact same matrix, but all switches
///     are disabled and the save bar is never rendered.
///
/// The widget is a stateful consumer because it holds the
/// per-role pending-change maps in local state. Applying them only
/// happens on save, so the Owner can reset without a network call.
class RolePermissionsEditor extends ConsumerStatefulWidget {
  const RolePermissionsEditor({
    super.key,
    required this.orgId,
    required this.canEdit,
  });

  final String orgId;
  final bool canEdit;

  @override
  ConsumerState<RolePermissionsEditor> createState() =>
      _RolePermissionsEditorState();
}

class _RolePermissionsEditorState
    extends ConsumerState<RolePermissionsEditor> {
  /// role -> (permission key -> desired granted value). Empty when no
  /// changes are pending for that role.
  final Map<String, Map<String, bool>> _pending = <String, Map<String, bool>>{
    'admin': <String, bool>{},
    'member': <String, bool>{},
    'viewer': <String, bool>{},
  };
  String _expandedRole = 'admin';
  bool _saving = false;

  @override
  Widget build(BuildContext context) {
    final matrixAsync = ref.watch(rolePermissionsMatrixProvider);
    return matrixAsync.when(
      data: (matrix) => _buildEditor(context, matrix),
      loading: () => const _EditorSkeleton(),
      error: (err, _) => _ErrorCard(
        onRetry: () => ref.invalidate(rolePermissionsMatrixProvider),
      ),
    );
  }

  Widget _buildEditor(BuildContext context, RolePermissionsMatrix matrix) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final pendingTotal = _pending.values.fold<int>(
      0,
      (sum, map) => sum + map.length,
    );

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
          width: 1,
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _HeaderSection(
            title: l10n.teamRolePermissionsTitle,
            subtitle: l10n.teamRolePermissionsSubtitle,
          ),
          if (!widget.canEdit)
            _ReadOnlyBanner(
              title: l10n.teamRolePermissionsReadOnlyTitle,
              description: l10n.teamRolePermissionsReadOnlyDescription,
            ),
          const Divider(height: 1),
          Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                for (final roleKey in _editableRoles) ...[
                  _RoleCard(
                    role: roleKey,
                    row: matrix.rowFor(roleKey),
                    expanded: _expandedRole == roleKey,
                    onToggleExpand: () => setState(
                      () => _expandedRole =
                          _expandedRole == roleKey ? '' : roleKey,
                    ),
                    pending: _pending[roleKey] ?? const <String, bool>{},
                    canEdit: widget.canEdit && !_saving,
                    onTogglePermission: (cell) => _togglePermission(
                      roleKey: roleKey,
                      cell: cell,
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
                _OwnerExclusiveSection(
                  cells: _collectLockedCells(matrix),
                ),
              ],
            ),
          ),
          if (widget.canEdit && pendingTotal > 0)
            _SaveBar(
              pendingCount: pendingTotal,
              saving: _saving,
              onDiscard: _saving ? null : _discardPending,
              onSave: _saving ? null : () => _confirmAndSave(matrix),
            ),
        ],
      ),
    );
  }

  void _discardPending() {
    setState(() {
      for (final role in _editableRoles) {
        _pending[role] = <String, bool>{};
      }
    });
  }

  void _togglePermission({
    required String roleKey,
    required RolePermissionCell cell,
  }) {
    if (!widget.canEdit || _saving || cell.locked) return;
    setState(() {
      final pending = _pending[roleKey] ??= <String, bool>{};
      final serverGranted = cell.granted;
      final currentlyPending = pending.containsKey(cell.key);
      final newValue =
          currentlyPending ? !pending[cell.key]! : !serverGranted;
      if (newValue == serverGranted) {
        pending.remove(cell.key);
      } else {
        pending[cell.key] = newValue;
      }
    });
  }

  Future<void> _confirmAndSave(RolePermissionsMatrix matrix) async {
    // Find the first role with pending changes. We save one role at a
    // time to keep the flow explicit — the Owner can tap save again if
    // another role has pending changes too.
    String? targetRole;
    for (final role in _editableRoles) {
      final pending = _pending[role];
      if (pending != null && pending.isNotEmpty) {
        targetRole = role;
        break;
      }
    }
    if (targetRole == null) return;

    final row = matrix.rowFor(targetRole);
    if (row == null) return;

    final l10n = AppLocalizations.of(context)!;
    final pendingCount = _pending[targetRole]!.length;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogCtx) => AlertDialog(
        title: Text(l10n.teamRolePermissionsConfirmTitle),
        content: Text(
          l10n.teamRolePermissionsConfirmDescription(
            pendingCount,
            _localizedRoleLabel(l10n, targetRole!),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx).pop(false),
            child: Text(l10n.teamRolePermissionsCancelButton),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogCtx).pop(true),
            child: Text(l10n.teamRolePermissionsConfirmButton),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await _performSave(row: row, roleKey: targetRole);
  }

  Future<void> _performSave({
    required RolePermissionsRow row,
    required String roleKey,
  }) async {
    setState(() => _saving = true);
    final l10n = AppLocalizations.of(context)!;
    final pending = _pending[roleKey] ?? const <String, bool>{};
    final overrides = _buildOverridesPayload(row: row, pending: pending);
    try {
      final repo = ref.read(teamRepositoryProvider);
      final result = await repo.updateRolePermissions(
        orgId: widget.orgId,
        role: roleKey,
        overrides: overrides,
      );
      if (!mounted) return;
      setState(() => _pending[roleKey] = <String, bool>{});
      ref.invalidate(rolePermissionsMatrixProvider);
      // Members list permissions surfaces may have changed if the
      // current user is in the affected role — refresh it so the
      // team screen reflects the new state immediately.
      ref.invalidate(teamMembersProvider);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            l10n.teamRolePermissionsSaveSuccess(result.affectedMembers),
          ),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final apiError = ApiException.fromDioException(e);
      _showErrorSnack(
        apiError.message.isNotEmpty
            ? apiError.message
            : l10n.teamRolePermissionsSaveFailed,
      );
    } catch (_) {
      if (!mounted) return;
      _showErrorSnack(l10n.teamRolePermissionsSaveFailed);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  void _showErrorSnack(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  /// Builds the full "desired state" map of every non-locked cell for
  /// the target role. This matches the web editor's behaviour: the
  /// PATCH body describes the complete intent so the server can
  /// collapse any cell that equals the default back to "no override".
  Map<String, bool> _buildOverridesPayload({
    required RolePermissionsRow row,
    required Map<String, bool> pending,
  }) {
    final out = <String, bool>{};
    for (final cell in row.permissions) {
      if (cell.locked) continue;
      if (_nonOverridablePermissionKeys.contains(cell.key)) continue;
      final override =
          pending.containsKey(cell.key) ? pending[cell.key]! : cell.granted;
      out[cell.key] = override;
    }
    return out;
  }

  /// Collects every locked / non-overridable cell encountered in the
  /// matrix. The backend emits them per-role, but we only want to
  /// show them once — so we deduplicate by key and take the first
  /// occurrence's metadata.
  List<RolePermissionCell> _collectLockedCells(
    RolePermissionsMatrix matrix,
  ) {
    final seen = <String, RolePermissionCell>{};
    for (final row in matrix.roles) {
      for (final cell in row.permissions) {
        if (cell.locked ||
            _nonOverridablePermissionKeys.contains(cell.key)) {
          seen.putIfAbsent(cell.key, () => cell);
        }
      }
    }
    return seen.values.toList();
  }

  String _localizedRoleLabel(AppLocalizations l10n, String role) {
    switch (role) {
      case 'admin':
        return l10n.teamRolePermissionRoleAdmin;
      case 'member':
        return l10n.teamRolePermissionRoleMember;
      case 'viewer':
        return l10n.teamRolePermissionRoleViewer;
      case 'owner':
        return l10n.teamRolePermissionRoleOwner;
      default:
        return role;
    }
  }
}
