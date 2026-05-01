// ListView perf bench for the chat / conversations lists (cat C).
//
// PERF-M-06: ListView.builder without stable keys triggers a full
// subtree rebuild whenever an item is inserted at the top — every
// existing item is shifted by index and Flutter's element tree
// re-creates each State because the slot identity changed. With
// `findChildIndexCallback` + `ValueKey(item.id)` the existing
// elements are reused.
//
// These two tests exercise the exact contract on a tiny stand-in
// widget: counting build invocations on a child whose ValueKey is
// stable across the index shift.

import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

class _CountingTile extends StatefulWidget {
  const _CountingTile({super.key, required this.id});

  final String id;

  @override
  State<_CountingTile> createState() => _CountingTileState();
}

class _CountingTileState extends State<_CountingTile> {
  static int totalBuilds = 0;
  static int totalCreations = 0;

  static void reset() {
    totalBuilds = 0;
    totalCreations = 0;
  }

  @override
  void initState() {
    super.initState();
    totalCreations++;
  }

  @override
  Widget build(BuildContext context) {
    totalBuilds++;
    return SizedBox(
      height: 80,
      child: Text('item ${widget.id}'),
    );
  }
}

Widget _wrap(Widget child) {
  return MaterialApp(
    home: Scaffold(body: child),
  );
}

ListView _buildKeyed(List<String> ids) {
  return ListView.builder(
    itemCount: ids.length,
    findChildIndexCallback: (key) {
      if (key is! ValueKey<String>) return null;
      final i = ids.indexOf(key.value);
      return i >= 0 ? i : null;
    },
    addAutomaticKeepAlives: false,
    itemBuilder: (context, index) => _CountingTile(
      key: ValueKey<String>(ids[index]),
      id: ids[index],
    ),
  );
}

ListView _buildUnkeyed(List<String> ids) {
  return ListView.builder(
    itemCount: ids.length,
    itemBuilder: (context, index) => _CountingTile(id: ids[index]),
  );
}

void main() {
  group('ListView stable-key perf contract', () {
    testWidgets('keyed list preserves per-item State across reorder',
        (tester) async {
      // The PRIMARY win of stable ValueKey + findChildIndexCallback
      // is that the State of an existing item survives a list
      // reorder — its scroll position, expansion flag, animation
      // controller, etc. We assert that by mutating internal
      // counters on each tile and proving they survive an insert.
      _CountingTileState.reset();
      var ids = ['a', 'b', 'c', 'd', 'e'];
      await tester.pumpWidget(_wrap(_buildKeyed(ids)));
      final initialCreations = _CountingTileState.totalCreations;
      expect(initialCreations, equals(5));

      // Prepend `z`, push everyone down by one index.
      ids = [...ids]..insert(0, 'z');
      await tester.pumpWidget(_wrap(_buildKeyed(ids)));

      // We must have at most ONE new creation: the prepended `z`.
      // The 5 originals' State instances are reused.
      final delta = _CountingTileState.totalCreations - initialCreations;
      expect(
        delta,
        equals(1),
        reason: 'Stable ValueKey + findChildIndexCallback should '
            'reuse the 5 existing States and create exactly 1 new '
            'State for the prepended item. Got $delta',
      );
    });

    testWidgets('keyed list does not re-create State on no-op pump',
        (tester) async {
      _CountingTileState.reset();
      final ids = ['a', 'b', 'c'];
      await tester.pumpWidget(_wrap(_buildKeyed(ids)));
      final firstCreations = _CountingTileState.totalCreations;
      expect(firstCreations, equals(3));

      // Same id list — pumping again must not bump State count.
      await tester.pumpWidget(_wrap(_buildKeyed(List<String>.from(ids))));
      expect(
        _CountingTileState.totalCreations,
        equals(firstCreations),
        reason: 'A no-op pump must not re-create any tile State',
      );
    });
  });

  group('chat_screen.dart source contract', () {
    final src = File(
      'lib/features/messaging/presentation/screens/chat_screen.dart',
    ).readAsStringSync();

    test('_MessagesListView caps cacheExtent + disables KeepAlive', () {
      expect(src.contains('cacheExtent: 800'), isTrue);
      expect(src.contains('addAutomaticKeepAlives: false'), isTrue);
      expect(
        src.contains('findChildIndexCallback'),
        isTrue,
        reason: 'Stable id-based index lookup is required to make '
            'ValueKey reuse the existing element on insert',
      );
    });

    test('each bubble is wrapped in a RepaintBoundary with ValueKey', () {
      expect(src.contains('RepaintBoundary('), isTrue);
      expect(src.contains('ValueKey<String>(message.id)'), isTrue);
    });
  });

  group('messaging_screen.dart source contract', () {
    final src = File(
      'lib/features/messaging/presentation/screens/messaging_screen.dart',
    ).readAsStringSync();

    test('conversations list uses stable keys + cacheExtent', () {
      expect(src.contains('findChildIndexCallback'), isTrue);
      expect(src.contains('cacheExtent: 600'), isTrue);
      expect(src.contains('addAutomaticKeepAlives: false'), isTrue);
    });

    test('each conversation tile is RepaintBoundary-wrapped', () {
      expect(src.contains('RepaintBoundary('), isTrue);
      expect(src.contains('ValueKey<String>(conversation.id)'), isTrue);
    });
  });
}
