import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../providers/subscription_providers.dart';

/// Landing screen after a successful Stripe Checkout.
///
/// The Checkout flow redirects the browser here via a universal link.
/// Because the Stripe webhook is async, we poll `subscriptionProvider`
/// every 2 seconds until a non-null [Subscription] arrives or the 30s
/// safety timeout fires.
class BillingSuccessScreen extends ConsumerStatefulWidget {
  const BillingSuccessScreen({super.key});

  @override
  ConsumerState<BillingSuccessScreen> createState() =>
      _BillingSuccessScreenState();
}

class _BillingSuccessScreenState extends ConsumerState<BillingSuccessScreen> {
  static const _pollInterval = Duration(seconds: 2);
  static const _timeout = Duration(seconds: 30);

  Timer? _pollTimer;
  Timer? _timeoutTimer;
  bool _timedOut = false;
  bool _done = false;

  @override
  void initState() {
    super.initState();
    _startPolling();
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _timeoutTimer?.cancel();
    super.dispose();
  }

  void _startPolling() {
    _timedOut = false;
    _done = false;
    // Force an immediate refresh so we don't rely on the autoDispose
    // cache being dirty already.
    ref.invalidate(subscriptionProvider);
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(_pollInterval, (_) {
      if (!mounted) return;
      ref.invalidate(subscriptionProvider);
    });
    _timeoutTimer?.cancel();
    _timeoutTimer = Timer(_timeout, () {
      if (!mounted) return;
      setState(() => _timedOut = true);
      _pollTimer?.cancel();
    });
  }

  void _stopPolling() {
    _pollTimer?.cancel();
    _timeoutTimer?.cancel();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // Listen to the provider to catch the first non-null subscription.
    ref.listen(subscriptionProvider, (previous, next) {
      next.whenOrNull(
        data: (sub) {
          if (sub != null && !_done) {
            _done = true;
            _stopPolling();
            if (mounted) setState(() {});
          }
        },
      );
    });

    final async = ref.watch(subscriptionProvider);
    final hasSub = async.maybeWhen(
      data: (sub) => sub != null,
      orElse: () => false,
    );

    return Scaffold(
      appBar: AppBar(title: const Text('Premium')),
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                if (hasSub)
                  _SuccessState()
                else if (_timedOut)
                  _TimeoutState(onRetry: _startPolling)
                else
                  _LoadingState(),
                const SizedBox(height: 32),
                if (hasSub)
                  ElevatedButton(
                    onPressed: () => context.go('/dashboard'),
                    child: const Text('Accéder à mon espace'),
                  )
                else if (_timedOut)
                  OutlinedButton(
                    onPressed: _startPolling,
                    child: const Text('Rafraîchir'),
                  )
                else
                  Text(
                    'Cela ne devrait prendre que quelques secondes.',
                    style: theme.textTheme.bodySmall,
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _LoadingState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        const SizedBox(
          width: 48,
          height: 48,
          child: CircularProgressIndicator(strokeWidth: 3),
        ),
        const SizedBox(height: 16),
        Text(
          "Finalisation de ton abonnement…",
          style: Theme.of(context).textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
      ],
    );
  }
}

class _TimeoutState extends StatelessWidget {
  const _TimeoutState({required this.onRetry});

  // Required by the layout — kept explicit so the parent knows to pass
  // it, even if this inner widget does not call it directly (the
  // outer screen has the primary retry button).
  // ignore: unused_element
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Icon(
          Icons.hourglass_bottom,
          size: 48,
          color: Theme.of(context).colorScheme.primary,
        ),
        const SizedBox(height: 16),
        Text(
          'Prend un peu plus de temps que prévu',
          style: Theme.of(context).textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        Text(
          "Ton paiement est bien enregistré côté Stripe. "
          "Réessaie dans quelques instants.",
          style: Theme.of(context).textTheme.bodySmall,
          textAlign: TextAlign.center,
        ),
      ],
    );
  }
}

class _SuccessState extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Icon(
          Icons.check_circle,
          size: 64,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(height: 16),
        Text(
          'Bienvenue sur Premium 🎉',
          style: theme.textTheme.headlineMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        Text(
          'Tu gardes 100% de tes revenus sur chaque mission. '
          'Tu peux gérer ton abonnement à tout moment.',
          style: theme.textTheme.bodyMedium,
          textAlign: TextAlign.center,
        ),
      ],
    );
  }
}
