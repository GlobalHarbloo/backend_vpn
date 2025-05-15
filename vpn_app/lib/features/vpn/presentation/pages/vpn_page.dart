import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'dart:convert';
import '../../application/vpn_cubit.dart';
import '../../platform/vpn_platform_channel.dart';
import 'dart:io';
import '../../../../core/di/injectable.dart';
import '../../../../core/network/api_client.dart';

class VpnPage extends StatefulWidget {
  const VpnPage({super.key});

  @override
  State<VpnPage> createState() => _VpnPageState();
}

class _VpnPageState extends State<VpnPage> {
  bool _vpnActive = false;

  @override
  void initState() {
    super.initState();
    _loadConfig();
  }

  Future<void> _loadConfig() async {
    final apiClient = getIt<ApiClient>();
    final token = await apiClient.getToken();
    if (!mounted) return;
    if (token != null && token.isNotEmpty) {
      context.read<VpnCubit>().fetchConfig(token);
    }
  }

  void _toggleVpnFromServerConfig(String serverConfigJson) async {
    setState(() {
      _vpnActive = !_vpnActive;
    });
    if (_vpnActive) {
      // 1. Двойной jsonDecode для вложенного JSON
      dynamic serverConfig;
      try {
        final firstDecode = jsonDecode(serverConfigJson);
        serverConfig =
            firstDecode is String ? jsonDecode(firstDecode) : firstDecode;
      } catch (e) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Ошибка парсинга VPN-конфига с сервера'),
          ),
        );
        setState(() {
          _vpnActive = false;
        });
        return;
      }
      if (serverConfig == null ||
          serverConfig['inbounds'] == null ||
          serverConfig['inbounds'] is! List ||
          serverConfig['inbounds'].isEmpty ||
          serverConfig['inbounds'][0]['settings'] == null) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Некорректные данные VPN-конфига с сервера'),
          ),
        );
        setState(() {
          _vpnActive = false;
        });
        return;
      }
      final inbound = serverConfig['inbounds'][0];
      final clientsList =
          inbound['settings']['clients'] ?? inbound['settings']['users'];
      if (clientsList == null || clientsList is! List || clientsList.isEmpty) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Некорректные данные VPN-конфига с сервера'),
          ),
        );
        setState(() {
          _vpnActive = false;
        });
        return;
      }
      final client = clientsList[0];
      final email = client['email'] ?? client['name'] ?? '';
      final uuid = client['id'] ?? client['uuid'] ?? '';

      // 2. Сформировать sing-box-конфиг
      final config = _generateSingBoxConfig(uuid, email);

      // 3. Сохранить и запустить
      final tempDir = await Directory.systemTemp.createTemp('vpn_config');
      final configPath = '${tempDir.path}/config.json';
      await File(configPath).writeAsString(config);
      await VpnPlatformChannel.startVpn(configPath);
    } else {
      await VpnPlatformChannel.stopVpn();
    }
  }

  String _generateSingBoxConfig(String uuid, String email) {
    return '''
{
  "log": {
    "level": "warn",
    "output": "box.log",
    "timestamp": true
  },
  "dns": {
    "servers": [
      {
        "tag": "dns-remote",
        "address": "udp://1.1.1.1"
      },
      {
        "tag": "dns-local",
        "address": "local",
        "detour": "direct"
      }
    ],
    "final": "dns-remote"
  },
  "inbounds": [
    {
      "type": "tun",
      "tag": "tun-in",
      "mtu": 9000,
      "address": [
        "172.19.0.2/28"
      ],
      "auto_route": true,
      "strict_route": true,
      "endpoint_independent_nat": true,
      "stack": "system",
      "sniff": true,
      "sniff_override_destination": true
    }
  ],
  "outbounds": [
    {
      "type": "vless",
      "tag": "main-vless",
      "server": "193.124.182.210",
      "server_port": 433,
      "uuid": "$uuid",
      "tls": {
        "enabled": true,
        "server_name": "yahoo.com",
        "utls": {
          "enabled": true,
          "fingerprint": "chrome"
        },
        "reality": {
          "enabled": true,
          "public_key": "Ewr5HbGZ07NPWi1JYzZO8DzYVasXnCGy2OTESXkFvl4",
          "short_id": "71fb02"
        }
      },
      "packet_encoding": "xudp"
    },
    {
      "type": "direct",
      "tag": "direct"
    },
    {
      "type": "dns",
      "tag": "dns-out"
    },
    {
      "type": "block",
      "tag": "block"
    }
  ],
  "route": {
    "rules": [
      {
        "protocol": "dns",
        "outbound": "dns-remote"
      },
      {
        "geoip": [
          "private"
        ],
        "outbound": "direct"
      },
      {
        "ip_cidr": [
          "0.0.0.0/0",
          "::/0"
        ],
        "outbound": "main-vless"
      }
    ],
    "final": "direct",
    "auto_detect_interface": true
  }
}
''';
  }

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<VpnCubit, VpnState>(
      builder: (context, state) {
        String status = _vpnActive ? 'Подключено' : 'Отключено';
        String buttonText = _vpnActive ? 'Отключить' : 'Подключить';
        String traffic = '0 МБ / 10000 МБ';
        return Scaffold(
          appBar: AppBar(title: const Text('VPN Подключение')),
          body: Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text('Статус: $status'),
                const SizedBox(height: 16),
                if (state is VpnLoading)
                  const CircularProgressIndicator()
                else if (state is VpnConnected)
                  ElevatedButton(
                    onPressed: () {
                      _toggleVpnFromServerConfig(state.config);
                    },
                    child: Text(buttonText),
                  )
                else if (state is VpnError)
                  Text('Ошибка: ${state.message}'),
                const SizedBox(height: 24),
                Text('Трафик: $traffic'),
              ],
            ),
          ),
        );
      },
    );
  }
}
