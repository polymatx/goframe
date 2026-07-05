# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2026-07-06

### Fixed

- Migrated `.golangci.yml` to the golangci-lint v2 configuration format
- Replaced deprecated `reflect.Ptr` with `reflect.Pointer` in the IoC container
- Documented intentional user-path writes in the CLI with `#nosec` justifications
  (gosec G702/G703 taint findings in `goframe new`/`build` scaffolding)
- Import formatting in `pkg/cache` and `pkg/rabbit`

### Changed

- CI actions updated: checkout v7, setup-go v6, golangci-lint-action v9, codecov v7

## [0.1.0] - 2026-07-06

First tagged release.

### Added

- Core framework: `app` (HTTP server, route groups, context, graceful shutdown),
  `middleware` (recovery, logging, CORS, gzip, per-IP rate limiting, Prometheus metrics)
- Authentication: JWT (issue/validate/refresh), Basic Auth, API keys (`pkg/auth`)
- Databases: PostgreSQL / MySQL / SQLite via GORM with named-connection registry (`pkg/database`)
- MongoDB, Redis (standalone + cluster), Elasticsearch, RabbitMQ, MQTT clients
  with the same `Register → Initialize → Get` lifecycle
- WebSocket support with hub pattern (`pkg/websocket`)
- IoC dependency-injection container with factories, singletons, and struct-tag injection (`pkg/container`)
- Configuration (Viper: YAML + env), structured logging (Logrus), validation helpers
- `goframe` CLI: `new` (go:embed standalone scaffolding), `gen model|handler|crud|middleware`,
  `serve` (hot reload), `build`, `migrate`
- 11 runnable examples, Docker Compose dev stack, CI (tests, lint, build, gosec)

### Changed

- **RabbitMQ driver migrated** from the archived `streadway/amqp` to the
  officially maintained `rabbitmq/amqp091-go`
- **Redis client upgraded** from `go-redis/redis/v8` to `redis/go-redis/v9`
- All dependencies updated to current versions (GORM 1.31, Viper, Prometheus client, JWT v5, ...)
- Minimum Go version is now **1.25**; CI tests against Go 1.25 and 1.26

[0.1.1]: https://github.com/polymatx/goframe/releases/tag/v0.1.1
[0.1.0]: https://github.com/polymatx/goframe/releases/tag/v0.1.0
