import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_v2ray/flutter_v2ray.dart';
import 'package:permission_handler/permission_handler.dart';
import 'package:v2ray_client/features/auth/services/auth_service.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:intl/intl.dart';

class VpnPage extends StatefulWidget {
  const VpnPage({Key? key}) : super(key: key);

  @override
  State<VpnPage> createState() => _VpnPageState();
}

class _VpnPageState extends State<VpnPage> {
  late FlutterV2ray _flutterV2ray;
  String _status = 'Отключено';
  bool _isConnected = false;
  bool _isLoading = false;
  String? _error;
  Map<String, dynamic>? _profile;
  bool _profileLoading = false;
  bool _hasAccess = false;
  String _selectedConfig = 'ws';
  bool _ignoreStatusChange = false;
  V2RayURL? _parser;

  final Map<String, Map<String, dynamic>> _configs = {
    'ws': {
      "v": "2",
      "ps": "openai-ws",
      "add": "193.124.182.210",
      "port": "1443",
      "id": "18b72a2a-a1a6-4f4e-b2fd-997f6060622a",
      "aid": "0",
      "net": "ws",
      "type": "none",
      "host": "",
      "path": "/openai",
      "tls": "",
    },
    'reality': {
      "v": "2",
      "ps": "openai-reality",
      "add": "193.124.182.210",
      "port": "443",
      "id": "18b72a2a-a1a6-4f4e-b2fd-997f6060622a",
      "aid": "0",
      "net": "tcp",
      "type": "none",
      "host": "yahoo.com",
      "path": "",
      "tls": "reality",
      "sni": "yahoo.com",
      "fp": "",
      "alpn": "",
      "pbk": "OKwC9g9Skph_WdQyDu9izHZY37bOlU9pA-V4PzbUJWk",
      "sid": "71fb02",
    },
  };

  @override
  void initState() {
    super.initState();
    _loadProfile();
    _flutterV2ray = FlutterV2ray(
      onStatusChanged: (status) {
        if (_ignoreStatusChange) return;
        setState(() {
          final isInstance = status.toString().contains("V2RayStatus");
          _status = isInstance ? "connected" : "stopped";
          _isConnected = isInstance;
          _isLoading = false;
        });
        debugPrint('[VPN] Status changed: ${status.toString()}');
      },
    );
  }

  String _fmtDate(dynamic value) {
    if (value == null || (value is String && value.isEmpty)) return '-';
    try {
      final dt = DateTime.parse(value.toString()).toLocal();
      return DateFormat('dd.MM.yyyy HH:mm').format(dt);
    } catch (_) {
      return value.toString();
    }
  }

  Future<void> _loadProfile() async {
    setState(() {
      _profileLoading = true;
    });
    try {
      final data = await AuthService.getProfile();
      setState(() {
        _profile = data;
        _hasAccess = data['has_access'] == true;
        _profileLoading = false;
      });
    } catch (e) {
      setState(() {
        _profile = null;
        _profileLoading = false;
      });
    }
  }

  Future<void> _requestNotificationPermission() async {
    if (await Permission.notification.isDenied) {
      await Permission.notification.request();
    }
  }

  String _encodeVlessLink(Map<String, dynamic> config) {
    if (config['tls'] == 'reality') {
      return 'vless://${config['id']}@${config['add']}:${config['port']}?type=tcp&security=reality&sni=${config['sni']}&fp=${config['fp']}&pbk=${config['pbk']}&sid=${config['sid']}&alpn=${config['alpn']}&host=${config['host']}#${config['ps']}';
    } else {
      return 'vless://${config['id']}@${config['add']}:${config['port']}?type=ws&path=${Uri.encodeComponent(config['path'])}&encryption=none#${config['ps']}';
    }
  }

  Future<String?> _getUserUuid() async {
    String? uuid = await AuthService.getUuid();
    if (uuid == null) {
      try {
        final profile = await AuthService.getProfile();
        uuid = profile['uuid'];
      } catch (e) {
        setState(() {
          _error = 'Ошибка получения профиля:\n${e.toString()}';
          _isLoading = false;
        });
        return null;
      }
    }
    return uuid;
  }

