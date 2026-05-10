import 'package:flutter/foundation.dart';

/// Severity level for a dashboard "actions à faire" row.
///
/// - [critical]: blocks core flow (KYC restricted, unread > 24h, etc.).
///   Surfaced first, corail/red tone.
/// - [warning]: needs attention soon (Stripe onboarding incomplete,
///   billing missing, profile < 80%, premium < 7d). Amber tone.
/// - [info]: informational nudges (no current use, reserved for
///   future low-priority cards).
enum DashboardActionSeverity { critical, warning, info }

/// One row in the "actions à faire" card on the dashboard.
///
/// Plain Dart immutable holder — no Freezed dependency to keep the
/// dashboard feature build-runner-free. Equality is value-based so
/// Riverpod can deduplicate rebuilds when an aggregator emits the same
/// list twice.
@immutable
class DashboardAction {
  const DashboardAction({
    required this.id,
    required this.severity,
    required this.label,
    required this.route,
    this.detail,
  });

  /// Stable identifier, used for the row's [Key] and for deduplication
  /// upstream. Examples: `kyc_pending`, `profile_incomplete`,
  /// `messages_unread`, `proposals_pending`.
  final String id;

  final DashboardActionSeverity severity;

  /// Pre-localised label shown as the row's primary line.
  final String label;

  /// `go_router` path the row pushes when tapped.
  final String route;

  /// Optional pre-localised secondary line (count, deadline, etc.).
  final String? detail;

  @override
  bool operator ==(Object other) =>
      other is DashboardAction &&
      other.id == id &&
      other.severity == severity &&
      other.label == label &&
      other.route == route &&
      other.detail == detail;

  @override
  int get hashCode => Object.hash(id, severity, label, route, detail);
}
