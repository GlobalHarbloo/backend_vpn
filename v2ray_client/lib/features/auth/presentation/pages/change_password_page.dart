import 'package:flutter/material.dart';
import '../../services/auth_service.dart';

class ChangePasswordPage extends StatefulWidget {
  const ChangePasswordPage({super.key});

  @override
  State<ChangePasswordPage> createState() => _ChangePasswordPageState();
}

class _ChangePasswordPageState extends State<ChangePasswordPage> {
  final _formKey = GlobalKey<FormState>();
  final _oldController = TextEditingController();
  final _newController = TextEditingController();
  final _confirmController = TextEditingController();
  bool _loading = false;
  String? _error;

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      await AuthService.changePassword(
        _oldController.text,
        _newController.text,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Пароль успешно изменён')));
      Navigator.of(context).pop();
    } catch (e) {
      setState(() {
        _error = e.toString().replaceAll('Exception: ', '');
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Смена пароля')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                TextFormField(
                  controller: _oldController,
                  decoration: const InputDecoration(
                    labelText: 'Текущий пароль',
                  ),
                  obscureText: true,
                  validator: (v) =>
                      v == null || v.isEmpty ? 'Введите текущий пароль' : null,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _newController,
                  decoration: const InputDecoration(labelText: 'Новый пароль'),
                  obscureText: true,
                  validator: (v) =>
                      v == null || v.length < 8 ? 'Минимум 8 символов' : null,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _confirmController,
                  decoration: const InputDecoration(
                    labelText: 'Подтвердите новый пароль',
                  ),
                  obscureText: true,
                  validator: (v) =>
                      v != _newController.text ? 'Пароли не совпадают' : null,
                ),
                const SizedBox(height: 16),
                if (_error != null)
                  Text(_error!, style: const TextStyle(color: Colors.red)),
                const SizedBox(height: 12),
                ElevatedButton(
                  onPressed: _loading ? null : _submit,
                  child: _loading
                      ? const CircularProgressIndicator(color: Colors.white)
                      : const Text('Изменить пароль'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
