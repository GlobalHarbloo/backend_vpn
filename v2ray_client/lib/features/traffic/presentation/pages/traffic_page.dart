import 'package:flutter/material.dart';

class TrafficPage extends StatefulWidget {
  const TrafficPage({super.key});

  @override
  State<TrafficPage> createState() => _TrafficPageState();
}

class _TrafficPageState extends State<TrafficPage> {
  int? _traffic;
  bool _loading = false;
  String? _error;

  Future<void> _loadTraffic() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    // TODO: Реализовать запрос к backend
    await Future.delayed(const Duration(seconds: 1));
    setState(() {
      _traffic = 12345678; // Пример: 12 МБ
      _loading = false;
    });
  }

  @override
  void initState() {
    super.initState();
    _loadTraffic();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Трафик')),
      body: Center(
        child: _loading
            ? const CircularProgressIndicator()
            : _error != null
                ? Text('Ошибка: $_error', style: const TextStyle(color: Colors.red))
                : Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Text('Трафик: ${_traffic != null ? (_traffic! / (1024 * 1024)).toStringAsFixed(2) : '-'} МБ'),
                      const SizedBox(height: 24),
                      ElevatedButton(
                        onPressed: _loadTraffic,
                        child: const Text('Обновить'),
                      ),
                    ],
                  ),
      ),
    );
  }
} 