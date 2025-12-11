# ANTIPLAGUE

**Antiplague** — это система для автоматической проверки студенческих работ на плагиат. Система состоит из трёх микросервисов, которые взаимодействуют через API Gateway, обеспечивая:

- Загрузку исходного кода в различных форматах  
- Автоматический анализ на плагиат после загрузки  
- Детальные отчёты о сходстве между работами  
- Визуализацию частоты слов через облако слов  
- RESTful API с полной документацией (Swagger/OpenAPI)  
- Полная контейнеризация через Docker
---

## Структура проекта

```
antiplague/
├── api-gateway/                    # Сервис API Gateway
│   ├── main.go                     # Точка входа, маршрутизация запросов
│   └── Dockerfile                  # Контейнеризация Gateway
│
├── file-storing-service/           # Сервис хранения файлов
│   ├── main.go                     # Логика загрузки и хранения файлов
│   ├── files.db                    # SQLite БД (создаётся автоматически)
│   ├── uploads/                    # Папка для хранения загруженных файлов
│   └── Dockerfile                  # Контейнеризация Storing Service
│
├── file-analysis-service/          # Сервис анализа на плагиат
│   ├── main.go                     # Логика анализа и сравнения файлов
│   └── Dockerfile                  # Контейнеризация Analysis Service
│
├── docker-compose.yml              # Конфигурация Docker Compose
├── swagger.yaml                    # OpenAPI 3.0 документация
└── README.md

```

### Порты:

| Сервис                    | Порт    | Назначение                        |
|---------------------------|---------|-----------------------------------|
| **API Gateway**           | 8080    | Основная точка входа для клиентов |
| **File Analysis Service** | 8081    | Анализ и отчёты по плагиату       |
| **File Storing Service**  | 8082    | Загрузка и получение файлов       |
| **Swagger UI**            | 8083    | Интерактивная документация API    |

---

## Запуск и остановка программы

### Вариант 1: Запуск через Docker

```bash
git clone https://github.com/milvus151/antiplague.git
cd antiplague

docker compose up --build
```

Система запустится, создаст базу данных, настроит сеть контейнеров.

После этого система работает на:
- **http://localhost:8080** — API Gateway
- **http://localhost:8083** — Swagger UI (документация)

### Вариант 2: Остановка системы

```bash
# Остановить контейнеры (сохраняет данные)
docker compose down

docker compose down -v
```

---

## Архитектура системы

### Общая структура

```
┌─────────────────────────────────────────────────────────────┐
│                      Клиент (Postman/Browser)               │
└────────────────────────────┬────────────────────────────────┘
                             │ HTTP Request
                             ↓
┌─────────────────────────────────────────────────────────────┐
│            API GATEWAY (127.0.0.1:8080)                     │
│  - Единая точка входа для всех запросов                     │
│  - Маршрутизирует запросы к микросервисам                   │
│  - Запускает асинхронный анализ при загрузке файла          │
└──────────┬──────────────────────────────┬───────────────────┘
           │                              │
           ↓                              ↓
┌──────────────────────────┐   ┌──────────────────────────┐
│ FILE STORING SERVICE     │   │ FILE ANALYSIS SERVICE    │
│ (127.0.0.1:8082)         │   │ (127.0.0.1:8081)         │
│                          │   │                          │
│ Функции:                 │   │ Функции:                 │
│ - POST /upload           │   │ - POST /analyze          │
│ - GET /files             │   │ - GET /reports           │
│ - GET /files/{id}        │   │ - GET /reports/{id}      │
│                          │   │ - GET /wordCloud/{id}    │
│                          │   │                          │
│ БД: SQLite files.db      │   │ БД: SQLite (shared)      │
│ Хранилище: uploads/      │   │ Алгоритм: Сравнение слов │
└──────────────────────────┘   └──────────────────────────┘
           ↕                              ↕
     Одна общая SQLite база данных + общая папка uploads
```

### Микросервисы

#### **API Gateway** (`api-gateway/main.go`)

