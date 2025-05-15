Базовый URL:
http://<ваш_сервер>:8081

---

Аутентификация:

1. Регистрация:
   POST /register
   Описание: Создает нового пользователя.
   Тело запроса:
   {
     "email": "test@example.com",
     "password": "password123"
   }
   Пример ответа:
   {
     "message": "User registered successfully"
   }

2. Вход:
   POST /login
   Описание: Аутентифицирует пользователя и возвращает токен.
   Тело запроса:
   {
     "email": "test@example.com",
     "password": "password123"
   }
   Пример ответа:
   {
     "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
   }

---

Пользователь:

1. Получение информации о текущем пользователе:
   GET /user/me
   Описание: Возвращает информацию о текущем пользователе.
   Заголовки:
   Authorization: Bearer <токен>
   Пример ответа:
   {
     "id": 10,
     "email": "test@example.com",
     "uuid": "4c6288c3-5b0b-4212-addb-1afd9aad9dfc",
     "tariff_id": 1,
     "traffic": 32768,
     "expires_at": "2025-12-31T23:59:59Z"
   }

2. Смена тарифа:
   POST /user/change-tariff
   Описание: Меняет тариф текущего пользователя.
   Заголовки:
   Authorization: Bearer <токен>
   Тело запроса:
   {
     "tariff_id": 2
   }
   Пример ответа:
   {
     "message": "Tariff changed successfully"
   }

3. Удаление аккаунта:
   POST /user/delete-account
   Описание: Удаляет аккаунт текущего пользователя.
   Заголовки:
   Authorization: Bearer <токен>
   Пример ответа:
   {
     "message": "Account deleted successfully"
   }

---

Тарифы:

1. Получение списка тарифов:
   GET /tariffs
   Описание: Возвращает список доступных тарифов.
   Пример ответа:
   [
     {
       "id": 1,
       "name": "Basic",
       "price": 10,
       "traffic_limit": 100000,
       "duration_days": 30
     },
     {
       "id": 2,
       "name": "Premium",
       "price": 20,
       "traffic_limit": 200000,
       "duration_days": 30
     }
   ]

2. Получение текущего тарифа пользователя:
   GET /user/tariff
   Описание: Возвращает информацию о текущем тарифе пользователя.
   Заголовки:
   Authorization: Bearer <токен>
   Пример ответа:
   {
     "id": 1,
     "name": "Basic",
     "price": 10,
     "traffic_limit": 100000,
     "duration_days": 30
   }

---

Трафик:

1. Получение текущего использования трафика:
   GET /user/traffic
   Описание: Возвращает информацию об использованном трафике.
   Заголовки:
   Authorization: Bearer <токен>
   Пример ответа:
   {
     "uplink": 16384,
     "downlink": 16384,
     "total": 32768
   }

2. Проверка лимитов трафика:
   GET /user/traffic/limits
   Описание: Возвращает информацию о лимитах трафика.
   Заголовки:
   Authorization: Bearer <токен>
   Пример ответа:
   {
     "traffic_used": 32768,
     "traffic_limit": 100000,
     "remaining_traffic": 67232
   }

---

Платежи:

1. Создание нового платежа:
   POST /user/payments
   Описание: Создает новый платеж.
   Заголовки:
   Authorization: Bearer <токен>
   Тело запроса:
   {
     "amount": 100,
     "tariff_id": 1,
     "payment_method": "credit_card"
   }
   Пример ответа:
   {
     "status": "payment created"
   }

2. Получение списка платежей:
   GET /user/payments
   Описание: Возвращает список всех платежей текущего пользователя.
   Заголовки:
   Authorization: Bearer <токен>
   Пример ответа:
   [
     {
       "id": 1,
       "user_id": 10,
       "amount": 100,
       "tariff_id": 1,
       "payment_method": "credit_card",
       "status": "pending",
       "created_at": "2025-05-07T12:00:00Z"
     }
   ]

3. Получение информации о конкретном платеже:
   GET /user/payments/{id}
   Описание: Возвращает информацию о конкретном платеже.
   Заголовки:
   Authorization: Bearer <токен>
   Пример ответа:
   {
     "id": 1,
     "user_id": 10,
     "amount": 100,
     "tariff_id": 1,
     "payment_method": "credit_card",
     "status": "pending",
     "created_at": "2025-05-07T12:00:00Z"
   }

4. Обновление статуса платежа:
   PUT /user/payments/{id}
   Описание: Обновляет статус платежа.
   Заголовки:
   Authorization: Bearer <токен>
   Тело запроса:
   {
     "status": "completed"
   }
   Пример ответа:
   {
     "status": "payment status updated"
   }

---

Мониторинг:

1. Проверка состояния сервера:
   GET /health
   Описание: Проверяет состояние сервера.
   Пример ответа:
   {
     "status": "ok"
   }
