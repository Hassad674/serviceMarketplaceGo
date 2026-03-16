import '../entities/conversation.dart';
import '../entities/message.dart';

abstract class MessagingRepository {
  Future<List<Conversation>> getConversations();
  Future<List<Message>> getMessages(String conversationId, {int page, int limit});
  Future<Message> sendMessage({required String conversationId, required String content, String? type});
  Future<void> markAsRead(String conversationId);
}
