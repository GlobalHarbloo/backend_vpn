import 'package:injectable/injectable.dart';
import '../../../core/network/api_client.dart';

@singleton
class TrafficRepository {
  final ApiClient apiClient;
  TrafficRepository(this.apiClient);

  Future<Map<String, dynamic>> fetchTrafficStats(String token) async {
    return await apiClient.getTraffic();
  }
}
