import 'package:injectable/injectable.dart';
import '../../../core/network/api_client.dart';

@singleton
class ProfileRepository {
  final ApiClient apiClient;
  ProfileRepository(this.apiClient);

  Future<Map<String, dynamic>> fetchProfile(String token) async {
    final data = await apiClient.getProfile();
    return data;
  }
}
