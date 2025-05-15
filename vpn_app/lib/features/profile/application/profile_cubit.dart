import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:injectable/injectable.dart';
import '../data/profile_repository.dart';
import '../../auth/domain/models/user.dart';
import '../../../core/di/injectable.dart';
import '../../../core/network/api_client.dart';
import 'profile_state.dart';

@injectable
class ProfileCubit extends Cubit<ProfileState> {
  final ProfileRepository repository;
  ProfileCubit(this.repository) : super(ProfileInitial());

  Future<void> loadProfile(String token) async {
    emit(ProfileLoading());
    try {
      final data = await repository.fetchProfile(token);
      final user = User.fromJson(data);
      emit(ProfileLoaded(user));
    } catch (e) {
      emit(ProfileError(e.toString()));
    }
  }

  Future<void> loadProfileTokenless() async {
    emit(ProfileLoading());
    try {
      final apiClient = getIt<ApiClient>();
      final token = await apiClient.getToken();
      if (token == null) throw Exception("Token not found");
      final data = await repository.fetchProfile(token);
      final user = User.fromJson(data);
      emit(ProfileLoaded(user));
    } catch (e) {
      emit(ProfileError(e.toString()));
    }
  }

  void logout() {
    emit(ProfileInitial());
  }
}
