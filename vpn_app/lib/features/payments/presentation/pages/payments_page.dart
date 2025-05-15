import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../../../../core/di/injectable.dart';
import '../../../../core/network/api_client.dart';
import '../../application/payments_cubit.dart';

class PaymentsPage extends StatefulWidget {
  const PaymentsPage({super.key});

  @override
  State<PaymentsPage> createState() => _PaymentsPageState();
}

class _PaymentsPageState extends State<PaymentsPage> {
  @override
  void initState() {
    super.initState();
    _loadPayments();
  }

  Future<void> _loadPayments() async {
    final apiClient = getIt<ApiClient>();
    final token = await apiClient.getToken();
    if (!mounted) return;
    if (token != null && token.isNotEmpty) {
      context.read<PaymentsCubit>().loadPayments(token);
    }
  }

  void _showCreatePaymentDialog(BuildContext context) async {
    final apiClient = getIt<ApiClient>();
    final token = await apiClient.getToken();
    if (!mounted) return;
    int? amount;
    int? tariffId;
    String paymentMethod = 'credit_card';
    showDialog(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('Создать платеж'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                decoration: const InputDecoration(labelText: 'Сумма'),
                keyboardType: TextInputType.number,
                onChanged: (v) => amount = int.tryParse(v),
              ),
              TextField(
                decoration: const InputDecoration(labelText: 'ID тарифа'),
                keyboardType: TextInputType.number,
                onChanged: (v) => tariffId = int.tryParse(v),
              ),
              DropdownButton<String>(
                value: paymentMethod,
                items: const [
                  DropdownMenuItem(value: 'credit_card', child: Text('Карта')),
                  DropdownMenuItem(value: 'yoomoney', child: Text('ЮMoney')),
                ],
                onChanged: (v) {
                  if (v != null) paymentMethod = v;
                },
              ),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: const Text('Отмена'),
            ),
            ElevatedButton(
              onPressed: () {
                if (token != null && amount != null && tariffId != null) {
                  context.read<PaymentsCubit>().createPayment(
                    token,
                    amount!,
                    tariffId!,
                    paymentMethod,
                  );
                  Navigator.of(context).pop();
                }
              },
              child: const Text('Создать'),
            ),
          ],
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<PaymentsCubit, PaymentsState>(
      builder: (context, state) {
        if (state is PaymentsLoading) {
          return const Center(child: CircularProgressIndicator());
        } else if (state is PaymentsLoaded) {
          return Scaffold(
            appBar: AppBar(
              title: const Text('Платежи и подписка'),
              actions: [
                IconButton(
                  icon: const Icon(Icons.add),
                  onPressed: () => _showCreatePaymentDialog(context),
                  tooltip: 'Создать платеж',
                ),
              ],
            ),
            body: ListView.builder(
              padding: const EdgeInsets.all(16.0),
              itemCount: state.payments.length,
              itemBuilder: (context, index) {
                final payment = state.payments[index];
                return ListTile(
                  title: Text('Тариф: ${payment['tariff_id']}'),
                  subtitle: Text('Статус: ${payment['status']}'),
                  trailing: Text('${payment['amount']} ₽'),
                  onTap: () async {
                    final apiClient = getIt<ApiClient>();
                    final token = await apiClient.getToken();
                    if (!mounted) return;
                    if (token != null) {
                      context.read<PaymentsCubit>().changeTariff(
                        token,
                        payment['tariff_id'],
                      );
                    }
                  },
                );
              },
            ),
          );
        } else if (state is PaymentsError) {
          return Center(child: Text('Ошибка: ${state.message}'));
        } else {
          return const Center(child: Text('Нет данных по платежам'));
        }
      },
    );
  }
}
