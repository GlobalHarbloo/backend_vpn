#include "vpn_method_channel.h"
#include <flutter/method_channel.h>
#include <flutter/standard_method_codec.h>
#include <windows.h>
#include <memory>
#include <string>
#include <iostream>
#include <algorithm> // Для std::replace

void RegisterVpnMethodChannel(flutter::FlutterViewController* controller) {
  auto messenger = controller->engine()->messenger();
  auto channel = std::make_unique<flutter::MethodChannel<flutter::EncodableValue>>(
      messenger, "com.vpn_app.channel", &flutter::StandardMethodCodec::GetInstance());
  channel->SetMethodCallHandler(
      [](const flutter::MethodCall<flutter::EncodableValue>& call,
         std::unique_ptr<flutter::MethodResult<flutter::EncodableValue>> result) {
        if (call.method_name() == "startVpn") {
          const auto* args = std::get_if<flutter::EncodableMap>(call.arguments());
          if (args && args->count(flutter::EncodableValue("configPath"))) {
            std::string configPath = std::get<std::string>((*args).at(flutter::EncodableValue("configPath")));
            std::replace(configPath.begin(), configPath.end(), '/', '\\');
            std::string exePath = "C:\\Users\\glebs\\Desktop\\vpn-backend\\vpn_app\\bin\\sing-box.exe";
            // Используем start для асинхронного запуска
            std::string command = "start \"singbox\" \"" + exePath + "\" run -c " + configPath;
            std::cout << "RUN: " << command << std::endl;
            system(command.c_str());
            result->Success();
            return;
          }
        } else if (call.method_name() == "stopVpn") {
          system("taskkill /IM sing-box.exe /F");
          result->Success();
          return;
        }
        result->NotImplemented();
      });
}
