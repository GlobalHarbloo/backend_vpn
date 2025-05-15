import 'package:injectable/injectable.dart';
import '../../../core/network/api_client.dart';

@singleton
class VpnRepository {
  final ApiClient apiClient;
  VpnRepository(this.apiClient);

  Future<String> fetchVpnConfig(String token) async {
    // Получаем кусок конфига для клиента из API (поле "config" как строка)
    return await apiClient.getVpnConfig();
  }
}
