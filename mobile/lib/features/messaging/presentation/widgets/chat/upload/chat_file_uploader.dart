import 'dart:io';

import 'package:dio/dio.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';

import '../../../../../../core/utils/mime_type_helper.dart';
import '../../../../domain/repositories/messaging_repository.dart';

/// Output of a successful file upload — values needed to push a file
/// message into the conversation.
class ChatFileUploadResult {
  ChatFileUploadResult({
    required this.filename,
    required this.contentType,
    required this.fileKey,
    required this.fileUrl,
    required this.fileSize,
  });

  final String filename;
  final String contentType;
  final String fileKey;
  final String fileUrl;
  final int fileSize;
}

/// Picks a single file from the device and uploads it to the
/// presigned URL returned by [repo].
///
/// Returns `null` when the user cancels the picker. Throws a
/// `ChatFileUploadException` when the upload fails — the caller
/// surfaces a snackbar.
class ChatFileUploader {
  ChatFileUploader(this._repo);

  final MessagingRepository _repo;

  Future<ChatFileUploadResult?> pickAndUploadFile() async {
    final result = await FilePicker.platform.pickFiles(withData: true);
    if (result == null || result.files.isEmpty) return null;

    final file = result.files.first;
    if (file.name.isEmpty) return null;

    final contentType = guessContentType(file.name);

    final uploadInfo = await _repo.getUploadUrl(
      filename: file.name,
      contentType: contentType,
    );

    final Uint8List fileBytes;
    if (file.bytes != null && file.bytes!.isNotEmpty) {
      fileBytes = file.bytes!;
    } else if (file.path != null) {
      fileBytes = await File(file.path!).readAsBytes();
    } else {
      throw const ChatFileUploadException('Cannot read file: no bytes and no path');
    }

    final uploadDio = Dio(
      BaseOptions(
        connectTimeout: const Duration(seconds: 30),
        sendTimeout: const Duration(seconds: 120),
        receiveTimeout: const Duration(seconds: 30),
      ),
    );

    await uploadDio.put<void>(
      uploadInfo.uploadUrl,
      data: Stream.fromIterable([fileBytes]),
      options: Options(
        contentType: contentType,
        headers: {
          Headers.contentLengthHeader: fileBytes.length,
        },
      ),
    );

    final resolvedUrl = uploadInfo.publicUrl.isNotEmpty
        ? uploadInfo.publicUrl
        : uploadInfo.uploadUrl.split('?').first;

    return ChatFileUploadResult(
      filename: file.name,
      contentType: contentType,
      fileKey: uploadInfo.fileKey,
      fileUrl: resolvedUrl,
      fileSize: file.size,
    );
  }

  Future<ChatVoiceUploadResult?> uploadVoiceFile(
    String path,
    int durationSeconds,
  ) async {
    final file = File(path);
    if (!file.existsSync()) return null;
    final fileBytes = await file.readAsBytes();
    final fileSize = fileBytes.length;
    final ext = path.split('.').last.toLowerCase();
    final contentType = ext == 'm4a' ? 'audio/mp4' : 'audio/$ext';
    final filename = 'voice-${DateTime.now().millisecondsSinceEpoch}.$ext';

    final uploadInfo = await _repo.getUploadUrl(
      filename: filename,
      contentType: contentType,
    );

    final uploadDio = Dio(
      BaseOptions(
        connectTimeout: const Duration(seconds: 30),
        sendTimeout: const Duration(seconds: 60),
        receiveTimeout: const Duration(seconds: 30),
      ),
    );

    await uploadDio.put<void>(
      uploadInfo.uploadUrl,
      data: Stream.fromIterable([fileBytes]),
      options: Options(
        contentType: contentType,
        headers: {Headers.contentLengthHeader: fileSize},
      ),
    );

    final resolvedUrl = uploadInfo.publicUrl.isNotEmpty
        ? uploadInfo.publicUrl
        : uploadInfo.uploadUrl.split('?').first;

    // Clean up temporary recording file.
    file.delete().catchError((_) => file);

    return ChatVoiceUploadResult(
      voiceUrl: resolvedUrl,
      durationSeconds: durationSeconds.toDouble(),
      size: fileSize,
      mimeType: contentType,
    );
  }
}

class ChatVoiceUploadResult {
  ChatVoiceUploadResult({
    required this.voiceUrl,
    required this.durationSeconds,
    required this.size,
    required this.mimeType,
  });

  final String voiceUrl;
  final double durationSeconds;
  final int size;
  final String mimeType;
}

class ChatFileUploadException implements Exception {
  const ChatFileUploadException(this.message);
  final String message;

  @override
  String toString() => message;
}
