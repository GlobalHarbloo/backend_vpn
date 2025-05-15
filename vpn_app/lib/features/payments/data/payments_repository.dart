import 'package:injectable/injectable.dart';
import '../../../core/network/api_client.dart';

@singleton
class PaymentsRepository {
  final ApiClient apiClient;
  PaymentsRepository(this.apiClient);

  Future<List<dynamic>> fetchPayments(String token) async {
    return await apiClient.getPayments();
  }

  Future<bool> createPayment(
    String token,
    int amount,
    int tariffId,
    String paymentMethod,
  ) async {
    await apiClient.createPayment({
      'amount': amount,
      'tariff_id': tariffId,
      'payment_method': paymentMethod,
    });
    return true;
  }

  Future<bool> changeTariff(String token, int tariffId) async {
    try {
      await apiClient.changeTariff(tariffId);
      return true;
    } catch (_) {
      return false;
    }
  }
}
