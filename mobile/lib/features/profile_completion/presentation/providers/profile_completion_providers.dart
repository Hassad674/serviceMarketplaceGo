import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/profile_completion_repository_impl.dart';
import '../../domain/entities/profile_completion_report.dart';
import '../../domain/repositories/profile_completion_repository.dart';

/// Repository wiring — picks up the shared [apiClientProvider] so
/// the bar inherits the auth interceptor stack without any per-feature
/// setup.
final profileCompletionRepositoryProvider =
    Provider<ProfileCompletionRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ProfileCompletionRepositoryImpl(api);
});

/// Loads the report for the authenticated user. Auto-disposed so a
/// re-mount after a profile edit re-runs the fetch — matches the
/// staleTime contract on web (30s) without managing an explicit
/// timer.
final profileCompletionProvider =
    FutureProvider.autoDispose<ProfileCompletionReport>((ref) async {
  final repo = ref.watch(profileCompletionRepositoryProvider);
  return repo.getMy();
});

/// Persona-scoped variant — used by the /referral mobile screen to
/// surface the apporteur checklist alongside the default freelance
/// one (cached separately so both bars can coexist on the same
/// session). Pass `"referrer"` for the apporteur surface and `null`
/// to fall back to the default persona for the org type.
final profileCompletionByPersonaProvider = FutureProvider.autoDispose
    .family<ProfileCompletionReport, String?>((ref, persona) async {
  final repo = ref.watch(profileCompletionRepositoryProvider);
  return repo.getMy(persona: persona);
});