**Ответственность:** Центральный маршрутизатор, обработка HTTP-запросов, координация микросервисов.

**Основные функции:**
- Принимает запросы от клиентов на порту 8080
- Проксирует запросы к File Storing Service и File Analysis Service
- **Уникальная функция:** При загрузке файла (`POST /upload`) автоматически запускает анализ в фоновой горутине
- Возвращает полный ответ с информацией о загруженном файле и статусом анализа

**Ключевые эндпоинты:**
```
POST   /upload              → File Storing Service
GET    /files               → File Storing Service
GET    /files/{id}          → File Storing Service
GET    /analyze             → File Analysis Service (direct)
GET    /reports             → File Analysis Service
GET    /reports/{id}        → File Analysis Service
GET    /wordCloud/{id}      → File Analysis Service
```

#### **File Storing Service** (`file-storing-service/main.go`)

**Ответственность:** Хранение, управление и выдача файлов работ.

**Основные функции:**
- Сохранение загруженных файлов на диск (`/app/uploads/`)
- Регистрация метаданных файлов в БД (студент, задание, время загрузки)
- Выдача списка всех загруженных файлов
- Получение деталей конкретного файла по ID

**Механизм сохранения:**
- Файлы сохраняются с именем формата: `work_{student_id}_{assignment_id}_{timestamp}.{ext}`
- Пример: `work_std_0013_task-001_1733867227.py`
- Поддерживаемые форматы: `.txt`, `.go`, `.py`, `.java`, `.cpp`, `.c`, `.h`, `.js`, `.ts`, `.md`

**Таблица БД `files`:**
```sql
CREATE TABLE files (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    student_id      TEXT NOT NULL,
    assignment_id   TEXT NOT NULL,
    file_path       TEXT NOT NULL,
    uploaded_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    status          TEXT DEFAULT 'pending'
);
```

#### **File Analysis Service** (`file-analysis-service/main.go`)

**Ответственность:** Анализ файлов на плагиат, генерация отчётов, визуализация.

**Основные функции:**
- Получает путь к загруженному файлу
- Сравнивает содержимое файла со всеми **другими файлами этого же задания**
- Создаёт отчёт о результатах анализа
- Интеграция с QuickChart API для создания облака слов

