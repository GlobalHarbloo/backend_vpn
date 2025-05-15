import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../../../../core/di/injectable.dart';
import '../../../../core/network/api_client.dart';
import '../../application/traffic_cubit.dart';

class TrafficPage extends StatefulWidget {
  const TrafficPage({super.key});

  @override
  State<TrafficPage> createState() => _TrafficPageState();
}

class _TrafficPageState extends State<TrafficPage> {
  @override
  void initState() {
    super.initState();
    _loadTraffic();
  }

  Future<void> _loadTraffic() async {
    final apiClient = getIt<ApiClient>();
    final token = await apiClient.getToken();
    if (!mounted) return;
    if (token != null && token.isNotEmpty) {
      context.read<TrafficCubit>().loadTraffic(token);
    }
  }

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<TrafficCubit, TrafficState>(
      builder: (context, state) {
        if (state is TrafficLoading) {
          return const Center(child: CircularProgressIndicator());
        } else if (state is TrafficLoaded) {
          return Scaffold(
            appBar: AppBar(title: const Text('Статистика трафика')),
            body: Padding(
              padding: const EdgeInsets.all(16.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Загружено: ${state.downlink ~/ 1024} МБ'),
                  const SizedBox(height: 8),
                  Text('Отправлено: ${state.uplink ~/ 1024} МБ'),
                  const SizedBox(height: 8),
                  Text('Всего: ${state.total ~/ 1024} МБ'),
                ],
              ),
            ),
          );
        } else if (state is TrafficError) {
          return Center(child: Text('Ошибка: ${state.message}'));
        } else {
          return const Center(child: Text('Нет данных по трафику'));
        }
      },
    );
  }
}
