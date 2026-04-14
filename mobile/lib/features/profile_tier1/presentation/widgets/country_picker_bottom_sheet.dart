import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../utils/country_catalog.dart';
import '../utils/flag_emoji.dart';

/// Opens the country picker as a modal bottom sheet. Resolves to
/// the selected ISO code, or `null` when the user dismisses the
/// sheet without choosing.
Future<String?> showCountryPickerBottomSheet({
  required BuildContext context,
  String? initialCode,
}) {
  return showModalBottomSheet<String>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => _CountryPickerSheet(initialCode: initialCode),
  );
}

class _CountryPickerSheet extends StatefulWidget {
  const _CountryPickerSheet({required this.initialCode});

  final String? initialCode;

  @override
  State<_CountryPickerSheet> createState() => _CountryPickerSheetState();
}

class _CountryPickerSheetState extends State<_CountryPickerSheet> {
  final _query = TextEditingController();
  late String _locale;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _locale = Localizations.localeOf(context).languageCode;
  }

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  List<CountryEntry> _filter(List<CountryEntry> all) {
    final q = _query.text.trim().toLowerCase();
    if (q.isEmpty) return all;
    return all.where((e) {
      return e.labelEn.toLowerCase().contains(q) ||
          e.labelFr.toLowerCase().contains(q) ||
          e.code.toLowerCase().contains(q);
    }).toList(growable: false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final entries = _filter(CountryCatalog.entries);

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.8,
          minChildSize: 0.5,
          maxChildSize: 0.95,
          builder: (_, scrollController) {
            return Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 16, 12, 8),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          l10n.tier1LocationCountryLabel,
                          style: theme.textTheme.titleLarge,
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.close),
                        onPressed: () => Navigator.of(context).pop(),
                      ),
                    ],
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 0, 20, 12),
                  child: TextField(
                    controller: _query,
                    onChanged: (_) => setState(() {}),
                    decoration: InputDecoration(
                      prefixIcon: const Icon(Icons.search, size: 20),
                      hintText: l10n.tier1LocationCountryPlaceholder,
                      border: OutlineInputBorder(
                        borderRadius:
                            BorderRadius.circular(AppTheme.radiusMd),
                      ),
                    ),
                  ),
                ),
                const Divider(height: 1),
                Expanded(
                  child: ListView.builder(
                    controller: scrollController,
                    itemCount: entries.length,
                    itemBuilder: (ctx, i) {
                      final entry = entries[i];
                      final selected =
                          widget.initialCode?.toUpperCase() == entry.code;
                      final label = _locale.startsWith('fr')
                          ? entry.labelFr
                          : entry.labelEn;
                      return ListTile(
                        leading: Text(
                          countryCodeToFlagEmoji(entry.code),
                          style: const TextStyle(fontSize: 24),
                        ),
                        title: Text(label),
                        trailing: selected
                            ? Icon(
                                Icons.check,
                                color: theme.colorScheme.primary,
                              )
                            : null,
                        onTap: () => Navigator.of(context).pop(entry.code),
                      );
                    },
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}
