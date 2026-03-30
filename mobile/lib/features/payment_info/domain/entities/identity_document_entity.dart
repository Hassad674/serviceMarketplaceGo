/// Domain entity for an identity verification document.
class IdentityDocument {
  final String id;
  final String userId;
  final String category;
  final String documentType;
  final String side;
  final String fileUrl;
  final String status;
  final String rejectionReason;
  final DateTime createdAt;
  final DateTime updatedAt;

  const IdentityDocument({
    required this.id,
    required this.userId,
    this.category = 'identity',
    required this.documentType,
    this.side = 'front',
    this.fileUrl = '',
    this.status = 'pending',
    this.rejectionReason = '',
    required this.createdAt,
    required this.updatedAt,
  });

  factory IdentityDocument.fromJson(Map<String, dynamic> json) {
    return IdentityDocument(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      category: json['category'] as String? ?? 'identity',
      documentType: json['document_type'] as String,
      side: json['side'] as String? ?? 'front',
      fileUrl: json['file_url'] as String? ?? '',
      status: json['status'] as String? ?? 'pending',
      rejectionReason: json['rejection_reason'] as String? ?? '',
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  bool get isPending => status == 'pending';
  bool get isVerified => status == 'verified';
  bool get isRejected => status == 'rejected';
}
