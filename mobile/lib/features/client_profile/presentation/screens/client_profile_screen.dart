import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/models/review.dart';
import '../../../../core/network/upload_service.dart';
import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/upload_bottom_sheet.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../../profile/presentation/providers/profile_provider.dart';
import '../../domain/entities/client_profile.dart';
import '../providers/client_profile_provider.dart';
import '../widgets/client_profile_description_widget.dart';
import '../widgets/client_profile_header.dart';
import '../widgets/client_project_history_widget.dart';
import '../widgets/client_reviews_list_widget.dart';

/// Permission gating the private edit surface. Mirrors the backend's
/// `org_client_profile.edit` permission key (also exposed on
/// [OrgPermission.orgClientProfileEdit]).
const _clientProfileEditPermission = OrgPermission.orgClientProfileEdit;

/// Private (authenticated) client-profile screen.
///
/// Only available to operators of `agency` and `enterprise` orgs.
/// Every other org type (provider_personal, solo, etc.) gets a
/// "not available" placeholder — flip the gate in [_ClientProfileBody]
/// when a new org type needs to opt in.
class ClientProfileScreen extends ConsumerWidget {
  const ClientProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final orgType = authState.organization?['type'] as String?;
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.clientProfileTitle),
      ),
      body: SafeArea(
        child: _isClientOrg(orgType)
            ? const _ClientProfileBody()
            : const _NotAvailablePlaceholder(),
      ),
    );
  }

  bool _isClientOrg(String? orgType) {
    return orgType == 'agency' || orgType == 'enterprise';
  }
}

// ---------------------------------------------------------------------------
// Main body — reads the same GET /api/v1/profile as the provider screen
// ---------------------------------------------------------------------------

class _ClientProfileBody extends ConsumerWidget {
  const _ClientProfileBody();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(profileProvider);
    final formState = ref.watch(clientProfileFormProvider);

    // Listen for save success / errors and surface a snackbar. We do
    // this in a `listen` so the notifier-state-change fires even when
    // the widget tree has not rebuilt for another reason.
    ref.listen<ClientProfileFormState>(clientProfileFormProvider,
        (previous, next) {
      if (previous?.status == next.status) return;
      final messenger = ScaffoldMessenger.of(context);
      final l10n = AppLocalizations.of(context)!;
      if (next.didSucceed) {
        messenger.showSnackBar(
          SnackBar(content: Text(l10n.clientProfileSaveSuccess)),
        );
        ref.read(clientProfileFormProvider.notifier).reset();
      } else if (next.didFail) {
        messenger.showSnackBar(
          SnackBar(
            content: Text(
              next.errorMessage?.isNotEmpty == true
                  ? '${l10n.clientProfileSaveError}: ${next.errorMessage}'
                  : l10n.clientProfileSaveError,
            ),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    });

    return profileAsync.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (error, _) => _ErrorState(
        onRetry: () => ref.invalidate(profileProvider),
      ),
      data: (profile) => _ClientProfileContent(
        profile: profile,
        isSaving: formState.isSaving,
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Content — renders header + description + history + reviews
// ---------------------------------------------------------------------------

class _ClientProfileContent extends ConsumerWidget {
  const _ClientProfileContent({
    required this.profile,
    required this.isSaving,
  });

  final Map<String, dynamic> profile;
  final bool isSaving;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authProvider);
    final canEdit = ref.watch(
      hasPermissionProvider(_clientProfileEditPermission),
    );
    final orgType = authState.organization?['type'] as String?;
    final companyName = _readString(profile['company_name']).isNotEmpty
        ? _readString(profile['company_name'])
        : _readString(authState.organization?['company_name']);
    final avatarUrl = _readString(profile['photo_url']).isNotEmpty
        ? _readString(profile['photo_url'])
        : null;
    final description = _readString(profile['client_description']);

    final totalSpent = _readInt(profile['total_spent']);
    final reviewCount = _readInt(profile['client_review_count']);
    final averageRating = _readDouble(profile['client_avg_rating']);
    final projectsCompleted =
        _readInt(profile['projects_completed_as_client']);

    final reviewsRaw = profile['client_reviews'];
    final reviews = reviewsRaw is List
        ? reviewsRaw
            .whereType<Map<String, dynamic>>()
            .map(Review.fromJson)
            .toList(growable: false)
        : const <Review>[];

    final historyRaw = profile['client_project_history'];
    final history = historyRaw is List
        ? historyRaw
            .whereType<Map<String, dynamic>>()
            .map(ClientProjectEntry.fromJson)
            .toList(growable: false)
        : const <ClientProjectEntry>[];

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          ClientProfileHeader(
            companyName: companyName,
            avatarUrl: avatarUrl,
            orgType: orgType,
            totalSpentCents: totalSpent,
            reviewCount: reviewCount,
            averageRating: averageRating,
            projectsCompleted: projectsCompleted,
            onAvatarTap:
                canEdit ? () => _uploadAvatar(context, ref) : null,
          ),
          const SizedBox(height: 16),
          ClientProfileDescriptionWidget(
            description: description,
            onTap: canEdit
                ? () => _openEditSheet(
                      context,
                      ref,
                      initialCompanyName: companyName,
                      initialDescription: description,
                      isSaving: isSaving,
                    )
                : null,
          ),
          if (!canEdit) ...[
            const SizedBox(height: 8),
            _PermissionDeniedHint(),
          ],
          const SizedBox(height: 16),
          ClientProjectHistoryWidget(projects: history),
          const SizedBox(height: 16),
          ClientReviewsListWidget(reviews: reviews),
          const SizedBox(height: 24),
        ],
      ),
    );
  }

