import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import '../../../profile/application/profile_cubit.dart';
import '../../../auth/presentation/pages/login_page.dart';
import 'main_nav_page.dart';
import '../../../../core/di/injectable.dart';
import '../../../../core/network/api_client.dart';

class SplashPage extends StatefulWidget {
  const SplashPage({super.key});

  @override
  State<SplashPage> createState() => _SplashPageState();
}

class _SplashPageState extends State<SplashPage> {
  late final ApiClient _apiClient;

  @override
  void initState() {
    super.initState();
    _apiClient = getIt<ApiClient>();
    _checkAuth();
  }

  Future<void> _checkAuth() async {
    try {
      final hasToken = await _apiClient.hasToken();
      if (!mounted) return;
      if (hasToken) {
        final profileCubit = context.read<ProfileCubit>();
        await profileCubit.loadProfileTokenless();
        if (!mounted) return;
        Navigator.of(context).pushReplacement(
          MaterialPageRoute(builder: (_) => const MainNavPage()),
        );
      } else {
        if (!mounted) return;
        Navigator.of(
          context,
        ).pushReplacement(MaterialPageRoute(builder: (_) => const LoginPage()));
      }
    } catch (e) {
      if (!mounted) return;
      Navigator.of(
        context,
      ).pushReplacement(MaterialPageRoute(builder: (_) => const LoginPage()));
    }
  }

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: CircularProgressIndicator()));
  }
}