  Future<bool> _ensureAccessOrPrompt() async {
    try {
      final profile = await AuthService.getProfile();
      final hasAccess = profile['has_access'] == true;
      if (!hasAccess && mounted) {
        await showDialog(
          context: context,
          builder: (ctx) => AlertDialog(
            title: const Text('Нет доступа'),
            content: const Text(
              'Триал закончился или нет подписки. Перейдите во вкладку Платежи и оформите подписку.',
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('OK'),
              ),
            ],
          ),
        );
      }
      return hasAccess;
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
      return false;
    }
  }

  Future<void> _connect() async {
    setState(() {
      _isLoading = true;
      _error = null;
      _status = 'connecting';
      _isConnected = false;
      _ignoreStatusChange = false;
    });
    try {
      // Проверка доступа
      final allowed = await _ensureAccessOrPrompt();
      if (!allowed) {
        setState(() {
          _isLoading = false;
          _status = 'Отключено';
        });
        return;
      }

      await _requestNotificationPermission();
      await _flutterV2ray.initializeV2Ray();
      final granted = await _flutterV2ray.requestPermission();
      if (!granted) {
        setState(() {
          _status = 'Нет разрешения';
          _isLoading = false;
        });
        return;
      }
      final config = Map<String, dynamic>.from(_configs[_selectedConfig]!);
      final uuid = await _getUserUuid();
      if (uuid == null) return;
      config['id'] = uuid;
      _parser = FlutterV2ray.parseFromURL(_encodeVlessLink(config));
      setState(() {
        _error = null;
        _status = 'connected';
        _isConnected = true;
        _isLoading = false;
      });
      await _flutterV2ray.startV2Ray(
        remark: _parser!.remark,
        config: _parser!.getFullConfiguration(),
        proxyOnly: false,
      );
    } catch (e) {
      setState(() {
        _status = 'Ошибка';
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  Future<void> _disconnect() async {
    setState(() {
      _isLoading = true;
      _error = null;
      _status = 'stopped';
      _isConnected = false;
      _ignoreStatusChange = true;
    });
    try {
      await _flutterV2ray.stopV2Ray();
      debugPrint('[VPN] stopV2Ray called');
      setState(() {
        _isLoading = false;
      });
    } catch (e) {
      setState(() {
        _status = 'Ошибка';
        _error = e.toString();
        _isLoading = false;
      });
    }
  }

  String _statusText(String status) {
    switch (status) {
      case 'connecting':
        return 'Подключение...';
      case 'connected':
        return 'Подключено';
      case 'stopped':
      case 'Отключено':
        return 'Отключено';
      case 'Нет разрешения':
        return 'Нет разрешения';
      case 'Ошибка':
        return 'Ошибка';
      default:
        return status;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('VPN-клиент')),
      body: Center(
        child: SingleChildScrollView(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              if (_profileLoading) ...[
                const Padding(
                  padding: EdgeInsets.all(12.0),
                  child: CircularProgressIndicator(),
                ),
              ],
              // Баннер для триала (если есть trial_ends_at и доступа нет)
              if ((_profile != null &&
                  !_hasAccess &&
                  _profile?['trial_ends_at'] != null)) ...[
                Card(
                  color: Colors.yellow[100],
                  margin: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 8,
                  ),
                  child: Padding(
                    padding: const EdgeInsets.all(12.0),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Expanded(
                          child: Text(
                            'Триал активирован до: ${_fmtDate(_profile?['trial_ends_at'])}. Чтобы продолжить, оплатите через нашего Telegram-бота.',
                            style: const TextStyle(fontSize: 14),
                          ),
                        ),
                        const SizedBox(width: 8),
                        ElevatedButton(
                          onPressed: () async {
                            try {
                              final link = await AuthService.getBotLink();
                              if (link.isNotEmpty)
                                await launchUrl(Uri.parse(link));
                            } catch (e) {
                              if (context.mounted) {
                                ScaffoldMessenger.of(context).showSnackBar(
                                  SnackBar(
                                    content: Text('Ошибка: ${e.toString()}'),
                                  ),
                                );
                              }
                            }
                          },
                          child: const Text('Приобрести лицензию'),
                        ),
                      ],
                    ),
                  ),
                ),
              ],
              const Text('Выберите тип подключения:'),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Radio<String>(
                    value: 'ws',
                    groupValue: _selectedConfig,
                    onChanged: (v) {
                      setState(() {
                        _selectedConfig = v!;
                        _parser = null;
                      });
                    },
                  ),
                  const Text('VLESS WS'),
                  Radio<String>(
                    value: 'reality',
                    groupValue: _selectedConfig,
                    onChanged: (v) {
                      setState(() {
                        _selectedConfig = v!;
                        _parser = null;
                      });
                    },
                  ),
                  const Text('VLESS Reality'),
                ],
              ),
              Text(
                'Статус: ${_statusText(_status)}',
                style: const TextStyle(fontSize: 20),
              ),
              if (_error != null) ...[
                const SizedBox(height: 8),
                Text(
                  'Ошибка: $_error',
                  style: const TextStyle(color: Colors.red),
                ),
              ],
              const SizedBox(height: 32),
              ElevatedButton(
                onPressed: _isConnected || _isLoading ? null : _connect,
                child: _isLoading && !_isConnected
                    ? const CircularProgressIndicator(color: Colors.white)
                    : const Text('Подключиться'),
              ),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: !_isConnected || _isLoading ? null : _disconnect,
                child: _isLoading && _isConnected
                    ? const CircularProgressIndicator(color: Colors.white)
                    : const Text('Отключиться'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