  Future<void> _uploadAvatar(BuildContext context, WidgetRef ref) async {
    final l10n = AppLocalizations.of(context)!;
    showUploadBottomSheet(
      context: context,
      type: UploadMediaType.photo,
      onUpload: (File file) async {
        final uploadService = ref.read(uploadServiceProvider);
        await uploadService.uploadPhoto(file);
        ref.invalidate(profileProvider);
        if (context.mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.photoUpdated)),
          );
        }
      },
    );
  }

  void _openEditSheet(
    BuildContext context,
    WidgetRef ref, {
    required String initialCompanyName,
    required String initialDescription,
    required bool isSaving,
  }) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _EditClientProfileSheet(
        initialCompanyName: initialCompanyName,
        initialDescription: initialDescription,
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Edit sheet — company name + description
// ---------------------------------------------------------------------------

class _EditClientProfileSheet extends ConsumerStatefulWidget {
  const _EditClientProfileSheet({
    required this.initialCompanyName,
    required this.initialDescription,
  });

  final String initialCompanyName;
  final String initialDescription;

  @override
  ConsumerState<_EditClientProfileSheet> createState() =>
      _EditClientProfileSheetState();
}

class _EditClientProfileSheetState
    extends ConsumerState<_EditClientProfileSheet> {
  late final TextEditingController _nameCtrl;
  late final TextEditingController _descCtrl;

  @override
  void initState() {
    super.initState();
    _nameCtrl = TextEditingController(text: widget.initialCompanyName);
    _descCtrl = TextEditingController(text: widget.initialDescription);
  }

  @override
  void dispose() {
    _nameCtrl.dispose();
    _descCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final formState = ref.watch(clientProfileFormProvider);

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              l10n.clientProfileTitle,
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _nameCtrl,
              decoration: InputDecoration(
                labelText: l10n.clientProfileCompanyName,
                hintText: l10n.clientProfileCompanyNameHint,
                border: const OutlineInputBorder(),
              ),
              maxLength: 120,
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _descCtrl,
              decoration: InputDecoration(
                labelText: l10n.clientProfileDescription,
                hintText: l10n.clientProfileDescriptionHint,
                helperText: l10n.clientProfileDescriptionHelp,
                border: const OutlineInputBorder(),
              ),
              maxLines: 5,
              maxLength: 1000,
            ),
            const SizedBox(height: 8),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: formState.isSaving ? null : _submit,
                child: Text(
                  formState.isSaving
                      ? l10n.clientProfileSaving
                      : l10n.clientProfileSave,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final notifier = ref.read(clientProfileFormProvider.notifier);
    await notifier.submit(
      companyName: _nameCtrl.text.trim(),
      clientDescription: _descCtrl.text.trim(),
      onSuccess: () async {
        ref.invalidate(profileProvider);
      },
    );
    if (!mounted) return;
    final state = ref.read(clientProfileFormProvider);
    if (state.didSucceed) {
      Navigator.of(context).pop();
    }
  }
}

// ---------------------------------------------------------------------------
// Empty / not-available / permission-denied states
// ---------------------------------------------------------------------------

class _NotAvailablePlaceholder extends StatelessWidget {
  const _NotAvailablePlaceholder();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.business_outlined,
              size: 48,
              color: appColors?.mutedForeground,
            ),
            const SizedBox(height: 16),
            Text(
              l10n.clientProfileNotAvailable,
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyLarge,
            ),
          ],
        ),
      ),
    );
  }
}

class _PermissionDeniedHint extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: appColors?.muted,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
      ),
      child: Row(
        children: [
          Icon(
            Icons.lock_outline,
            size: 16,
            color: appColors?.mutedForeground,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              l10n.clientProfilePermissionDenied,
              style: theme.textTheme.bodySmall?.copyWith(
                color: appColors?.mutedForeground,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  const _ErrorState({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 48),
            const SizedBox(height: 16),
            ElevatedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh),
              label: Text(l10n.retry),
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// JSON read helpers — defensive against null/loose types
// ---------------------------------------------------------------------------

String _readString(dynamic value) {
  if (value is String) return value;
  return '';
}

int _readInt(dynamic value) {
  if (value == null) return 0;
  if (value is int) return value;
  if (value is double) return value.toInt();
  if (value is String) return int.tryParse(value) ?? 0;
  return 0;
}

double _readDouble(dynamic value) {
  if (value == null) return 0;
  if (value is double) return value;
  if (value is int) return value.toDouble();
  if (value is String) return double.tryParse(value) ?? 0;
  return 0;
}
