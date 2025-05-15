import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:injectable/injectable.dart';
import '../data/traffic_repository.dart';

abstract class TrafficState {}

class TrafficInitial extends TrafficState {}

class TrafficLoading extends TrafficState {}

class TrafficLoaded extends TrafficState {
  final int uplink;
  final int downlink;
  final int total;
  TrafficLoaded({
    required this.uplink,
    required this.downlink,
    required this.total,
  });
}

class TrafficError extends TrafficState {
  final String message;
  TrafficError(this.message);
}

@injectable
class TrafficCubit extends Cubit<TrafficState> {
  final TrafficRepository repository;
  TrafficCubit(this.repository) : super(TrafficInitial());

  Future<void> loadTraffic(String token) async {
    emit(TrafficLoading());
    try {
      final data = await repository.fetchTrafficStats(token);
      emit(
        TrafficLoaded(
          uplink:
              data['uplink'] is int
                  ? data['uplink']
                  : int.tryParse(data['uplink']?.toString() ?? '') ?? 0,
          downlink:
              data['downlink'] is int
                  ? data['downlink']
                  : int.tryParse(data['downlink']?.toString() ?? '') ?? 0,
          total:
              data['total'] is int
                  ? data['total']
                  : int.tryParse(data['total']?.toString() ?? '') ?? 0,
        ),
      );
    } catch (e) {
      emit(TrafficError(e.toString()));
    }
  }
}
