import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/messaging/domain/entities/message_entity.dart';
import 'package:marketplace_mobile/features/messaging/presentation/widgets/chat/bubbles/system_message_palette.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

MessageEntity _msg({required String type, String content = ''}) {
  return MessageEntity(
    id: 'm',
    conversationId: 'conv',
    senderId: 'user',
    type: type,
    content: content,
    seq: 1,
    createdAt: DateTime.now().toIso8601String(),
  );
}

Future<AppLocalizations> _loadEn() async {
  return AppLocalizations.delegate.load(const Locale('en'));
}

/// Resolves the visuals struct under a real Soleil [Theme] so the helper
/// can read `colorScheme` + the `AppColors` extension.
Future<SystemMessageVisuals> _resolve(
  WidgetTester tester, {
  required MessageEntity message,
  required AppLocalizations l10n,
}) async {
  late SystemMessageVisuals visuals;
  await tester.pumpWidget(
    MaterialApp(
      theme: AppTheme.light,
      home: Builder(
        builder: (context) {
          visuals = systemMessageVisualsFor(
            context: context,
            message: message,
            l10n: l10n,
          );
          return const SizedBox.shrink();
        },
      ),
    ),
  );
  return visuals;
}

void main() {
  testWidgets('returns the proposal_sent palette', (tester) async {
    final l10n = await _loadEn();
    final visuals = await _resolve(
      tester,
      message: _msg(type: 'proposal_sent'),
      l10n: l10n,
    );
    expect(visuals.icon, Icons.description_outlined);
    expect(visuals.color, AppTheme.light.colorScheme.primary);
    expect(visuals.label, l10n.proposalNewMessage);
  });

  testWidgets('returns success for proposal_completed', (tester) async {
    final l10n = await _loadEn();
    final visuals = await _resolve(
      tester,
      message: _msg(type: 'proposal_completed'),
      l10n: l10n,
    );
    expect(visuals.color, AppTheme.light.extension<AppColors>()!.success);
    expect(visuals.icon, Icons.task_alt);
  });

  testWidgets('returns destructive (error) for proposal_declined',
      (tester) async {
    final l10n = await _loadEn();
    final visuals = await _resolve(
      tester,
      message: _msg(type: 'proposal_declined'),
      l10n: l10n,
    );
    expect(visuals.color, AppTheme.light.colorScheme.error);
  });

  testWidgets('falls back to info_outline + content for unknown types',
      (tester) async {
    final l10n = await _loadEn();
    final visuals = await _resolve(
      tester,
      message: _msg(type: 'mystery', content: 'something happened'),
      l10n: l10n,
    );
    expect(visuals.icon, Icons.info_outline);
    expect(visuals.label, 'something happened');
  });

  testWidgets('uses the muted foreground for call_ended', (tester) async {
    final l10n = await _loadEn();
    final visuals = await _resolve(
      tester,
      message: _msg(type: 'call_ended'),
      l10n: l10n,
    );
    // mutedForeground falls back to onSurfaceVariant — compare against
    // whichever the live theme exposes (Soleil v2 uses tabac).
    expect(
      visuals.color,
      AppTheme.light.extension<AppColors>()?.mutedForeground ??
          AppTheme.light.colorScheme.onSurfaceVariant,
    );
  });
}
