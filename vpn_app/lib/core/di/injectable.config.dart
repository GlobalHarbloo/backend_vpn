// GENERATED CODE - DO NOT MODIFY BY HAND

// **************************************************************************
// InjectableConfigGenerator
// **************************************************************************

// ignore_for_file: type=lint
// coverage:ignore-file

// ignore_for_file: no_leading_underscores_for_library_prefixes
import 'package:flutter_secure_storage/flutter_secure_storage.dart' as _i558;
import 'package:get_it/get_it.dart' as _i174;
import 'package:injectable/injectable.dart' as _i526;
import 'package:vpn_app/core/di/di_module.dart' as _i513;
import 'package:vpn_app/core/network/api_client.dart' as _i1051;
import 'package:vpn_app/features/auth/application/auth_cubit.dart' as _i158;
import 'package:vpn_app/features/auth/data/auth_repository.dart' as _i578;
import 'package:vpn_app/features/payments/application/payments_cubit.dart'
    as _i986;
import 'package:vpn_app/features/payments/data/payments_repository.dart'
    as _i781;
import 'package:vpn_app/features/profile/application/profile_cubit.dart'
    as _i784;
import 'package:vpn_app/features/profile/data/profile_repository.dart'
    as _i1008;
import 'package:vpn_app/features/traffic/application/traffic_cubit.dart'
    as _i164;
import 'package:vpn_app/features/traffic/data/traffic_repository.dart' as _i929;
import 'package:vpn_app/features/vpn/application/vpn_cubit.dart' as _i655;
import 'package:vpn_app/features/vpn/data/vpn_repository.dart' as _i479;

extension GetItInjectableX on _i174.GetIt {
// initializes the registration of main-scope dependencies inside of GetIt
  _i174.GetIt init({
    String? environment,
    _i526.EnvironmentFilter? environmentFilter,
  }) {
    final gh = _i526.GetItHelper(
      this,
      environment,
      environmentFilter,
    );
    final diModule = _$DiModule();
    gh.lazySingleton<_i558.FlutterSecureStorage>(() => diModule.storage);
    gh.lazySingleton<_i1051.ApiClient>(
        () => _i1051.ApiClient(gh<_i558.FlutterSecureStorage>()));
    gh.singleton<_i578.AuthRepository>(
        () => _i578.AuthRepository(gh<_i1051.ApiClient>()));
    gh.singleton<_i781.PaymentsRepository>(
        () => _i781.PaymentsRepository(gh<_i1051.ApiClient>()));
    gh.singleton<_i1008.ProfileRepository>(
        () => _i1008.ProfileRepository(gh<_i1051.ApiClient>()));
    gh.singleton<_i929.TrafficRepository>(
        () => _i929.TrafficRepository(gh<_i1051.ApiClient>()));
    gh.singleton<_i479.VpnRepository>(
        () => _i479.VpnRepository(gh<_i1051.ApiClient>()));
    gh.factory<_i986.PaymentsCubit>(
        () => _i986.PaymentsCubit(gh<_i781.PaymentsRepository>()));
    gh.factory<_i784.ProfileCubit>(
        () => _i784.ProfileCubit(gh<_i1008.ProfileRepository>()));
    gh.factory<_i655.VpnCubit>(() => _i655.VpnCubit(gh<_i479.VpnRepository>()));
    gh.factory<_i158.AuthCubit>(
        () => _i158.AuthCubit(gh<_i578.AuthRepository>()));
    gh.factory<_i164.TrafficCubit>(
        () => _i164.TrafficCubit(gh<_i929.TrafficRepository>()));
    return this;
  }
}

class _$DiModule extends _i513.DiModule {}
