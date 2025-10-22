import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:v2ray_client/features/auth/services/auth_service.dart';

class PaymentsService {
  static const _baseUrl = 'http://193.124.182.210:8081';

  static Future<({String confirmationUrl, String providerId})> createYooKassaPayment({required int months}) async {
    final token = await AuthService.getToken();
    if (token == null) {
      throw Exception('Нет токена');
    }
    final response = await http.post(
      Uri.parse('$_baseUrl/user/payments/yookassa'),
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      },
      body: jsonEncode({'months': months}),
    );
    if (response.statusCode == 201 || response.statusCode == 200) {
      final data = jsonDecode(response.body);
      final url = data['confirmation_url'] as String?;
      final pid = data['provider_id'] as String?;
      if (url == null) throw Exception('Не получена ссылка на оплату');
      return (confirmationUrl: url, providerId: pid ?? '');
    }
    throw Exception('Ошибка создания платежа: ${response.body}');
  }
} 