import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class AuthService {
  static const _storage = FlutterSecureStorage();
  static const _baseUrl = 'http://193.124.182.210:8081';

  static Future<void> register(String email, String password) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/register'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email, 'password': password}),
    );
    if (response.statusCode != 200) {
      try {
        final data = jsonDecode(response.body);
        throw Exception(data['error'] ?? response.body);
      } catch (_) {
        throw Exception('Ошибка регистрации: ${response.body}');
      }
    }
  }

  static Future<void> login(String email, String password) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/login'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email, 'password': password}),
    );
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      final token = data['token'];
      if (token == null) throw Exception('Нет токена');
      await _storage.write(key: 'token', value: token);
    } else {
      try {
        final data = jsonDecode(response.body);
        throw Exception(data['error'] ?? response.body);
      } catch (_) {
        throw Exception('Ошибка авторизации: ${response.body}');
      }
    }
  }

  static Future<void> logout() async {
    await _storage.delete(key: 'token');
  }

  static Future<String?> getToken() async {
    return await _storage.read(key: 'token');
  }

  static Future<bool> isLoggedIn() async {
    return (await getToken()) != null;
  }
} 