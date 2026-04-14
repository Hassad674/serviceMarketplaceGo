import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../data/city_search_service.dart';

/// Inline city autocomplete for the mobile location editor. Opens
/// a dropdown of BAN (France) or Photon (international) matches as
/// the user types, with debounce + in-flight cancellation. The
/// user MUST pick a row — bare text is never persisted, matching
/// the web behavior.
class CitySearchField extends StatefulWidget {
  const CitySearchField({
    super.key,
    required this.selection,
    required this.countryCode,
    required this.onSelected,
    CitySearchService? service,
  }) : _service = service;

  final CitySearchResult? selection;
  final String countryCode;
  final ValueChanged<CitySearchResult?> onSelected;
  final CitySearchService? _service;

  @override
  State<CitySearchField> createState() => _CitySearchFieldState();
}

class _CitySearchFieldState extends State<CitySearchField> {
  static const _debounce = Duration(milliseconds: 250);

  late final CitySearchService _service;
  late final TextEditingController _controller;
  Timer? _debounceHandle;
  CancelToken? _cancelToken;
  List<CitySearchResult> _results = const [];
  bool _isLoading = false;
  bool _focused = false;
  final FocusNode _focusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    _service = widget._service ?? CitySearchService();
    _controller = TextEditingController(text: widget.selection?.city ?? '');
    _focusNode.addListener(_onFocusChange);
  }

  @override
  void didUpdateWidget(covariant CitySearchField oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.selection != widget.selection) {
      final next = widget.selection?.city ?? '';
      if (_controller.text != next) _controller.text = next;
    }
    if (oldWidget.countryCode != widget.countryCode) {
      _results = const [];
    }
  }

  @override
  void dispose() {
    _debounceHandle?.cancel();
    _cancelToken?.cancel();
    _focusNode.removeListener(_onFocusChange);
    _focusNode.dispose();
    _controller.dispose();
    super.dispose();
  }

  void _onFocusChange() {
    setState(() => _focused = _focusNode.hasFocus);
    if (!_focusNode.hasFocus) {
      // Restore the canonical label when the user taps away without
      // picking anything — a bare search term is never savable.
      final fallback = widget.selection?.city ?? '';
      if (_controller.text != fallback) _controller.text = fallback;
    }
  }

  void _onChanged(String next) {
    // Typing invalidates the previous canonical selection so the
    // parent knows nothing is picked yet.
    if (widget.selection != null) widget.onSelected(null);
    _debounceHandle?.cancel();
    if (next.trim().length < kCitySearchMinChars) {
      setState(() {
        _results = const [];
        _isLoading = false;
      });
      return;
    }
    _debounceHandle = Timer(_debounce, () => _runSearch(next));
  }

  Future<void> _runSearch(String query) async {
    _cancelToken?.cancel();
    final token = CancelToken();
    _cancelToken = token;
    setState(() => _isLoading = true);
    try {
      final results = await _service.search(
        query: query,
        countryCode: widget.countryCode,
        cancelToken: token,
      );
      if (!mounted || token.isCancelled) return;
      setState(() {
        _results = results;
        _isLoading = false;
      });
    } on DioException {
      if (!mounted || token.isCancelled) return;
      setState(() {
        _results = const [];
        _isLoading = false;
      });
    }
  }

  void _commit(CitySearchResult result) {
    widget.onSelected(result);
    _controller.text = result.city;
    setState(() {
      _results = const [];
      _isLoading = false;
    });
    _focusNode.unfocus();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final hasResults = _results.isNotEmpty;
    final hint = _controller.text.trim().length < kCitySearchMinChars
        ? l10n.tier1LocationCityAutocompleteHint
        : l10n.tier1LocationCityAutocompleteEmpty;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: _controller,
          focusNode: _focusNode,
          textInputAction: TextInputAction.search,
          onChanged: _onChanged,
          decoration: InputDecoration(
            hintText: l10n.tier1LocationCityAutocompletePlaceholder,
            suffixIcon: _isLoading
                ? const Padding(
                    padding: EdgeInsets.all(12),
                    child: SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  )
                : const Icon(Icons.search),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
          ),
        ),
        if (_focused && (hasResults || !_isLoading))
          Container(
            margin: const EdgeInsets.only(top: 4),
            decoration: BoxDecoration(
              color: theme.colorScheme.surface,
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              border: Border.all(
                color: appColors?.border ?? theme.dividerColor,
              ),
            ),
            child: hasResults
                ? _ResultsList(results: _results, onTap: _commit)
                : Padding(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 10,
                    ),
                    child: Text(
                      hint,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: appColors?.mutedForeground,
                      ),
                    ),
                  ),
          ),
      ],
    );
  }
}

class _ResultsList extends StatelessWidget {
  const _ResultsList({required this.results, required this.onTap});

  final List<CitySearchResult> results;
  final ValueChanged<CitySearchResult> onTap;

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: results.length,
      separatorBuilder: (_, __) => const Divider(height: 1),
      itemBuilder: (_, index) {
        final result = results[index];
        return ListTile(
          dense: true,
          title: Text(result.city),
          subtitle: result.context.isEmpty
              ? null
              : Text(
                  result.context,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
          onTap: () => onTap(result),
        );
      },
    );
  }
}
