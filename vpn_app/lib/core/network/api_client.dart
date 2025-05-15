import 'package:http/http.dart' as http;
import 'package:flutter/foundation.dart' show kIsWeb;
import 'dart:io' show Platform;
import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:injectable/injectable.dart';
import '../config/app_config.dart';

@lazySingleton
class ApiClient {
  final FlutterSecureStorage _secureStorage;
  static const String _tokenKey = 'auth_token';

  ApiClient(this._secureStorage);

  Future<void> saveToken(String token) async {
    if (_isDesktop()) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_tokenKey, token);
    } else {
      await _secureStorage.write(key: _tokenKey, value: token);
    }
  }

  Future<void> clearToken() async {
    if (_isDesktop()) {
      final prefs = await SharedPreferences.getInstance();
      await prefs.remove(_tokenKey);
    } else {
      await _secureStorage.delete(key: _tokenKey);
    }
  }

  Future<bool> hasToken() async {
    final token = await getToken();
    return token != null && token.isNotEmpty;
  }

  Future<String?> getToken() async {
    if (_isDesktop()) {
      final prefs = await SharedPreferences.getInstance();
      return prefs.getString(_tokenKey);
    } else {
      return await _secureStorage.read(key: _tokenKey);
    }
  }

  bool _isDesktop() {
    return kIsWeb == false &&
        (Platform.isWindows || Platform.isLinux || Platform.isMacOS);
  }

  Future<String> login(String email, String password) async {
    final response = await http.post(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.loginEndpoint}'),
      headers: {'Content-Type': 'application/json'},
      body: '{"email":"$email","password":"$password"}',
    );
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      final token = data['token'] as String?;
      if (token != null) {
        await saveToken(token);
        return token;
      } else {
        throw Exception('Token not found in response');
      }
    } else {
      throw Exception('Login failed: ${response.body}');
    }
  }

  Future<void> register(String email, String password) async {
    final response = await http.post(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.registerEndpoint}'),
      headers: {'Content-Type': 'application/json'},
      body: '{"email":"$email","password":"$password"}',
    );
    if (response.statusCode != 200) {
      throw Exception('Registration failed: ${response.body}');
    }
  }

  Future<Map<String, dynamic>> getProfile() async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.userMeEndpoint}'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to fetch profile: ${response.body}');
    }
  }

  Future<List<dynamic>> getTariffs() async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/tariffs'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to fetch tariffs: ${response.body}');
    }
  }

  Future<Map<String, dynamic>> getCurrentTariff() async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/tariff'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to fetch current tariff: ${response.body}');
    }
  }

  Future<Map<String, dynamic>> getTraffic() async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.trafficEndpoint}'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to fetch traffic: ${response.body}');
    }
  }

  Future<Map<String, dynamic>> getTrafficLimits() async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/traffic/limits'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to fetch traffic limits: ${response.body}');
    }
  }

  Future<List<dynamic>> getPayments() async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.paymentsEndpoint}'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to fetch payments: ${response.body}');
    }
  }

  Future<Map<String, dynamic>> createPayment(Map<String, dynamic> data) async {
    final token = await getToken();
    final response = await http.post(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.paymentsEndpoint}'),
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: jsonEncode(data),
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to create payment: ${response.body}');
    }
  }

  Future<Map<String, dynamic>> getPaymentById(int id) async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.paymentsEndpoint}/$id'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      return jsonDecode(response.body);
    } else {
      throw Exception('Failed to fetch payment by ID: ${response.body}');
    }
  }

  Future<void> updatePaymentStatus(int id, String status) async {
    final token = await getToken();
    final response = await http.put(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.paymentsEndpoint}/$id'),
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: jsonEncode({'status': status}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to update payment status: ${response.body}');
    }
  }

  Future<void> changeTariff(int tariffId) async {
    final token = await getToken();
    final response = await http.post(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.changeTariffEndpoint}'),
      headers: {
        'Authorization': 'Bearer $token',
        'Content-Type': 'application/json',
      },
      body: jsonEncode({'tariff_id': tariffId}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to change tariff: ${response.body}');
    }
  }

  Future<void> deleteAccount() async {
    final token = await getToken();
    final response = await http.post(
      Uri.parse('${AppConfig.baseUrl}/delete-account'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to delete account: ${response.body}');
    }
  }

  Future<String> getVpnConfig() async {
    final token = await getToken();
    final response = await http.get(
      Uri.parse('${AppConfig.baseUrl}/${AppConfig.vpnConfigEndpoint}'),
      headers: {'Authorization': 'Bearer $token'},
    );
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      // Предполагается, что сервер возвращает {"config": "..."}
      return data['config'] as String;
    } else {
      throw Exception('Failed to fetch VPN config: ${response.body}');
    }
  }
}
