# sport-line-processor

Микросервис, который пуллит спортивные коэффициенты из другого сервиса, сохраняет их в базу данных и рассылает изменения коэффициентов подписчикам через gRPC API. Сделано с использованием Golang, Docker, MySQL, gRPC, GitHub Actions.


## Инструкции по запуску

- `make lint` — запуск статических анализаторов кода
- `make tests` — запуск юнит-тестов
- `make run` — запуск docker-compose с параметрами по умолчанию (уровень логирования, адрес сервиса и др.)
- `make stop` — остановка работы сервисов


## Параметры командной строки

- `--http` — адрес, по которому будет доступно HTTP API (ручка `\ready`).
- `--grpc` — адрес, по которому будет доступно gRPC API (ручка `\SubscribeOnSportLines`).
- `--provider` — адрес, по которому доступен `Lines Provider`.
- `--baseball`, `--football`, `--soccer` — интервалы (в секундах), с которыми будут пуллиться коэффициенты соответствущих спортов.
- `--log` — уровень логирования (debug, info, warn, error или fatal).


## Архитектура

### `Lines Provider`

Сервис, который отдает спортивные коэффициенты для бейсбола, американского футбола и футбола. Был изначально доступен в виде docker-контейнера `antonboom/lines-provider`

Пример работы с сервисом:

```
$ docker run -p8000:8000 antonboom/lines-provider
2020/07/24 08:44:55 listen and serve at :8000
$ curl http://localhost:8000/api/v1/lines/baseball
{"lines":{"BASEBALL":"0.774"}}
$ curl http://localhost:8000/api/v1/lines/football
{"lines":{"FOOTBALL":"0.721"}}
$ curl http://localhost:8000/api/v1/lines/soccer
{"lines":{"SOCCER":"1.667"}}
```

### `Sport Line Processor`

Микросервис, который:

- Пуллит спортивные коэффициенты из `Lines Provider`, используя отдельного воркера для каждого спорта. Каждый воркер пуллит свой спорт раз в N секунд (N для каждого воркера может быть разное, задается через флаги командной строки).
- Сохраняет их в хранилище (о нем ниже).
- После первой синхронизации коэффициентов готов принимать подписчиков (готовность можно проверить с помощью ручки `/ready`).
- Клиенты подписываются на изменения с помощью bidirectional streaming RPC (gRPC API метод `/SubscribeOnSportLines`). Параметры запроса клиента: список спортов и интервал ответа от сервера в секундах. Далее каждые M секунд клиент получает коэффициенты (в первом ответе) или их изменения (в последующих ответах) для выбранных спортов.

Пример общения через gRPC клиента и сервера:
```
Full-duplex stream:
10:00:00 > /SubscribeOnSportsLines [soccer, football] 3s
10:00:00 < {soccer: 1.13, football: 2.19}
10:00:03 < {soccer: 0.03, football: -0.01}

10:00:07 > /SubscribeOnSportsLines [soccer, football] 1s
10:00:07 < {soccer: 0.01, football: 0.02}
10:00:08 < {soccer: 0.02, football: -0.20}
10:00:09 < {soccer: -0.13, football: 0.03}

10:00:09 > /SubscribeOnSportsLines [baseball, football] 5s
10:00:09 < {baseball: 2.40, football: 1.98}
10:00:14 < {baseball: 0.12, football: 0.01}
10:00:19 < {baseball: -0.11, football: 0.03}
```

### `Хранилище для Sport Line Processor`

Реализовано с помощью MySQL базы данных, в которой хранятся только последние спуленные коэффициенты, так как исторические данные не используются.
