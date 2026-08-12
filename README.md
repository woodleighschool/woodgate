# woodgate

Internal check-in system with a Go API, a React admin console, and a native companion app for dedicated terminals.

Users and groups sync from Microsoft Entra. Administrators manage locations, permissions, check-ins, assets, and API keys from the web interface.

> [!WARNING]
> The API and configuration may change between releases.

## 🚀 Usage

Copy the example configuration and start the published image with PostgreSQL:

```bash
cp .env.example .env
docker compose up -d
```

The web interface listens on [http://localhost:8080](http://localhost:8080). The container serves it from the Go backend.

The companion app is configured once by QR code with the server URL and an API key. It pairs to one location and runs as a dedicated check-in terminal.

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

The root tasks cover the Go backend and web frontend. Companion-app checks live under `//app:`:

```bash
mise run //app:fmt-check
mise run //app:lint
mise run //app:build
```

## 📄 License

Licensed under the [Apache License 2.0](LICENSE).