**Таблица БД `reports`:**
```sql
CREATE TABLE reports (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id         INTEGER NOT NULL,
    plagiarism_score REAL NOT NULL,
    is_plagiarism   BOOLEAN NOT NULL,
    matched_file_id INTEGER NOT NULL,
    analysis_state  TEXT NOT NULL,
    same_details    TEXT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## API Endpoints

### Health Check

#### `GET /health`
Проверка статуса API Gateway.

**Response (200 OK):**
```json
{
    "status": "OK",
    "message": "API Gateway is running"
}
```

---

### Работа с файлами (File Storing)

#### `POST /upload`
Загрузка новой работы на проверку.

**Content-Type:** `multipart/form-data`

**Parameters:**

| Параметр        | Тип    | Обязательный | Описание                           |
|-----------------|--------|--------------|------------------------------------|
| `student_id`    | string | Да           | ID студента (например: `std_0013`) |
| `assignment_id` | string | Да           | ID задания (например: `task-001`)  |
| `file`          | file   | Да           | Сам файл                           |

**Response (200 OK):**
```json
{
    "status": "success",
    "message": "Файл получен, работа зарегистрирована",
    "file_id": 15,
    "student_id": "std_0013",
    "assignment_id": "task-001",
    "filename": "solution.py",
    "file_path": "/app/uploads/work_std_0013_task-001_1733867227.py",
    "analysis_status": "started"
}
```

**Что происходит:**
1. Файл сохраняется на диск
2. Информация о файле записывается в БД
3. **Автоматически** запускается анализ в фоновой горутине
4. Клиент немедленно получает ответ (не дожидается окончания анализа)

---

#### `GET /files`
Получить список всех загруженных файлов.

**Response (200 OK):**
```json
[
    {
        "id": "1",
        "student_id": "std_0001",
        "assignment_id": "task-001",
        "file_path": "/app/uploads/work_std_0001_task-001_1733867000.py",
        "uploaded_at": "2024-12-10T15:30:27Z",
        "status": "pending"
    },
    {
        "id": "2",
        "student_id": "std_0002",
        "assignment_id": "task-001",
        "file_path": "/app/uploads/work_std_0002_task-001_1733867100.py",
        "uploaded_at": "2024-12-10T15:35:00Z",
        "status": "completed"
    }
]
```

---

#### `GET /files/{id}`
Получить информацию о конкретном файле.

**Parameters:**

| Параметр | Тип            | Описание |
|----------|----------------|----------|
| `id`     | integer (path) | ID файла |

**Response (200 OK):**
```json
{
    "id": "15",
    "student_id": "std_0013",
    "assignment_id": "task-001",
    "file_path": "/app/uploads/work_std_0013_task-001_1733867227.py",
    "uploaded_at": "2024-12-10T16:10:27Z",
    "status": "completed"
}
```

---

### Анализ на плагиат

#### `POST /analyze` (Direct)
Прямой вызов анализа файла (обычно используется внутри системы, но доступен и напрямую).

**Content-Type:** `application/json`

**Request Body:**
```json
{
    "file_id": 15,
    "file_path": "/app/uploads/work_std_0013_task-001_1733867227.py",
    "student_id": "std_0013",
    "assignment_id": "task-001"
}
```

**Response (200 OK):**
```json
{
    "id": 5,
    "file_id": 15,
    "plagiarism_score": 0.52,
    "is_plagiarism": true,
    "matched_file_id": 12,
    "analysis_state": "completed",
    "same_details": "Совпадение 52.00% с File ID 12"
}
```

---

### Отчёты по плагиату

#### `GET /reports`
Получить все отчёты по плагиату.

**Response (200 OK):**
```json
[
    {
        "id": 1,
        "file_id": 1,
        "plagiarism_score": 0.02,
        "is_plagiarism": false,
        "matched_file_id": 0,
        "analysis_state": "completed",
        "same_details": "Совпадение 2.00% с File ID 0"
    },
    {
        "id": 2,
        "file_id": 2,
        "plagiarism_score": 0.85,
        "is_plagiarism": true,
        "matched_file_id": 1,
        "analysis_state": "completed",
        "same_details": "Совпадение 85.00% с File ID 1"
    }
]
```

---

#### `GET /reports/{id}`
Получить отчёт по конкретному файлу.

**Parameters:**

| Параметр  | Тип            | Описание  |
|-----------|----------------|-----------|
| `id`      | integer (path) | ID отчёта |

**Response (200 OK):**
```json
{
    "id": 5,
    "file_id": 15,
    "plagiarism_score": 0.52,
    "is_plagiarism": true,
    "matched_file_id": 12,
    "analysis_state": "completed",
    "same_details": "Совпадение 52.00% с File ID 12"
}
```

---

### Визуализация (Облако слов)

#### `GET /wordCloud/{id}`
Получить облако слов для файла (PNG изображение).

**Parameters:**

| Параметр | Тип            | Описание |
|----------|----------------|----------|
| `id`     | integer (path) | ID файла |

**Response (200 OK):**
- **Content-Type:** `image/png`
- **Тело ответа:** PNG изображение облака слов

**Для чего:**
- Визуализация частоты слов в файле
- Чем больше слово на облаке, тем чаще оно встречается в коде
- Полезно для быстрого анализа содержания работы

**Пример использования:**
```html
<img src="http://localhost:8080/wordCloud/15" alt="Word Cloud">
```

---

## Алгоритм определения плагиата

### Общая идея

Система использует алгоритм **сравнения наборов слов** для определения сходства между работами. Это выполняется на уровне **одного задания** — работы для `task-001` сравниваются только с другими работами `task-001`, и не сравниваются с `task-002`.

### Пошаговый процесс

#### Шаг 1: Препроцессинг

Когда файл загружается:
1. Проверяется расширение файла (поддерживаемые: `.txt`, `.go`, `.py`, `.java`, `.cpp`, `.c`, `.h`, `.js`, `.ts`, `.md`)
2. Содержимое файла читается в памяти
3. Текст преобразуется в нижний регистр
4. Текст разбивается на слова (по пробелам)

#### Шаг 2: Выборка файлов для сравнения

Из БД выбираются **все файлы**:
- **Другие студенты** (исключаются работы текущего студента)
- **Одного и того же задания** (например, если загружена работа для `task-001`, берутся только файлы с `assignment_id = "task-001"`) (мне кажется так логичнее)
- **Исключается текущий файл** (не сравниваем файл с самим собой)

SQL запрос:
```sql
SELECT id, file_path, student_id
FROM files
WHERE id != ?
AND student_id != ?
AND assignment_id = ?
ORDER BY id ASC
```

#### Шаг 3: Сравнение по словам

Для каждого файла из выборки:
1. Читается его содержимое
2. Разбивается на слова (аналогично препроцессингу)
3. Подсчитывается количество **совпадающих слов**
4. Вычисляется **коэффициент сходства**

**Формула сходства:**
```
Similarity = (Количество совпадающих слов) / max(Количество слов в файле 1, Количество слов в файле 2)
```

#### Шаг 4: Выбор максимального сходства

Из всех сравнений выбирается **максимальное значение сходства**:
- Этот процент записывается в отчёт как `plagiarism_score`
- ID файла с максимальным сходством записывается как `matched_file_id`

#### Шаг 5: Определение факта плагиата

```
if plagiarism_score > 0.5 (50%):
    is_plagiarism = true
