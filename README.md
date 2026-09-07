# WoodGate 🚪

[![Release](https://img.shields.io/github/v/release/woodleighschool/woodgate?display_name=tag&sort=semver)](https://github.com/woodleighschool/woodgate/releases/latest)
[![CI](https://github.com/woodleighschool/woodgate/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/woodleighschool/woodgate/actions/workflows/ci.yaml)
[![Go](https://img.shields.io/github/go-mod/go-version/woodleighschool/woodgate?logo=go)](https://github.com/woodleighschool/woodgate/blob/main/go.mod)
[![Container](https://img.shields.io/badge/container-ghcr.io-2496ED?logo=github&logoColor=white)](https://github.com/orgs/woodleighschool/packages/container/package/woodgate)
[![License](https://img.shields.io/github/license/woodleighschool/woodgate)](https://github.com/woodleighschool/woodgate/blob/main/LICENSE)

Manages check-ins with a web administration interface and a native companion app for dedicated terminals.

> [!WARNING]
> This project may be unstable or have bugs, use with caution.
> Also expect breaking changes between releases for now.

## 🌱 What's inside

- Check-in history, notes, photos, and exports
- Locations, rosters, and terminal branding
- Entra directory sync for people and groups
- Staff sign-in and resource-based roles
- Location-scoped app keys for companion terminals
- File or S3 storage for resource attachments

## 🚀 Usage

Use the container `ghcr.io/woodleighschool/woodgate:rolling`.

### Docker Compose

Start from the example environment:

```bash
cp .env.example .env
openssl rand -hex 32
```

Before starting, set these values in `.env`:

| Variable                          | Value                                             |
| --------------------------------- | ------------------------------------------------- |
| `WOODGATE_URL`                    | HTTPS address used by browsers and companion apps |
| `WOODGATE_STORAGE_CAPABILITY_KEY` | Output from `openssl rand -hex 32`                |

```bash
docker compose up -d
docker compose exec woodgate /woodgate user create \
  --email you@example.com \
  --name "Your Name" \
  --role admin
```

The server listens on port 8080. Terminate HTTPS at the reverse proxy. For local HTTP development, set `WOODGATE_SESSION_COOKIE_SECURE=false`.

Pair the companion app with a QR code containing the server URL and an app key, then select a permitted location. We distribute it as a private Custom App through Apple School Manager and our MDM.

## ⚙️ Configuration

The [configuration reference](docs/configuration.md) covers authentication, directory sync, storage, and HTTP settings. The [companion guide](app/README.md) covers pairing, the review mock, and app development.

## 🧑‍💻 Development

Mise owns the toolchain and commands:

```bash
mise install
mise run deps
mise run dev
mise run test
mise run lint
```

Backend code lives under `cmd/woodgate` and `internal/`; the React app lives under `web/`; the native companion lives under `app/`; the review mock lives under `mock/`.

Run `mise run generate` after changing the API contract. PostgreSQL component tests use `mise run test-postgres` with `WOODGATE_TEST_DATABASE_URL` pointing at a local test server. Companion checks use `mise run //app:build` and `mise run //app:lint`.

## 📦 Releases

The server and companion app have independent releases:

- Numeric releases such as `1.4.0` publish the server container through GitHub Actions.
- App releases such as `app-1.3.1` update `app/Config/Version.xcconfig` for distribution through App Store Connect.

See the [companion guide](app/README.md) for app release details.

## 📄 License

Licensed under the [Apache License 2.0](LICENSE).
