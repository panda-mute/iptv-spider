# Repository Guidelines

## Project Structure & Module Organization

- `main.go`: Application entrypoint initializing logger, configuration, databases, and HTTP routes.
- `initialize/`: Startup bootstrappers (`viper`, `zap`, `gorm`, `cache`, `cron`, `oss`, `panel`).
- `modules/`: Core domain engines:
  - `panel/`: Playlist management, channel matching, EPG parsing, and logo handling.
  - `spider/`: Upstream IPTV stream discovery and playback crawling.
  - `auth/`, `m3u/`, `jsvm/`, `http_client/`: Supporting protocol and execution modules.
- `router/`: Iris web framework router and API endpoints (`router/api`).
- `model/`: Data models, channel definitions, XMLTV structures, and database entities.
- `config/`: Configuration structs mapped from `config.yaml`.
- `utils/`: Crypto (AES, RSA), formatting, OSS, and helper utilities.

## Build, Test, and Development Commands

- `make build`: Compiles the binary to `bin/iptv-spider` with `-trimpath`.
- `make test`: Executes all unit tests with the race detector (`go test -race ./...`).
- `make run`: Launches the application directly in development mode (`go run .`).
- `docker compose up -d`: Runs the service containerized using `compose.yaml`.

## Coding Style & Naming Conventions

- **Formatting**: Adhere to standard Go guidelines; format code with `gofmt` using tabs for indentation.
- **Naming**: Use `CamelCase` for unexported and `PascalCase` for exported identifiers. Keep package names concise, lowercase, and free of underscores.
- **Error Handling & Logging**: Return errors explicitly rather than panicking. Use structured logging via `global.GVA_LOG` (`zap`).
- **Configuration**: Expose configurable values through `config/` structs and `config.yaml` rather than hardcoded constants.

## Testing Guidelines

- **Conventions**: Place unit tests alongside source code with `_test.go` suffixes (e.g., `router_test.go`, `utils_test.go`).
- **Execution**: Run `go test -race ./...` before submitting changes.
- **Isolation**: Keep tests self-contained and mock external network or database calls where possible (see `modules/panel/test_helper_test.go`).

## Commit & Pull Request Guidelines

- **Commit Messages**: Follow Conventional Commits format, matching repository history (e.g., `feat: initial commit for iptv-spider`, `fix: resolve channel resolution fallback`).
- **Pull Requests**: Provide a concise summary of changes, reference related issues, ensure `make test` and `make build` pass, and note any required schema updates in `config.example.yaml`.
