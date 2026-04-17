/// location_section.dart — city autocomplete, 2-char country code,
/// and radius km. All three fields feed the backend `city`,
/// `countryCode`, `geoRadiusKm` params respectively.
///
/// The city input debounces edits so a fast-typing user does not
/// flood the parent with intermediate states. Country code coerces
/// to uppercase and caps at 2 chars so ISO-2 compliance is enforced
/// at the UI boundary.
library;

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'filter_primitives.dart';

/// CityFieldDebounceMs is exposed so the integration test can use
/// the same constant. Do not inline — the test checks timing.
const Duration kCityDebounce = Duration(milliseconds: 350);

class LocationSection extends StatefulWidget {
  const LocationSection({
    super.key,
    required this.city,
    required this.countryCode,
    required this.radiusKm,
    required this.onCityChanged,
    required this.onCountryChanged,
    required this.onRadiusChanged,
    required this.sectionTitle,
    required this.cityLabel,
    required this.countryLabel,
    required this.radiusLabel,
  });

  final String city;
  final String countryCode;
  final int? radiusKm;
  final ValueChanged<String> onCityChanged;
  final ValueChanged<String> onCountryChanged;
  final ValueChanged<int?> onRadiusChanged;
  final String sectionTitle;
  final String cityLabel;
  final String countryLabel;
  final String radiusLabel;

  @override
  State<LocationSection> createState() => _LocationSectionState();
}

class _LocationSectionState extends State<LocationSection> {
  late final TextEditingController _cityCtrl =
      TextEditingController(text: widget.city);
  late final TextEditingController _countryCtrl =
      TextEditingController(text: widget.countryCode);
  Timer? _debounce;

  @override
  void didUpdateWidget(covariant LocationSection oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.city != _cityCtrl.text) {
      _cityCtrl.value = TextEditingValue(
        text: widget.city,
        selection: TextSelection.collapsed(offset: widget.city.length),
      );
    }
    if (widget.countryCode != _countryCtrl.text) {
      _countryCtrl.value = TextEditingValue(
        text: widget.countryCode,
        selection:
            TextSelection.collapsed(offset: widget.countryCode.length),
      );
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _cityCtrl.dispose();
    _countryCtrl.dispose();
    super.dispose();
  }

  void _onCityChanged(String raw) {
    _debounce?.cancel();
    _debounce = Timer(kCityDebounce, () {
      widget.onCityChanged(raw);
    });
  }

  void _onCountryChanged(String raw) {
    final coerced = raw.toUpperCase();
    // Coerce to uppercase + cap at 2.
    final clamped = coerced.length > 2 ? coerced.substring(0, 2) : coerced;
    if (clamped != raw) {
      _countryCtrl.value = TextEditingValue(
        text: clamped,
        selection: TextSelection.collapsed(offset: clamped.length),
      );
    }
    widget.onCountryChanged(clamped);
  }

  bool get _hasLocation =>
      widget.city.trim().isNotEmpty || widget.countryCode.trim().isNotEmpty;

  @override
  Widget build(BuildContext context) {
    return FilterSectionShell(
      title: widget.sectionTitle,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Semantics(
            textField: true,
            label: widget.cityLabel,
            child: TextField(
              controller: _cityCtrl,
              onChanged: _onCityChanged,
              decoration: InputDecoration(
                labelText: widget.cityLabel,
                border: const OutlineInputBorder(),
                isDense: true,
              ),
            ),
          ),
          const SizedBox(height: 8),
          Semantics(
            textField: true,
            label: widget.countryLabel,
            child: TextField(
              controller: _countryCtrl,
              onChanged: _onCountryChanged,
              decoration: InputDecoration(
                labelText: widget.countryLabel,
                border: const OutlineInputBorder(),
                isDense: true,
                helperText: 'ISO-2',
              ),
              maxLength: 2,
              textCapitalization: TextCapitalization.characters,
              inputFormatters: [UpperCaseTextFormatter()],
            ),
          ),
          const SizedBox(height: 8),
          Opacity(
            opacity: _hasLocation ? 1 : 0.5,
            child: IgnorePointer(
              ignoring: !_hasLocation,
              child: FilterNumberField(
                value: widget.radiusKm,
                onChanged: widget.onRadiusChanged,
                label: widget.radiusLabel,
                semanticsLabel: widget.radiusLabel,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// UpperCaseTextFormatter forces every input character to uppercase
/// — used by the country code field so a user typing "fr" never
/// bypasses the ISO-2 uppercase invariant.
class UpperCaseTextFormatter extends TextInputFormatter {
  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final upper = newValue.text.toUpperCase();
    if (upper == newValue.text) return newValue;
    return TextEditingValue(
      text: upper,
      selection: newValue.selection,
    );
  }
}
