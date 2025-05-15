import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'core/di/injectable.dart';
import 'features/auth/application/auth_cubit.dart';
import 'features/profile/application/profile_cubit.dart';
import 'features/traffic/application/traffic_cubit.dart';
import 'features/vpn/application/vpn_cubit.dart';
import 'features/payments/application/payments_cubit.dart';
import 'features/core/presentation/pages/splash_page.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  configureDependencies();
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MultiBlocProvider(
      providers: [
        BlocProvider<AuthCubit>(create: (_) => getIt<AuthCubit>()),
        BlocProvider<ProfileCubit>(create: (_) => getIt<ProfileCubit>()),
        BlocProvider<TrafficCubit>(create: (_) => getIt<TrafficCubit>()),
        BlocProvider<VpnCubit>(create: (_) => getIt<VpnCubit>()),
        BlocProvider<PaymentsCubit>(create: (_) => getIt<PaymentsCubit>()),
      ],
      child: MaterialApp(
        title: 'VPN App',
        theme: ThemeData(
          colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
          useMaterial3: true,
        ),
        home: const SplashPage(),
      ),
    );
  }
}

Future<void> clearAllAppData() async {
  final prefs = await SharedPreferences.getInstance();
  await prefs.clear();
  const storage = FlutterSecureStorage();
  await storage.deleteAll();
}
