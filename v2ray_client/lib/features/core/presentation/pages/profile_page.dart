import 'package:flutter/material.dart';
import '../../../auth/services/auth_service.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:intl/intl.dart';

class ProfilePage extends StatefulWidget {
  final ValueNotifier<int>? tabNotifier;
  final int? tabIndex;
  const ProfilePage({super.key, this.tabNotifier, this.tabIndex});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  Map<String, dynamic>? _profile;
  bool _loading = false;
  String? _error;
  bool _refreshCooldown = false;

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
      _loading = true;
      _error = null;
    });
    try {
      final data = await AuthService.getProfile();
      setState(() {
        _profile = data;
        _loading = false;
      });
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
    _loadProfile();
    widget.tabNotifier?.addListener(_onTabChanged);
  }

  void _onTabChanged() {
    if (widget.tabIndex != null && widget.tabNotifier != null) {
      if (widget.tabNotifier!.value == widget.tabIndex && !_loading) {
        _loadProfile();
      }
    }
  }

  @override
  void dispose() {
    widget.tabNotifier?.removeListener(_onTabChanged);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasAccess = _profile?['has_access'] == true;
    final expiresAt = _fmtDate(_profile?['expires_at']);
    final trialEndsAt = _fmtDate(_profile?['trial_ends_at']);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Профиль'),
        actions: [
          IconButton(
            onPressed: (_loading || _refreshCooldown)
                ? null
                : () async {
                    // prevent hammering the server: short cooldown
                    setState(() {
                      _refreshCooldown = true;
                    });
                    try {
                      await _loadProfile();
                    } finally {
                      await Future.delayed(const Duration(seconds: 2));
                      if (mounted) setState(() => _refreshCooldown = false);
                    }
                  },
            icon: const Icon(Icons.refresh),
            tooltip: 'Обновить',
          ),
        ],
      ),
      body: Center(
        child: _loading
            ? const CircularProgressIndicator()
            : _error != null
            ? Text('Ошибка: $_error', style: const TextStyle(color: Colors.red))
            : Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    // Баннер: если триал закончился и доступа нет — показываем ссылку на Telegram бота
                    if (!hasAccess && _profile?['trial_ends_at'] != null) ...[
                      Card(
                        color: Colors.yellow[100],
                        child: Padding(
                          padding: const EdgeInsets.all(12.0),
                          child: Row(
                            mainAxisAlignment: MainAxisAlignment.spaceBetween,
                            children: [
                              const Expanded(
                                child: Text(
                                  'Триал закончился — оплатите подписку через нашего Telegram-бота.',
                                ),
                              ),
                              ElevatedButton(
                                onPressed: () async {
                                  try {
                                    final link = await AuthService.getBotLink();
                                    if (link.isNotEmpty) {
                                      await launchUrl(Uri.parse(link));
                                    }
                                  } catch (e) {
                                    if (context.mounted) {
                                      ScaffoldMessenger.of(
                                        context,
                                      ).showSnackBar(
                                        SnackBar(
                                          content: Text(
                                            'Ошибка: ${e.toString()}',
                                          ),
                                        ),
                                      );
                                    }
                                  }
                                },
                                child: const Text('Оплатить'),
                              ),
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                    ],
                    // Telegram support link
                    Padding(
                      padding: const EdgeInsets.only(bottom: 16.0),
                      child: InkWell(
                        onTap: () async {
                          const telegramUrl = 'https://t.me/YOUR_TELEGRAM_BOT';
                          await launchUrl(Uri.parse(telegramUrl));
                        },
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: const [
                            Icon(Icons.support_agent, color: Colors.blue),
                            SizedBox(width: 8),
                            Text(
                              'Поддержка в Telegram',
                              style: TextStyle(
                                color: Colors.blue,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                    Card(
                      elevation: 4,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 24,
                          vertical: 32,
                        ),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            CircleAvatar(
                              radius: 40,
                              backgroundColor: theme.colorScheme.primary
                                  .withAlpha((0.1 * 255).round()),
                              child: const Icon(
                                Icons.person,
                                size: 48,
                                color: Colors.deepPurple,
                              ),
                            ),
                            const SizedBox(height: 16),
                            Text(
                              _profile?['email'] ?? 'Email',
                              style: theme.textTheme.titleMedium?.copyWith(
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                            const SizedBox(height: 12),
                            SelectableText(
                              'UUID: ${_profile?['uuid'] ?? '-'}',
                              style: theme.textTheme.bodySmall,
                            ),
                            const SizedBox(height: 12),
                            Row(
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                Icon(
                                  hasAccess
                                      ? Icons.check_circle
                                      : Icons.error_outline,
                                  size: 18,
                                  color: hasAccess ? Colors.green : Colors.red,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  hasAccess
                                      ? 'Доступ активен'
                                      : 'Доступ отсутствует',
                                  style: theme.textTheme.bodyMedium,
                                ),
                              ],
                            ),
                            const SizedBox(height: 8),
                            Row(
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                const Icon(
                                  Icons.calendar_today,
                                  size: 18,
                                  color: Colors.blueGrey,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  'Триал до: $trialEndsAt',
                                  style: theme.textTheme.bodyMedium,
                                ),
                              ],
                            ),
                            const SizedBox(height: 4),
                            Row(
                              mainAxisAlignment: MainAxisAlignment.center,
                              children: [
                                const Icon(
                                  Icons.event_available,
                                  size: 18,
                                  color: Colors.blueGrey,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  'Подписка до: $expiresAt',
                                  style: theme.textTheme.bodyMedium,
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 32),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Expanded(
                          child: ElevatedButton.icon(
                            icon: const Icon(Icons.lock_reset),
                            style: ElevatedButton.styleFrom(
                              padding: const EdgeInsets.symmetric(vertical: 16),
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                            ),
                            onPressed: () {
                              Navigator.of(
                                context,
                              ).pushNamed('/change-password');
                            },
                            label: const Text('Сменить пароль'),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Expanded(
                          child: OutlinedButton.icon(
                            icon: const Icon(Icons.email),
                            style: OutlinedButton.styleFrom(
                              padding: const EdgeInsets.symmetric(vertical: 16),
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                            ),
                            onPressed: () async {
                              final emailController = TextEditingController(
                                text: _profile?['email'] ?? '',
                              );
                              final result = await showDialog<String>(
                                context: context,
                                builder: (ctx) => AlertDialog(
                                  title: const Text('Сброс пароля'),
                                  content: TextField(
                                    controller: emailController,
                                    decoration: const InputDecoration(
                                      labelText: 'Email',
                                    ),
                                    keyboardType: TextInputType.emailAddress,
                                  ),
                                  actions: [
                                    TextButton(
                                      onPressed: () => Navigator.pop(ctx),
                                      child: const Text('Отмена'),
                                    ),
                                    TextButton(
                                      onPressed: () => Navigator.pop(
                                        ctx,
                                        emailController.text,
                                      ),
                                      child: const Text('Отправить'),
                                    ),
                                  ],
                                ),
                              );
                              if (result != null && result.isNotEmpty) {
                                String? error;
                                String? success;
                                try {
                                  await AuthService.requestPasswordReset(
                                    result,
                                  );
                                  success =
                                      'Письмо для сброса пароля отправлено (если email зарегистрирован)';
                                  if (context.mounted)
                                    Navigator.of(
                                      context,
                                    ).pushNamed('/reset-password');
                                } catch (e) {
                                  error = e.toString().replaceAll(
                                    'Exception: ',
                                    '',
                                  );
                                }
                                if (context.mounted) {
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    SnackBar(
                                      content: Text(error ?? success ?? ''),
                                    ),
                                  );
                                }
                              }
                            },
                            label: const Text('Сброс по коду'),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Expanded(
                          child: ElevatedButton.icon(
                            icon: const Icon(Icons.logout),
                            style: ElevatedButton.styleFrom(
                              backgroundColor: Colors.deepPurple,
                              foregroundColor: Colors.white,
                              padding: const EdgeInsets.symmetric(vertical: 16),
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                            ),
                            onPressed: () async {
                              await AuthService.logout();
                              if (context.mounted) {
                                Navigator.of(context).pushNamedAndRemoveUntil(
                                  '/login',
                                  (route) => false,
                                );
                              }
                            },
                            label: const Text('Выйти'),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 12),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Expanded(
                          child: ElevatedButton.icon(
                            icon: const Icon(Icons.delete_forever),
                            style: ElevatedButton.styleFrom(
                              backgroundColor: Colors.red,
                              foregroundColor: Colors.white,
                              padding: const EdgeInsets.symmetric(vertical: 16),
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(12),
                              ),
                            ),
                            onPressed: () async {
                              final confirmed = await showDialog<bool>(
                                context: context,
                                builder: (ctx) => AlertDialog(
                                  title: const Text('Удалить профиль?'),
                                  content: const Text(
                                    'Вы уверены, что хотите удалить свой профиль? Это действие необратимо.',
                                  ),
                                  actions: [
                                    TextButton(
                                      onPressed: () =>
                                          Navigator.pop(ctx, false),
                                      child: const Text('Отмена'),
                                    ),
                                    TextButton(
                                      onPressed: () => Navigator.pop(ctx, true),
                                      child: const Text(
                                        'Удалить',
                                        style: TextStyle(color: Colors.red),
                                      ),
                                    ),
                                  ],
                                ),
                              );
                              if (confirmed == true) {
                                try {
                                  await AuthService.deleteAccount();
                                  await AuthService.logout();
                                  if (context.mounted) {
                                    Navigator.of(
                                      context,
                                    ).pushNamedAndRemoveUntil(
                                      '/login',
                                      (route) => false,
                                    );
                                  }
                                } catch (e) {
                                  if (context.mounted) {
                                    ScaffoldMessenger.of(context).showSnackBar(
                                      SnackBar(
                                        content: Text(
                                          'Ошибка удаления: ${e.toString()}',
                                        ),
                                      ),
                                    );
                                  }
                                }
                              }
                            },
                            label: const Text('Удалить профиль'),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
      ),
    );
  }
}