else:
    is_plagiarism = false
```

Порог **0.5** выбран, чтобы минимизировать ложные срабатывания (совпадение обычных ключевых слов языка программирования).

## Тестирование и проверка

### Способ 1: Через Swagger UI (Интерактивно)

Самый удобный способ для быстрой проверки
```
https://localhost:8083
```

### Способ 2: Через Postman (Программно)

Здесь либо импортировать `swagger.yaml` и потом работать с коллекцией, либо писать все запросы вручную.


## Использованные технологии

| Компонент                 | Технология                        | Версия |
|---------------------------|-----------------------------------|--------|
| **Язык программирования** | Go                                | 1.21+  |
| **Веб-фреймворк**         | Стандартная библиотека `net/http` | —      |
| **База данных**           | SQLite                            | 3.x    |
| **Контейнеризация**       | Docker + Docker Compose           | 4.0+   |
| **API документация**      | Swagger/OpenAPI                   | 3.0.0  |
| **Визуализация слов**     | QuickChart API                    | —      |

---

## Соответствие требованиям задания

| Критерий                                 | Статус | Реализация                                                                                |
|------------------------------------------|--------|-------------------------------------------------------------------------------------------|
| **1. Основные требования**               | +      | Микросервисная архитектура с анализом плагиата и автоматической проверкой после загрузки  |
| **2a. Минимум 2 микросервиса + Gateway** | +      | File Storing Service + File Analysis Service + API Gateway                                |
| **2b. Обработка ошибок**                 | +      | Graceful degradation, HTTP error codes, логирование ошибок                                |
| **3a. Dockerfile и docker-compose.yml**  | +      | Все сервисы упакованы, один файл `docker-compose.yml`                                     |
| **3b. Docker контейнеры**                | +      | Все 3 микросервиса + Swagger UI в контейнерах                                             |
| **3c. Запуск через `docker compose up`** | +      | Одна команда, система полностью разворачивается                                           |
| **4. Swagger/Postman коллекция**         | +      | Интерактивная документация на http://localhost:8083                                       |
| **5a. Качество кода**                    | +      | Модульный, структурированный, с обработкой ошибок                                         |
| **5b. Архитектура и сценарии**           | +      | Подробно описаны в этом README                                                            |
| **6. Облако слов**                       | +      | Интеграция с QuickChart API, эндпоинт `/wordCloud/{id}`                                   |

---
