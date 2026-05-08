import 'package:freezed_annotation/freezed_annotation.dart';

part 'security_event.freezed.dart';

/// Coarse classification of the device behind a user-agent string.
///
/// Mirrors the backend's `access_kind` values (see
/// `internal/app/security/parser.go`). The UI uses it to pick an
/// icon; an unknown bucket falls back to a neutral question mark.
enum SecurityAccessKind {
  desktop,
  mobile,
  tablet,
  unknown,
}

/// Maps a wire `access_kind` string to the typed enum. Falls back to
/// [SecurityAccessKind.unknown] on novel values so an API addition
/// never crashes the screen.
SecurityAccessKind securityAccessKindFromString(String? raw) {
  switch (raw) {
    case 'desktop':
      return SecurityAccessKind.desktop;
    case 'mobile':
      return SecurityAccessKind.mobile;
    case 'tablet':
      return SecurityAccessKind.tablet;
    default:
      return SecurityAccessKind.unknown;
  }
}

/// One authentication-related audit row, projected for the
/// "Activité récente" tab.
///
/// `userAgentSummary` is the short label parsed server-side
/// ("Ordinateur (Chrome 120)") — the mobile app does not re-parse the
/// raw user-agent locally, the backend is the single source of truth.
@freezed
class SecurityEvent with _$SecurityEvent {
  const factory SecurityEvent({
    required String id,
    required String action,
    required String userAgentSummary,
    required SecurityAccessKind accessKind,
    required DateTime createdAt,
    String? ipAddress,
    String? countryHint,
  }) = _SecurityEvent;
}
