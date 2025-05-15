import 'package:vpn_app/core/network/api_client.dart';
import 'package:injectable/injectable.dart';

@singleton
class AuthRepository {
  final ApiClient apiClient;
  AuthRepository(this.apiClient);

  Future<String?> login(String email, String password) async {
    try {
      final token = await apiClient.login(email, password);
      return token;
    } catch (_) {
      return null;
    }
  }

  Future<bool> register(String email, String password) async {
    try {
      await apiClient.register(email, password);
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<void> clearToken() async {
    await apiClient.clearToken();
  }
}
