import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:injectable/injectable.dart';
import '../data/payments_repository.dart';

abstract class PaymentsState {}

class PaymentsInitial extends PaymentsState {}

class PaymentsLoading extends PaymentsState {}

class PaymentsLoaded extends PaymentsState {
  final List<dynamic> payments;
  PaymentsLoaded(this.payments);
}

class PaymentsError extends PaymentsState {
  final String message;
  PaymentsError(this.message);
}

@injectable
class PaymentsCubit extends Cubit<PaymentsState> {
  final PaymentsRepository repository;
  PaymentsCubit(this.repository) : super(PaymentsInitial());

  Future<void> loadPayments(String token) async {
    emit(PaymentsLoading());
    try {
      final payments = await repository.fetchPayments(token);
      emit(PaymentsLoaded(payments));
    } catch (e) {
      emit(PaymentsError(e.toString()));
    }
  }

  Future<void> createPayment(
    String token,
    int amount,
    int tariffId,
    String paymentMethod,
  ) async {
    emit(PaymentsLoading());
    try {
      final success = await repository.createPayment(
        token,
        amount,
        tariffId,
        paymentMethod,
      );
      if (success) {
        await loadPayments(token);
      } else {
        emit(PaymentsError('Ошибка при создании платежа'));
      }
    } catch (e) {
      emit(PaymentsError(e.toString()));
    }
  }

  Future<void> changeTariff(String token, int tariffId) async {
    emit(PaymentsLoading());
    try {
      final success = await repository.changeTariff(token, tariffId);
      if (success) {
        await loadPayments(token);
      } else {
        emit(PaymentsError('Ошибка при смене тарифа'));
      }
    } catch (e) {
      emit(PaymentsError(e.toString()));
    }
  }
}
