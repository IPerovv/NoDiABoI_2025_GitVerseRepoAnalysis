## 1. Запуск и работа с приложением

### 1.1 Запуск
`docker-compose up -d`

### 1.2 Старт сбора датасета
`curl http://localhost:8080/run`

### 1.3 Проверка статуса сбора датасета
`curl http://localhost:8080/status`

### 1.4 Получение данных в csv формате (доступно только после сбора данных)
`curl http://localhost:8080/export/csv`

`docker cp app_container:/app_binary/dataset/tables ./dataset/`

### 1.5 Просмотр датасета в БД
`docker exec -it mongodb /bin/bash`

`mongo`

`use golang`

#### 1.5.1 Просмотр количества записей в БД

`db.repos.count()`

#### 1.5.2 Просмотр записей в БД

`db.repos.find().pretty()`

### 1.6 Дамп данных
`docker exec -it mongodb mongodump --uri="mongodb://localhost:27017/golang" --out=/data/dump`

`docker cp mongodb:/data/dump ./mongo_dump`

### 1.7 Получение данных в json формате
`docker exec -it mongodb mongoexport --uri="mongodb://localhost:27017/golang" --collection=repos --out=/data/repos.json --jsonArray`

`docker cp mongodb:/data/repos.json ./repos.json`

## 2. Структура датасета

Список записей вида:

| Поле              | Назначение           |
|-------------------|----------------------|
| `id`              | Уникальный ID        |
| `full_name`       | `owner/repo`         |
| `created_at`      | Когда создан         |
| `updated_at`      | Последняя активность |
| `archived`        | Был ли заархивирован |
| `stars_count`     | Количество звёзд     |
| `size`            | Размер репозитория   |
| `release_counter` | Количество релизов   |
| `tag_count`       | Количество тегов     |

## 3. Часть реальных данных из датасета

```json
[
  {
    "_id": {
      "$oid": "68e814d1bc4b327de5d2ac6a"
    },
    "id": 6144,
    "archived": false,
    "createdat": {
      "$date": "2024-03-10T06:57:02Z"
    },
    "fullname": "rnekrasov/superknowa",
    "releasecounter": 0,
    "size": 89171,
    "starscount": 1,
    "tagcount": 2,
    "updatedat": {
      "$date": "2024-03-10T06:57:09Z"
    }
  },
  {
    "_id": {
      "$oid": "68e814d1bc4b327de5d2ac6b"
    },
    "id": 124882,
    "archived": false,
    "createdat": {
      "$date": "2025-08-14T17:29:15Z"
    },
    "fullname": "Cifrasmm/test",
    "releasecounter": 0,
    "size": 22,
    "starscount": 0,
    "tagcount": 0,
    "updatedat": {
      "$date": "2025-08-14T17:29:16Z"
    }
  },
  {
    "_id": {
      "$oid": "68e814d1bc4b327de5d2ac6c"
    },
    "id": 108990,
    "archived": false,
    "createdat": {
      "$date": "2025-05-28T14:21:36Z"
    },
    "fullname": "slava3110/newTest",
    "releasecounter": 0,
    "size": 38,
    "starscount": 0,
    "tagcount": 0,
    "updatedat": {
      "$date": "2025-05-28T14:26:32Z"
    }
  },
  {
    "_id": {
      "$oid": "68e814d1bc4b327de5d2ac6d"
    },
    "id": 101996,
    "archived": false,
    "createdat": {
      "$date": "2025-04-22T14:44:46Z"
    },
    "fullname": "persoq4/DZ_OOP",
    "releasecounter": 0,
    "size": 54,
    "starscount": 0,
    "tagcount": 0,
    "updatedat": {
      "$date": "2025-04-22T15:19:33Z"
    }
  }
]
```
