// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$UserImpl _$$UserImplFromJson(Map<String, dynamic> json) => _$UserImpl(
      id: (json['id'] as num).toInt(),
      email: json['email'] as String,
      uuid: json['uuid'] as String?,
      tariffId: (json['tariff_id'] as num?)?.toInt(),
      traffic: (json['traffic'] as num?)?.toInt(),
      expiresAt: _dateTimeFromJson(json['expires_at']),
    );

Map<String, dynamic> _$$UserImplToJson(_$UserImpl instance) =>
    <String, dynamic>{
      'id': instance.id,
      'email': instance.email,
      'uuid': instance.uuid,
      'tariff_id': instance.tariffId,
      'traffic': instance.traffic,
      'expires_at': instance.expiresAt?.toIso8601String(),
    };
