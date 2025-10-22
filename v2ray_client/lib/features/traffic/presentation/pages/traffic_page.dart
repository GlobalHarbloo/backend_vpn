import 'package:flutter/material.dart';
import 'package:v2ray_client/features/auth/services/auth_service.dart';
import 'dart:convert';
import 'package:http/http.dart' as http;

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
    try {
      final token = await AuthService.getToken();
      if (token == null) throw Exception('Нет токена');
      final response = await http.get(
        Uri.parse('http://193.124.182.210:8081/user/traffic'),
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $token',
        },
      );
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body);
        setState(() {
          _traffic = data['traffic'] is int ? data['traffic'] : int.tryParse(data['traffic'].toString());
          _loading = false;
        });
      } else {
        setState(() {
          _error = 'Ошибка: ${response.body}';
          _loading = false;
        });
      }
    } catch (e) {
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
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