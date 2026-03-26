import 'types/conversation.dart';

/// Mock conversations matching the web version's mock data.
const List<Conversation> mockConversations = [
  Conversation(
    id: '1',
    name: 'Jean Dupont',
    role: 'freelancer',
    lastMessage: "I'll send the mockups tomorrow",
    lastMessageAt: '10:30',
    unread: 2,
    online: true,
  ),
  Conversation(
    id: '2',
    name: 'AgenceWeb Paris',
    role: 'agency',
    lastMessage: 'The project is on track',
    lastMessageAt: 'Yesterday',
    unread: 0,
    online: false,
  ),
  Conversation(
    id: '3',
    name: 'TechCorp',
    role: 'enterprise',
    lastMessage: null,
    lastMessageAt: null,
    unread: 0,
    online: true,
  ),
  Conversation(
    id: '4',
    name: 'Marie Laurent',
    role: 'freelancer',
    lastMessage: 'Can we schedule a call?',
    lastMessageAt: 'Mon',
    unread: 1,
    online: false,
  ),
  Conversation(
    id: '5',
    name: 'Studio Pixel',
    role: 'agency',
    lastMessage: 'Invoice sent for Q1',
    lastMessageAt: 'Mar 20',
    unread: 0,
    online: true,
  ),
];

/// Mock messages matching the web version's mock data.
const List<Message> mockMessages = [
  Message(
    id: 'm1',
    conversationId: '1',
    senderId: 'other',
    content: 'Hi! I wanted to discuss the project timeline.',
    sentAt: '09:15',
    isOwn: false,
  ),
  Message(
    id: 'm2',
    conversationId: '1',
    senderId: 'me',
    content: "Sure, I'm available this afternoon. What works for you?",
    sentAt: '09:22',
    isOwn: true,
  ),
  Message(
    id: 'm3',
    conversationId: '1',
    senderId: 'other',
    content: "3pm would be great. I'll prepare the mockups beforehand.",
    sentAt: '09:30',
    isOwn: false,
  ),
  Message(
    id: 'm4',
    conversationId: '1',
    senderId: 'me',
    content: "Perfect, I'll send you the brief document before then.",
    sentAt: '09:45',
    isOwn: true,
  ),
  Message(
    id: 'm5',
    conversationId: '1',
    senderId: 'other',
    content: "I'll send the mockups tomorrow",
    sentAt: '10:30',
    isOwn: false,
  ),
];
