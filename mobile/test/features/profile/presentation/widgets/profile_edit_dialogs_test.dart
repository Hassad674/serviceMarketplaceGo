import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/features/profile/presentation/widgets/profile_edit_dialogs.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

class _RecordingApiClient extends ApiClient {
  _RecordingApiClient() : super(storage: _NoopStorage());

  final List<({String method, String path, Object? data})> calls = [];

  @override
  Future<Response<T>> put<T>(String path, {dynamic data}) async {
    calls.add((method: 'PUT', path: path, data: data));
    return Response<T>(
      requestOptions: RequestOptions(path: path),
      statusCode: 200,
    );
  }

  @override
  Future<Response<T>> delete<T>(String path) async {
    calls.add((method: 'DELETE', path: path, data: null));
    return Response<T>(
      requestOptions: RequestOptions(path: path),
      statusCode: 200,
    );
  }
}

class _NoopStorage extends SecureStorageService {}

Widget _wrap(Widget child, ApiClient client) => ProviderScope(
      overrides: [
        apiClientProvider.overrideWithValue(client),
      ],
      child: MaterialApp(
        localizationsDelegates: const [
          AppLocalizations.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        supportedLocales: const [Locale('en'), Locale('fr')],
        locale: const Locale('en'),
        home: Scaffold(body: child),
      ),
    );

void main() {
  testWidgets('openProfileAboutEditor opens a bottom sheet with TextField',
      (tester) async {
    final api = _RecordingApiClient();
    late BuildContext capturedContext;
    late WidgetRef capturedRef;
    await tester.pumpWidget(
      _wrap(
        Consumer(
          builder: (context, ref, _) {
            capturedContext = context;
            capturedRef = ref;
            return const SizedBox();
          },
        ),
        api,
      ),
    );
    openProfileAboutEditor(capturedContext, capturedRef, 'hello');
    await tester.pumpAndSettle();

    expect(find.byType(TextField), findsOneWidget);
    expect(find.text('hello'), findsOneWidget);
  });

  testWidgets('openProfileAboutEditor save → calls PUT /api/v1/profile',
      (tester) async {
    final api = _RecordingApiClient();
    late BuildContext capturedContext;
    late WidgetRef capturedRef;
    await tester.pumpWidget(
      _wrap(
        Consumer(
          builder: (context, ref, _) {
            capturedContext = context;
            capturedRef = ref;
            return const SizedBox();
          },
        ),
        api,
      ),
    );
    openProfileAboutEditor(capturedContext, capturedRef, 'before');
    await tester.pumpAndSettle();

    // Edit text and tap save.
    await tester.enterText(find.byType(TextField), 'after edit');
    await tester.tap(find.byType(ElevatedButton));
    await tester.pumpAndSettle();

    expect(api.calls.length, 1);
    expect(api.calls.first.method, 'PUT');
    expect(api.calls.first.path, '/api/v1/profile');
    expect(api.calls.first.data, {'about': 'after edit'});
  });

  testWidgets(
      'confirmDeleteProfileVideo opens an AlertDialog with cancel/remove',
      (tester) async {
    final api = _RecordingApiClient();
    late BuildContext capturedContext;
    late WidgetRef capturedRef;
    await tester.pumpWidget(
      _wrap(
        Consumer(
          builder: (context, ref, _) {
            capturedContext = context;
            capturedRef = ref;
            return const SizedBox();
          },
        ),
        api,
      ),
    );
    confirmDeleteProfileVideo(capturedContext, capturedRef);
    await tester.pumpAndSettle();

    expect(find.byType(AlertDialog), findsOneWidget);
    final cancelBtn = find.byType(TextButton).first;
    await tester.tap(cancelBtn);
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsNothing);
    expect(api.calls, isEmpty);
  });

  testWidgets('openProfilePhotoUpload opens an upload bottom sheet',
      (tester) async {
    final api = _RecordingApiClient();
    late BuildContext capturedContext;
    late WidgetRef capturedRef;
    await tester.pumpWidget(
      _wrap(
        Consumer(
          builder: (context, ref, _) {
            capturedContext = context;
            capturedRef = ref;
            return const SizedBox();
          },
        ),
        api,
      ),
    );
    openProfilePhotoUpload(capturedContext, capturedRef);
    await tester.pumpAndSettle();
    // Upload bottom sheet renders some interactive content — we just verify
    // the function does not throw and that something is opened.
    expect(find.byType(BottomSheet), findsOneWidget);
  });

  testWidgets('openProfileVideoUpload opens an upload bottom sheet',
      (tester) async {
    final api = _RecordingApiClient();
    late BuildContext capturedContext;
    late WidgetRef capturedRef;
    await tester.pumpWidget(
      _wrap(
        Consumer(
          builder: (context, ref, _) {
            capturedContext = context;
            capturedRef = ref;
            return const SizedBox();
          },
        ),
        api,
      ),
    );
    openProfileVideoUpload(capturedContext, capturedRef);
    await tester.pumpAndSettle();
    expect(find.byType(BottomSheet), findsOneWidget);
  });

  testWidgets('confirmDeleteProfileVideo remove → calls DELETE /api/v1/upload/video',
      (tester) async {
    final api = _RecordingApiClient();
    late BuildContext capturedContext;
    late WidgetRef capturedRef;
    await tester.pumpWidget(
      _wrap(
        Consumer(
          builder: (context, ref, _) {
            capturedContext = context;
            capturedRef = ref;
            return const SizedBox();
          },
        ),
        api,
      ),
    );
    confirmDeleteProfileVideo(capturedContext, capturedRef);
    await tester.pumpAndSettle();

    // Remove button is the second TextButton (after cancel).
    final removeBtn = find.byType(TextButton).last;
    await tester.tap(removeBtn);
    await tester.pumpAndSettle();

    expect(api.calls.length, 1);
    expect(api.calls.first.method, 'DELETE');
    expect(api.calls.first.path, '/api/v1/upload/video');
  });
}
