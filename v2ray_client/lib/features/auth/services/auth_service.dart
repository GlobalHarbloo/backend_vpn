import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

// Improvements:
// - reuse a single http.Client instance to avoid creating many short-lived connections
// - store both access_token and refresh_token
// - automatic token refresh on 401 responses

class _HttpClient {
  static final http.Client instance = http.Client();
}

class AuthService {
  static const _storage = FlutterSecureStorage();
  static const _baseUrl = 'http://193.124.182.210:8081';
  static const _accessKey = 'access_token';
  static const _refreshKey = 'refresh_token';

  static Future<void> register(String email, String password) async {
    final resp = await _HttpClient.instance.post(
      Uri.parse('$_baseUrl/register'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email, 'password': password}),
    );
    if (resp.statusCode == 200 || resp.statusCode == 201) {
      // Регистрация успешна — сервер возвращает access_token + refresh_token
      final data = jsonDecode(resp.body);
      final access = data['access_token'] ?? data['token'];
      final refresh = data['refresh_token'];
      if (access != null) await _storage.write(key: _accessKey, value: access);
      if (refresh != null)
        await _storage.write(key: _refreshKey, value: refresh);
      return;
    }
    try {
      final data = jsonDecode(resp.body);
      throw Exception(data['error'] ?? resp.body);
    } catch (_) {
      throw Exception('Ошибка регистрации: ${resp.body}');
    }
  }

  static Future<void> login(String email, String password) async {
    final resp = await _HttpClient.instance.post(
      Uri.parse('$_baseUrl/login'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email, 'password': password}),
    );
    if (resp.statusCode == 200) {
      final data = jsonDecode(resp.body);
      final access = data['access_token'] ?? data['token'];
      final refresh = data['refresh_token'];
      if (access == null) throw Exception('Нет токена');
      await _storage.write(key: _accessKey, value: access);
      if (refresh != null)
        await _storage.write(key: _refreshKey, value: refresh);
      return;
    }
    try {
      final data = jsonDecode(resp.body);
      throw Exception(data['error'] ?? resp.body);
    } catch (_) {
      throw Exception('Ошибка авторизации: ${resp.body}');
    }
  }

  static Future<void> logout() async {
    // call server logout to clear refresh token
    final access = await _storage.read(key: _accessKey);
    if (access != null) {
      try {
        await _HttpClient.instance.post(
          Uri.parse('$_baseUrl/user/logout'),
          headers: {'Authorization': 'Bearer $access'},
        );
      } catch (_) {}
    }
    await _storage.delete(key: _accessKey);
    await _storage.delete(key: _refreshKey);
  }

  static Future<String?> getToken() async {
    return await _storage.read(key: _accessKey);
  }

  static Future<String> getBotLink() async {
    final token = await getToken();
    if (token == null) throw Exception('Нет токена');
    final response = await _HttpClient.instance.get(
      Uri.parse('$_baseUrl/user/bot-link'),
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      },
    );
    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return data['link'];
    }
    try {
      final data = jsonDecode(response.body);
      throw Exception(data['error'] ?? response.body);
    } catch (_) {
      throw Exception('Ошибка получения ссылки на бота: ${response.body}');
    }
  }

  static Future<bool> isLoggedIn() async {
    return (await getToken()) != null;
  }

  static Future<Map<String, dynamic>> getProfile() async {
    // Try to get profile, if 401 -> try refresh once and retry.
    final token = await getToken();
    if (token == null) throw Exception('Нет токена');
    try {
      final response = await _HttpClient.instance.get(
        Uri.parse('$_baseUrl/user/me'),
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $token',
        },
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        if (data['uuid'] != null) {
          await _storage.write(key: 'uuid', value: data['uuid']);
        }
        return data;
      }
      if (response.statusCode == 401) {
        final refreshed = await _refresh();
        if (refreshed) {
          final newToken = await getToken();
          if (newToken == null) throw Exception('Нет токена после обновления');
          final resp2 = await _HttpClient.instance.get(
            Uri.parse('$_baseUrl/user/me'),
            headers: {
              'Content-Type': 'application/json',
              'Authorization': 'Bearer $newToken',
            },
          );
          if (resp2.statusCode == 200) {
            final data = jsonDecode(resp2.body);
            if (data['uuid'] != null) {
              await _storage.write(key: 'uuid', value: data['uuid']);
            }
            return data;
          }
          throw Exception(
            'Ошибка получения профиля после обновления токена: ${resp2.body}',
          );
        }
        throw Exception('Unauthorized');
      }
      throw Exception('Ошибка получения профиля: ${response.body}');
    } catch (e) {
      // network errors or truncated responses — retry once after short delay
      try {
        await Future.delayed(const Duration(milliseconds: 300));
        final retryResp = await _HttpClient.instance.get(
          Uri.parse('$_baseUrl/user/me'),
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer $token',
          },
        );
        if (retryResp.statusCode == 200) {
          final data = jsonDecode(retryResp.body);
          if (data['uuid'] != null)
            await _storage.write(key: 'uuid', value: data['uuid']);
          return data;
        }
        if (retryResp.statusCode == 401) {
          final refreshed = await _refresh();
          if (refreshed) {
            final newToken = await getToken();
            if (newToken == null)
              throw Exception('Нет токена после обновления');
            final resp2 = await _HttpClient.instance.get(
              Uri.parse('$_baseUrl/user/me'),
              headers: {
                'Content-Type': 'application/json',
                'Authorization': 'Bearer $newToken',
              },
            );
            if (resp2.statusCode == 200) {
              final data = jsonDecode(resp2.body);
              if (data['uuid'] != null)
                await _storage.write(key: 'uuid', value: data['uuid']);
              return data;
            }
            throw Exception(
              'Ошибка получения профиля после обновления токена: ${resp2.body}',
            );
          }
          throw Exception('Unauthorized');
        }
        throw Exception('Ошибка получения профиля: ${retryResp.body}');
      } catch (e2) {
        rethrow;
      }
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
    final response = await _HttpClient.instance.post(
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

  static Future<void> changePassword(
    String oldPassword,
    String newPassword,
  ) async {
    final token = await getToken();
    if (token == null) throw Exception('Нет токена');
    final response = await _HttpClient.instance.post(
      Uri.parse('$_baseUrl/user/change-password'),
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer $token',
      },
      body: jsonEncode({
        'old_password': oldPassword,
        'new_password': newPassword,
      }),
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
    final response = await _HttpClient.instance.post(
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

  static Future<void> resetPassword(String code, String password) async {
    final response = await _HttpClient.instance.post(
      Uri.parse('$_baseUrl/reset-password'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'code': code, 'password': password}),
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

  // Attempt token refresh using stored refresh_token. Returns true if refreshed.
  static Future<bool> _refresh() async {
    final refresh = await _storage.read(key: _refreshKey);
    if (refresh == null) return false;
    try {
      final resp = await _HttpClient.instance.post(
        Uri.parse('$_baseUrl/refresh'),
        headers: {'Content-Type': 'application/json'},
        body: jsonEncode({'refresh_token': refresh}),
      );
      if (resp.statusCode == 200) {
        final data = jsonDecode(resp.body);
        final access = data['access_token'] ?? data['token'];
        final newRefresh = data['refresh_token'];
        if (access != null)
          await _storage.write(key: _accessKey, value: access);
        if (newRefresh != null)
          await _storage.write(key: _refreshKey, value: newRefresh);
        return true;
      }
    } catch (_) {}
    return false;
  }

  // Helper: run an API call that requires auth; if it returns 401, try refresh once and retry.
  // (Removed unused generic helper)
}
