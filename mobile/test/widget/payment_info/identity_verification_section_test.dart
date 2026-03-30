import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/identity_document_entity.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/providers/identity_document_provider.dart';
import 'package:marketplace_mobile/features/payment_info/presentation/widgets/identity_verification_section.dart';

import 'test_helpers.dart';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

IdentityDocument _buildDoc({
  String status = 'pending',
  String documentType = 'passport',
  String rejectionReason = '',
}) {
  return IdentityDocument(
    id: 'doc-1',
    userId: 'user-1',
    category: 'identity',
    documentType: documentType,
    side: 'single',
    fileUrl: 'https://example.com/doc.jpg',
    status: status,
    rejectionReason: rejectionReason,
    createdAt: DateTime(2026, 3, 1),
    updatedAt: DateTime(2026, 3, 2),
  );
}

// ---------------------------------------------------------------------------
// Fakes to prevent real FlutterSecureStorage / Dio initialization
// ---------------------------------------------------------------------------

class _FakeStorage extends Fake implements SecureStorageService {
  @override
  Future<String?> getAccessToken() async => null;

  @override
  Future<String?> getRefreshToken() async => null;

  @override
  Future<bool> hasTokens() async => false;

  @override
  Future<void> saveTokens(String access, String refresh) async {}

  @override
  Future<void> clearTokens() async {}

  @override
  Future<void> clearAll() async {}

  @override
  Future<void> saveUser(Map<String, dynamic> user) async {}

  @override
  Future<Map<String, dynamic>?> getUser() async => null;
}

class _FakeApiClient extends ApiClient {
  _FakeApiClient() : super(storage: _FakeStorage());
}

List<Override> _docOverrides(
  Future<List<IdentityDocument>> Function(
    FutureProviderRef<List<IdentityDocument>>,
  ) builder,
) {
  return [
    secureStorageProvider.overrideWithValue(_FakeStorage()),
    apiClientProvider.overrideWithValue(_FakeApiClient()),
    identityDocumentsProvider.overrideWith(builder),
  ];
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('IdentityVerificationSection', () {
    testWidgets('shows upload prompt when no documents', (
      WidgetTester tester,
    ) async {
      await tester.pumpWidget(
        buildTestableWidget(
          const IdentityVerificationSection(),
          overrides: _docOverrides(
            (ref) => Future.value(<IdentityDocument>[]),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Upload prompt text
      expect(find.text('Upload identity document'), findsOneWidget);
      expect(
        find.text('Upload a clear photo of your document'),
        findsOneWidget,
      );
      expect(find.byIcon(Icons.upload_file), findsOneWidget);
    });

    testWidgets('shows pending status when document is pending', (
      WidgetTester tester,
    ) async {
      final pendingDoc = _buildDoc(status: 'pending');

      await tester.pumpWidget(
        buildTestableWidget(
          const IdentityVerificationSection(),
          overrides: _docOverrides(
            (ref) => Future.value([pendingDoc]),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Pending banner
      expect(
        find.text('Your document is being reviewed'),
        findsOneWidget,
      );
      expect(find.byIcon(Icons.schedule), findsOneWidget);
      // Document type label
      expect(find.text('Passport'), findsOneWidget);
      // Replace button
      expect(find.text('Replace'), findsOneWidget);
    });

    testWidgets('shows verified status when document is verified', (
      WidgetTester tester,
    ) async {
      final verifiedDoc = _buildDoc(status: 'verified');

      await tester.pumpWidget(
        buildTestableWidget(
          const IdentityVerificationSection(),
          overrides: _docOverrides(
            (ref) => Future.value([verifiedDoc]),
          ),
        ),
      );

      await tester.pumpAndSettle();

      // Verified banner
      expect(
        find.text('Your identity has been verified'),
        findsOneWidget,
      );
      expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
    });

    testWidgets(
      'shows rejected status with reason when document is rejected',
      (WidgetTester tester) async {
        final rejectedDoc = _buildDoc(
          status: 'rejected',
          rejectionReason: 'Document is blurry',
        );

        await tester.pumpWidget(
          buildTestableWidget(
            const IdentityVerificationSection(),
            overrides: _docOverrides(
              (ref) => Future.value([rejectedDoc]),
            ),
          ),
        );

        await tester.pumpAndSettle();

        // Rejected banner
        expect(
          find.text('Your document was rejected'),
          findsOneWidget,
        );
        // Rejection reason displayed
        expect(find.text('Document is blurry'), findsOneWidget);
        expect(find.byIcon(Icons.warning_amber), findsOneWidget);

        // Re-upload button should appear
        expect(find.text('Upload identity document'), findsOneWidget);
      },
    );

    testWidgets(
      'shows rejected status without reason when reason is empty',
      (WidgetTester tester) async {
        final rejectedDoc = _buildDoc(
          status: 'rejected',
          rejectionReason: '',
        );

        await tester.pumpWidget(
          buildTestableWidget(
            const IdentityVerificationSection(),
            overrides: _docOverrides(
              (ref) => Future.value([rejectedDoc]),
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(
          find.text('Your document was rejected'),
          findsOneWidget,
        );
        // No reason text should be present (only the banner text)
      },
    );

    testWidgets('shows loading indicator when fetching documents', (
      WidgetTester tester,
    ) async {
      // Use a Completer so the future never resolves but leaves no
      // pending timers when the test framework tears down widgets.
      final completer = Completer<List<IdentityDocument>>();

      await tester.pumpWidget(
        buildTestableWidget(
          const IdentityVerificationSection(),
          overrides: _docOverrides(
            (ref) => completer.future,
          ),
        ),
      );

      await tester.pump();

      // Section title should still be visible
      expect(find.text('Identity verification'), findsOneWidget);
      // Loading indicator inside the section
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets(
      'shows ID card document type label for id_card',
      (WidgetTester tester) async {
        final idCardDoc = _buildDoc(
          status: 'pending',
          documentType: 'id_card',
        );

        await tester.pumpWidget(
          buildTestableWidget(
            const IdentityVerificationSection(),
            overrides: _docOverrides(
              (ref) => Future.value([idCardDoc]),
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(find.text('ID Card'), findsOneWidget);
      },
    );

    testWidgets(
      'shows driving license label for driving_license',
      (WidgetTester tester) async {
        final dlDoc = _buildDoc(
          status: 'verified',
          documentType: 'driving_license',
        );

        await tester.pumpWidget(
          buildTestableWidget(
            const IdentityVerificationSection(),
            overrides: _docOverrides(
              (ref) => Future.value([dlDoc]),
            ),
          ),
        );

        await tester.pumpAndSettle();

        expect(find.text('Driving License'), findsOneWidget);
      },
    );
  });
}
