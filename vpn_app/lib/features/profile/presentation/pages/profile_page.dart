import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'dart:io' show Platform;
import '../../application/profile_cubit.dart';
import '../../application/profile_state.dart';
import '../../../auth/presentation/pages/login_page.dart';
import '../../../auth/application/auth_cubit.dart';

class ProfilePage extends StatelessWidget {
  const ProfilePage({super.key});

  Future<void> _logout(BuildContext context) async {
    context.read<ProfileCubit>().logout();
    context.read<AuthCubit>().logout();
    if (!context.mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const LoginPage()),
      (route) => false,
    );
  }

  Future<void> _clearToken(BuildContext context) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.clear();
    // Только для мобильных и web — безопасно вызывать flutter_secure_storage
    if (!(kIsWeb == false &&
        (Platform.isWindows || Platform.isLinux || Platform.isMacOS))) {
      const storage = FlutterSecureStorage();
      await storage.deleteAll();
    }
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text(
          'Токен удалён. Перезапустите приложение или войдите снова.',
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return BlocBuilder<ProfileCubit, ProfileState>(
      builder: (context, state) {
        if (state is ProfileLoading) {
          return const Center(child: CircularProgressIndicator());
        } else if (state is ProfileLoaded) {
          final user = state.user;
          return Scaffold(
            appBar: AppBar(title: const Text('Профиль')),
            body: Padding(
              padding: const EdgeInsets.all(16.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Email: ${user.email}'),
                  const SizedBox(height: 8),
                  Text('Тариф ID: ${user.tariffId?.toString() ?? "-"}'),
                  const SizedBox(height: 8),
                  Text(
                    'Дата окончания: ${user.expiresAt != null ? user.expiresAt!.toLocal().toString() : "-"}',
                  ),
                  const SizedBox(height: 8),
                  Text('Трафик: ${user.traffic?.toString() ?? "0"}'),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    onPressed: () => _logout(context),
                    child: const Text('Выйти'),
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton(
                    onPressed: () => _clearToken(context),
                    child: const Text('Сбросить токен (ручной выход)'),
                  ),
                ],
              ),
            ),
          );
        } else if (state is ProfileError) {
          return Center(child: Text('Ошибка: ${state.message}'));
        } else {
          return const Center(child: Text('Нет данных профиля'));
        }
      },
    );
  }
}
