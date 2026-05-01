import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
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

void main() {
  test('returns the proposal_sent palette', () async {
    final l10n = await _loadEn();
    final visuals = systemMessageVisualsFor(
      message: _msg(type: 'proposal_sent'),
      l10n: l10n,
      appColors: null,
    );
    expect(visuals.icon, Icons.description_outlined);
    expect(visuals.color, const Color(0xFFF43F5E));
    expect(visuals.label, l10n.proposalNewMessage);
  });

  test('returns success-green for proposal_completed', () async {
    final l10n = await _loadEn();
    final visuals = systemMessageVisualsFor(
      message: _msg(type: 'proposal_completed'),
      l10n: l10n,
      appColors: null,
    );
    expect(visuals.color, const Color(0xFF22C55E));
    expect(visuals.icon, Icons.task_alt);
  });

  test('returns destructive-red for proposal_declined', () async {
    final l10n = await _loadEn();
    final visuals = systemMessageVisualsFor(
      message: _msg(type: 'proposal_declined'),
      l10n: l10n,
      appColors: null,
    );
    expect(visuals.color, const Color(0xFFEF4444));
  });

  test('falls back to info_outline + content for unknown types', () async {
    final l10n = await _loadEn();
    final visuals = systemMessageVisualsFor(
      message: _msg(type: 'mystery', content: 'something happened'),
      l10n: l10n,
      appColors: null,
    );
    expect(visuals.icon, Icons.info_outline);
    expect(visuals.label, 'something happened');
  });

  test('uses default mutedFg color when AppColors is null', () async {
    final l10n = await _loadEn();
    final visuals = systemMessageVisualsFor(
      message: _msg(type: 'call_ended'),
      l10n: l10n,
      appColors: null,
    );
    expect(visuals.color, const Color(0xFF94A3B8));
  });
}
