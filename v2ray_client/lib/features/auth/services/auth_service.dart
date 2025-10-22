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
    if (response.statusCode == 200 || response.statusCode == 201) {
      // Регистрация успешна, сразу логиним пользователя
      await login(email, password);
      return;
    } else {
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

  static Future<Map<String, dynamic>> getProfile() async {
    final token = await getToken();
    if (token == null) throw Exception('Нет токена');
    final response = await http.get(
      Uri.parse('$_baseUrl/user/me'),
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      },
    );
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      // Сохраняем uuid для дальнейшего использования
      if (data['uuid'] != null) {
        await _storage.write(key: 'uuid', value: data['uuid']);
      }
      return data;
    } else {
      throw Exception('Ошибка получения профиля: ${response.body}');
    }
  }

  static Future<String?> getUuid() async {
    return await _storage.read(key: 'uuid');
  }

  static Future<void> setUuid(String uuid) async {
    await _storage.write(key: 'uuid', value: uuid);
  }

  static Future<void> deleteAccount() async {
    final token = await getToken();
    if (token == null) throw Exception('Нет токена');
    final response = await http.post(
      Uri.parse('$_baseUrl/user/delete-account'),
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      },
    );
    if (response.statusCode == 200) {
      return;
    } else {
      try {
        final data = jsonDecode(response.body);
        throw Exception(data['error'] ?? response.body);
      } catch (_) {
        throw Exception('Ошибка удаления: ${response.body}');
      }
    }
  }

  static Future<void> changePassword(String oldPassword, String newPassword) async {
    final token = await getToken();
    if (token == null) throw Exception('Нет токена');
    final response = await http.post(
      Uri.parse('$_baseUrl/user/change-password'),
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      },
      body: jsonEncode({'old_password': oldPassword, 'new_password': newPassword}),
    );
    if (response.statusCode == 200) {
      return;
    } else {
      try {
        final data = jsonDecode(response.body);
        throw Exception(data['error'] ?? response.body);
      } catch (_) {
        throw Exception('Ошибка смены пароля: ${response.body}');
      }
    }
  }

  static Future<void> requestPasswordReset(String email) async {
    final response = await http.post(
      Uri.parse('$_baseUrl/user/request-password-reset'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email}),
    );
    if (response.statusCode == 200) {
      return;
    } else {
      try {
        final data = jsonDecode(response.body);
        throw Exception(data['error'] ?? response.body);
      } catch (_) {
        throw Exception('Ошибка сброса пароля: ${response.body}');
      }
    }
  }
} 