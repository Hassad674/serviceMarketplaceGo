import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/job_application_entity.dart';
import '../providers/job_provider.dart';

class CandidateCard extends ConsumerWidget {
  const CandidateCard({
    super.key,
    required this.item,
    required this.jobId,
    required this.onContact,
  });

  final ApplicationWithProfile item;
  final String jobId;
  final void Function(String conversationId) onContact;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profile = item.profile;
    final application = item.application;
    final initials = '${profile.firstName.isNotEmpty ? profile.firstName[0] : ''}${profile.lastName.isNotEmpty ? profile.lastName[0] : ''}'.toUpperCase();

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            CircleAvatar(
              radius: 20,
              backgroundColor: const Color(0xFFFFF1F2),
              child: Text(initials.isNotEmpty ? initials : '?', style: const TextStyle(color: Color(0xFFF43F5E), fontWeight: FontWeight.w600)),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Flexible(
                        child: Text(
                          profile.displayName.isNotEmpty ? profile.displayName : '${profile.firstName} ${profile.lastName}',
                          style: Theme.of(context).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 6),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                        decoration: BoxDecoration(
                          color: profile.role == 'provider' ? const Color(0xFFFFF1F2) : Colors.blue.shade50,
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(
                          profile.role == 'provider' ? 'Freelance' : 'Agence',
                          style: TextStyle(fontSize: 10, fontWeight: FontWeight.w500, color: profile.role == 'provider' ? const Color(0xFFF43F5E) : Colors.blue.shade700),
                        ),
                      ),
                    ],
                  ),
                  if (profile.title.isNotEmpty) ...[
                    const SizedBox(height: 2),
                    Text(profile.title, style: Theme.of(context).textTheme.bodySmall?.copyWith(color: Colors.grey)),
                  ],
                  const SizedBox(height: 6),
                  Text(application.message, maxLines: 3, overflow: TextOverflow.ellipsis, style: Theme.of(context).textTheme.bodySmall),
                ],
              ),
            ),
            const SizedBox(width: 8),
            IconButton(
              icon: const Icon(Icons.send, color: Color(0xFFF43F5E), size: 20),
              tooltip: 'Envoyer un message',
              onPressed: () async {
                final convId = await contactApplicantAction(ref, jobId, application.applicantId);
                if (convId != null) onContact(convId);
              },
            ),
          ],
        ),
      ),
    );
  }
}
