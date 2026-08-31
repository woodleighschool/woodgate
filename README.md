# woodgate

Internal check-in system with a Go API, a React admin console, and a native companion app for dedicated terminals.

Users and groups sync from Microsoft Entra. Administrators manage locations, permissions, check-ins, assets, and API keys from the web interface.

> [!WARNING]
> This project may be unstable or have bugs, use with caution.
> Also expect breaking changes between releases for now.

## 🚀 Usage

Copy the example configuration and start the published image with PostgreSQL:

```bash
cp .env.example .env
docker compose up -d
```

The web interface listens on [http://localhost:8080](http://localhost:8080). The container serves it from the Go backend.

The companion app is configured once by QR code with the server URL and an API key. It pairs to one location and runs as a dedicated check-in terminal. We distribute it as a private Custom App through Apple School Manager and our MDM.

## ⚙️ Configuration

| Variable               | Required          | Default or purpose                                  |
| ---------------------- | ----------------- | --------------------------------------------------- |
| `WOODGATE_PORT`        | No                | `8080`                                              |
| `WOODGATE_BASE_URL`    | Yes               | Public URL used for cookies and auth callbacks      |
| `WOODGATE_MEDIA_ROOT`  | No                | `media`; keep it on persistent storage              |
| `LOG_LEVEL`            | No                | `info`; accepts `debug`, `info`, `warn`, or `error` |
| `DATABASE_HOST`        | Yes               | PostgreSQL host                                     |
| `DATABASE_PORT`        | No                | `5432`                                              |
| `DATABASE_USER`        | Yes               | PostgreSQL user                                     |
| `DATABASE_PASSWORD`    | Yes               | PostgreSQL password                                 |
| `DATABASE_NAME`        | Yes               | PostgreSQL database                                 |
| `DATABASE_SSLMODE`     | No                | `disable`                                           |
| `JWT_SECRET`           | Yes               | Session signing secret                              |
| `LOCAL_ADMIN_PASSWORD` | One auth provider | Enables the local `admin` login                     |
| `ENTRA_TENANT_ID`      | One auth provider | Microsoft Entra tenant                              |
| `ENTRA_CLIENT_ID`      | With Entra        | Microsoft Entra client ID                           |
| `ENTRA_CLIENT_SECRET`  | With Entra        | Microsoft Entra client secret                       |
| `ENTRA_SYNC_ENABLED`   | No                | `false`                                             |
| `ENTRA_SYNC_INTERVAL`  | No                | `1h`                                                |

Use HTTPS in production, set a strong `JWT_SECRET`, and persist both PostgreSQL and `WOODGATE_MEDIA_ROOT`.

## 🔐 Permissions

Permissions grant a subject—user or API key—an action on a resource. Check-in permissions can be scoped to one location, and API keys should receive only the access their paired terminal needs.

## 🧑‍💻 Development

```bash
mise install
mise run deps
mise run dev
```

The root tasks cover the Go backend, web frontend, and development mock. Companion-app checks live under `//app:`:

```bash
mise run //app:fmt-check
mise run //app:lint
mise run //app:build
```

## 📦 Releases

The server and companion app have independent releases:

- Numeric releases such as `1.4.0` publish the server container through GitHub Actions.
- App releases such as `app-1.3.1` update `app/Config/Version.xcconfig` for distribution through App Store Connect.

See [`app/README.md`](app/README.md) for app development and release details.

## 📄 License

Licensed under the [Apache License 2.0](LICENSE).
