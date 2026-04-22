import 'package:freezed_annotation/freezed_annotation.dart';

part 'cycle_preview.freezed.dart';

/// Invoice preview returned by `GET /subscriptions/me/cycle-preview`.
///
/// [amountDueCents] > 0 means the user is charged that amount today
/// (upgrade path). A zero amount means no immediate charge — the cycle
/// switch is scheduled at the end of the current period (downgrade).
///
/// [prorateImmediately] mirrors the backend flag so the UI can switch
/// copy ("You will be billed ..." vs "No charge today, switch on ...").
@freezed
class CyclePreview with _$CyclePreview {
  const factory CyclePreview({
    required int amountDueCents,
    required String currency,
    required DateTime periodStart,
    required DateTime periodEnd,
    required bool prorateImmediately,
  }) = _CyclePreview;
}
