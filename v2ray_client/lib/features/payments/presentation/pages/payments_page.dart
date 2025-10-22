import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../auth/services/auth_service.dart';
import '../../services/payments_service.dart';

class PaymentsPage extends StatefulWidget {
  const PaymentsPage({super.key});

  @override
  State<PaymentsPage> createState() => _PaymentsPageState();
}

class _PaymentsPageState extends State<PaymentsPage> {
  bool _loading = false;
  String? _error;

  Future<void> _buy(int months) async {
    setState(() { _loading = true; _error = null; });
    try {
      final result = await PaymentsService.createYooKassaPayment(months: months);
      final url = result.confirmationUrl;
      if (!await launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication)) {
        throw Exception('Не удалось открыть ссылку на оплату');
      }
    } catch (e) {
      setState(() { _error = e.toString(); });
    } finally {
      setState(() { _loading = false; });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Платежи')),
      body: Center(
        child: _loading
          ? const CircularProgressIndicator()
          : Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  if (_error != null)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 12.0),
                      child: Text('Ошибка: $_error', style: const TextStyle(color: Colors.red)),
                    ),
                  ElevatedButton(
                    onPressed: () => _buy(1),
                    child: const Text('Купить на 1 месяц — 200 ₽'),
                  ),
                  const SizedBox(height: 12),
                  ElevatedButton(
                    onPressed: () => _buy(3),
                    child: const Text('Купить на 3 месяца — 500 ₽'),
                  ),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    onPressed: () async {
                      try {
                        final profile = await AuthService.getProfile();
                        if (context.mounted) {
                          showDialog(context: context, builder: (_) => AlertDialog(
                            title: const Text('Статус'),
                            content: Text('Доступ: ${profile['has_access']}\nТриал до: ${profile['trial_ends_at'] ?? '-'}\nПодписка до: ${profile['expires_at'] ?? '-'}'),
                            actions: [TextButton(onPressed: ()=>Navigator.pop(context), child: const Text('OK'))],
                          ));
                        }
                      } catch (e) {
                        if (context.mounted) {
                          ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Ошибка: ${e.toString()}')));
                        }
                      }
                    },
                    child: const Text('Проверить статус подписки'),
                  ),
                ],
              ),
            ),
      ),
    );
  }
} 