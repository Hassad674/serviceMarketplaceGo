import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../../../l10n/app_localizations.dart';
import '../../../../../call/domain/entities/call_entity.dart';
import '../../../../../call/presentation/providers/call_provider.dart';
import '../../../../../call/presentation/screens/call_screen.dart';

/// Wraps audio/video call initiation logic for the chat screen.
///
/// Translates call provider error codes into localized messages and
/// pushes the dedicated call screen on success.
class ChatCallHandlers {
  ChatCallHandlers({
    required this.ref,
    required this.context,
    required this.conversationId,
  });

  final WidgetRef ref;
  final BuildContext context;
  final String conversationId;

  Future<void> startAudioCall(dynamic conversation) async {
    await _initiate(conversation, CallType.audio);
  }

  Future<void> startVideoCall(dynamic conversation) async {
    await _initiate(conversation, CallType.video);
  }

  Future<void> _initiate(dynamic conversation, CallType callType) async {
    if (conversation == null) return;
    final callNotifier = ref.read(callProvider.notifier);
    await callNotifier.initiateCall(
      conversationId: conversationId,
      recipientId: conversation.otherUserId,
      callType: callType,
    );
    if (!context.mounted) return;

    final callState = ref.read(callProvider);
    if (callState.status == CallStatus.ringingOutgoing) {
      Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => CallScreen(
            recipientName: conversation.otherOrgName ?? '',
            callType: callType,
          ),
        ),
      );
    } else if (callState.errorMessage != null) {
      final l10n = AppLocalizations.of(context)!;
      final msg = _errorToMessage(l10n, callState.errorMessage!);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
      callNotifier.clearError();
    }
  }

  String _errorToMessage(AppLocalizations l10n, String code) {
    switch (code) {
      case 'recipient_offline':
        return l10n.callRecipientOffline;
      case 'user_busy':
        return l10n.callUserBusy;
      default:
        return l10n.callFailed;
    }
  }
}
