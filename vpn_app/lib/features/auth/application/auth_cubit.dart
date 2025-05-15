import 'package:vpn_app/features/auth/data/auth_repository.dart';
import 'auth_state.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:injectable/injectable.dart';

@injectable
class AuthCubit extends Cubit<AuthState> {
  final AuthRepository authRepository;
  AuthCubit(this.authRepository) : super(AuthInitial());

  Future<void> login(String email, String password) async {
    emit(AuthLoading());
    final token = await authRepository.login(email, password);
    if (token != null) {
      emit(AuthAuthenticated(token));
    } else {
      emit(AuthError('Ошибка авторизации'));
    }
  }

  Future<void> register(String email, String password) async {
    emit(AuthLoading());
    final success = await authRepository.register(email, password);
    if (success) {
      emit(AuthRegistered());
    } else {
      emit(AuthError('Ошибка регистрации'));
    }
  }

  Future<void> logout() async {
    await authRepository.clearToken();
    emit(AuthInitial());
  }
}
