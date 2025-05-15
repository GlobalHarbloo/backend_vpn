import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:injectable/injectable.dart';
import '../data/vpn_repository.dart';

abstract class VpnState {}

class VpnInitial extends VpnState {}

class VpnLoading extends VpnState {}

class VpnConnected extends VpnState {
  final String config;
  VpnConnected(this.config);
}

class VpnDisconnected extends VpnState {}

class VpnError extends VpnState {
  final String message;
  VpnError(this.message);
}

@injectable
class VpnCubit extends Cubit<VpnState> {
  final VpnRepository repository;
  VpnCubit(this.repository) : super(VpnInitial());

  Future<void> fetchConfig(String token) async {
    emit(VpnLoading());
    try {
      final config = await repository.fetchVpnConfig(token);
      emit(VpnConnected(config));
    } catch (e) {
      emit(VpnError(e.toString()));
    }
  }

  void disconnect() {
    emit(VpnDisconnected());
  }
}
