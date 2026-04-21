import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/client_profile_repository_impl.dart';
import '../../domain/entities/client_profile.dart';
import '../../domain/repositories/client_profile_repository.dart';

// ---------------------------------------------------------------------------
// Dependency injection
// ---------------------------------------------------------------------------

/// Exposes the concrete [ClientProfileRepository]. Swapped in tests via
/// `ProviderScope.overrides`.
final clientProfileRepositoryProvider =
    Provider<ClientProfileRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ClientProfileRepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Public profile — read side
// ---------------------------------------------------------------------------

/// Fetches the public client profile for a given organization id.
///
/// Uses `autoDispose` so the profile is re-fetched when the screen is
/// revisited, matching the behaviour of the existing public profile
/// screens.
final publicClientProfileProvider = FutureProvider.autoDispose
    .family<ClientProfile, String>((ref, orgId) async {
  final repo = ref.watch(clientProfileRepositoryProvider);
  return repo.getPublicClientProfile(orgId);
});

// ---------------------------------------------------------------------------
// Private profile — mutation side
// ---------------------------------------------------------------------------

/// Tracks the submission state of the private client-profile form.
///
/// `idle`     — nothing in flight, no error, no pending success message.
/// `saving`   — a `PUT` is being awaited; UI disables the save button.
/// `success`  — the last submission succeeded; UI shows a toast and
///              resets to `idle` on the next mutation.
/// `error`    — the last submission failed; UI surfaces [errorMessage].
class ClientProfileFormState {
  const ClientProfileFormState({
    this.status = ClientProfileFormStatus.idle,
    this.errorMessage,
  });

  final ClientProfileFormStatus status;
  final String? errorMessage;

  bool get isSaving => status == ClientProfileFormStatus.saving;
  bool get didSucceed => status == ClientProfileFormStatus.success;
  bool get didFail => status == ClientProfileFormStatus.error;

  ClientProfileFormState copyWith({
    ClientProfileFormStatus? status,
    String? errorMessage,
  }) {
    return ClientProfileFormState(
      status: status ?? this.status,
      errorMessage: errorMessage,
    );
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is ClientProfileFormState &&
        other.status == status &&
        other.errorMessage == errorMessage;
  }

  @override
  int get hashCode => Object.hash(status, errorMessage);
}

enum ClientProfileFormStatus { idle, saving, success, error }

/// State notifier wiring the form to the repository. Keeps the UI
/// declarative — the screen watches this and renders accordingly.
class ClientProfileFormNotifier
    extends StateNotifier<ClientProfileFormState> {
  ClientProfileFormNotifier(this._repository)
      : super(const ClientProfileFormState());

  final ClientProfileRepository _repository;

  /// Submits the form. Invokes [onSuccess] only when the PUT returns
  /// without throwing so callers can trigger a refresh of the read
  /// provider.
  Future<void> submit({
    String? companyName,
    String? clientDescription,
    Future<void> Function()? onSuccess,
  }) async {
    state = const ClientProfileFormState(
      status: ClientProfileFormStatus.saving,
    );
    try {
      await _repository.updateClientProfile(
        companyName: companyName,
        clientDescription: clientDescription,
      );
      state = const ClientProfileFormState(
        status: ClientProfileFormStatus.success,
      );
      if (onSuccess != null) await onSuccess();
    } on Object catch (error) {
      state = ClientProfileFormState(
        status: ClientProfileFormStatus.error,
        errorMessage: _userFriendlyMessage(error),
      );
    }
  }

  /// Resets the notifier back to the idle state — typically called
  /// after the UI has consumed a success/error notification.
  void reset() {
    state = const ClientProfileFormState();
  }

  String _userFriendlyMessage(Object error) {
    final raw = error.toString();
    // Trim the Dart type prefix for more readable error banners.
    return raw.replaceFirst('Exception: ', '');
  }
}

/// Exposes the form state notifier to the screen.
final clientProfileFormProvider = StateNotifierProvider.autoDispose<
    ClientProfileFormNotifier, ClientProfileFormState>((ref) {
  final repository = ref.watch(clientProfileRepositoryProvider);
  return ClientProfileFormNotifier(repository);
});
