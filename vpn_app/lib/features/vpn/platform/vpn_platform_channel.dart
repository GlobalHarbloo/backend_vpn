import 'package:flutter/services.dart';

class VpnPlatformChannel {
  static const MethodChannel _channel = MethodChannel('com.vpn_app.channel');

  static Future<void> startVpn(String configPath) async {
    await _channel.invokeMethod('startVpn', {'configPath': configPath});
  }

  static Future<void> stopVpn() async {
    await _channel.invokeMethod('stopVpn');
  }
}
