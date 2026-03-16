import 'package:flutter/material.dart';
import 'package:dio/dio.dart';

class TestScreen extends StatefulWidget {
  const TestScreen({super.key});

  @override
  State<TestScreen> createState() => _TestScreenState();
}

class _TestScreenState extends State<TestScreen> {
  static const baseUrl = String.fromEnvironment(
    'API_URL',
    defaultValue: 'http://10.0.2.2:8083',
  );

  final _dio = Dio(BaseOptions(baseUrl: baseUrl, connectTimeout: const Duration(seconds: 5)));
  final _controller = TextEditingController();

  String _backendStatus = 'checking...';
  String _dbStatus = 'checking...';
  List<Map<String, dynamic>> _words = [];
  bool _adding = false;

  @override
  void initState() {
    super.initState();
    _checkHealth();
    _fetchWords();
  }

  Future<void> _checkHealth() async {
    try {
      final res = await _dio.get('/api/v1/test/health-check');
      setState(() {
        _backendStatus = res.data['backend'] == 'ok' ? 'ok' : 'error';
        _dbStatus = res.data['database'] == 'ok' ? 'ok' : 'error';
      });
    } catch (e) {
      setState(() {
        _backendStatus = 'error';
        _dbStatus = 'error';
      });
    }
  }

  Future<void> _fetchWords() async {
    try {
      final res = await _dio.get('/api/v1/test/words');
      setState(() {
        _words = List<Map<String, dynamic>>.from(res.data['words'] ?? []);
      });
    } catch (_) {}
  }

  Future<void> _addWord() async {
    if (_controller.text.trim().isEmpty) return;
    setState(() => _adding = true);
    try {
      await _dio.post('/api/v1/test/words', data: {'word': _controller.text.trim()});
      _controller.clear();
      await _fetchWords();
    } catch (_) {}
    setState(() => _adding = false);
  }

  Widget _statusBadge(String status) {
    Color bg;
    Color fg;
    String text;

    switch (status) {
      case 'ok':
        bg = const Color(0xFFDCFCE7);
        fg = const Color(0xFF16A34A);
        text = '● OK';
        break;
      case 'error':
        bg = const Color(0xFFFEE2E2);
        fg = const Color(0xFFDC2626);
        text = '● ERROR';
        break;
      default:
        bg = const Color(0xFFF1F5F9);
        fg = const Color(0xFF64748B);
        text = 'Checking...';
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(20),
      ),
      child: Text(text, style: TextStyle(color: fg, fontSize: 12, fontWeight: FontWeight.w600)),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Test Backend & DB')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Status
            Row(
              children: [
                const Text('Backend: ', style: TextStyle(fontSize: 14, color: Color(0xFF64748B))),
                _statusBadge(_backendStatus),
                const SizedBox(width: 24),
                const Text('Database: ', style: TextStyle(fontSize: 14, color: Color(0xFF64748B))),
                _statusBadge(_dbStatus),
              ],
            ),
            const SizedBox(height: 24),

            // Input
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _controller,
                    decoration: const InputDecoration(
                      hintText: 'Tapez un mot...',
                      isDense: true,
                    ),
                    onSubmitted: (_) => _addWord(),
                  ),
                ),
                const SizedBox(width: 12),
                ElevatedButton(
                  onPressed: _adding ? null : _addWord,
                  child: Text(_adding ? '...' : 'Ajouter'),
                ),
              ],
            ),
            const SizedBox(height: 24),

            // Words
            if (_words.isNotEmpty) ...[
              Text('Mots enregistrés (${_words.length})', style: const TextStyle(fontSize: 14, color: Color(0xFF64748B))),
              const SizedBox(height: 12),
              Expanded(
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: _words.map((w) => Chip(
                    label: Text(w['word'] ?? ''),
                    backgroundColor: const Color(0xFFFFF1F2),
                    labelStyle: const TextStyle(color: Color(0xFFF43F5E)),
                  )).toList(),
                ),
              ),
            ],

            const Spacer(),
            Text('API: $baseUrl', style: const TextStyle(fontSize: 11, color: Color(0xFF94A3B8))),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }
}
