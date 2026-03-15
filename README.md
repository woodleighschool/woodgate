# WoodGate

WoodGate is a check-in/check-out system. It syncs users and groups from Entra ID, provides an frontend to manage locations and access.

> [!WARNING]
> This project may be unstable or have bugs, use with caution.
> Also expect breaking changes between releases for now.

## ✨ Overview

Three components:

- **Backend** — Go REST API (`/api/v1`), auth endpoints (`/auth`), and static frontend serving.
- **Frontend** — React Admin UI for managing users, groups, locations, check-ins, assets, and API keys.
- **App** — Native Swift app. Configured once via QR code (base URL, location ID, API key) and used as a dedicated check-in terminal.

Users and groups are sync-owned and come from Entra ID. Locations define whether check-ins require notes or photos, and which reusable assets provide background and logo. The app authenticates with an API key, pairs to a location on-device, and submits check-ins against the REST API.

## 🚀 Deploy (Docker)

1. Create a `.env` file with the required values (see Configuration below).
2. Start the stack:

```bash
docker compose up --build
```

The backend listens on `http://localhost:8080`. The Docker image builds the frontend and serves it from the backend.

### Production

- Put it behind HTTPS (Caddy, Nginx, Traefik, etc.).
- Set `WOODGATE_BASE_URL` to the public URL. Used for auth callbacks and cookie security.
- Set a strong `JWT_SECRET`.
- Keep Postgres data on a named volume.
- Keep `WOODGATE_MEDIA_ROOT` on persistent disk if storing asset files locally.

## 🧰 Configuration

| Name                   | What it does                     | Required                  | Notes                                                      |
| ---------------------- | -------------------------------- | ------------------------- | ---------------------------------------------------------- |
| `WOODGATE_PORT`        | HTTP listen port                 | No                        | Defaults to `8080`.                                        |
| `WOODGATE_BASE_URL`    | Public URL for cookies and OAuth | Yes, when auth is enabled | Must be the externally reachable URL.                      |
| `WOODGATE_MEDIA_ROOT`  | Local media storage root         | No                        | Defaults to `media`.                                       |
| `LOG_LEVEL`            | Log verbosity                    | No                        | `debug`, `info`, `warn`, `error`.                          |
| `DATABASE_HOST`        | Postgres host                    | Yes                       |                                                            |
| `DATABASE_PORT`        | Postgres port                    | No                        | Defaults to `5432`.                                        |
| `DATABASE_USER`        | Postgres user                    | Yes                       |                                                            |
| `DATABASE_PASSWORD`    | Postgres password                | Yes                       |                                                            |
| `DATABASE_NAME`        | Postgres database name           | Yes                       |                                                            |
| `DATABASE_SSLMODE`     | Postgres SSL mode                | No                        | Defaults to `disable`.                                     |
| `JWT_SECRET`           | Signing secret for auth          | Yes, when auth is enabled | Keep it dedicated to JWT signing.                          |
| `LOCAL_ADMIN_PASSWORD` | Enable local admin login         | No                        | Username is always `admin`.                                |
| `ENTRA_TENANT_ID`      | Entra tenant ID                  | No                        | Set with the other `ENTRA_*` vars for Entra auth and sync. |
| `ENTRA_CLIENT_ID`      | Entra client ID                  | No                        | Set with the other `ENTRA_*` vars for Entra auth and sync. |
| `ENTRA_CLIENT_SECRET`  | Entra client secret              | No                        | Set with the other `ENTRA_*` vars for Entra auth and sync. |
| `ENTRA_SYNC_ENABLED`   | Enable periodic Entra sync       | No                        | Defaults to `false`.                                       |
| `ENTRA_SYNC_INTERVAL`  | Entra sync interval              | No                        | Defaults to `1h` when enabled.                             |

## 🖥️ App setup

The app is configured once via QR code, which encodes the base URL, and API key. After scanning, the app pairs to the location and operates as a dedicated check-in terminal.

API keys are created in the admin UI. Each key should have check-in permission scoped to the relevant location.

## Permissions

- A permission is a grant for a subject (`user` or `api_key`) with a resource and an action (`read`, `create`, `write`, `delete`).
- Check-in permissions are scoped per location.
- API keys and signed-in users only see or mutate resources their permissions allow.

## 🧪 Local development

**Backend:**

```bash
cd backend
go mod download
go generate ./...
go run ./cmd/woodgate
```

**Frontend:**

```bash
cd frontend
npm install
npm run dev
```

Vite proxies `/api` and `/auth` to `localhost:8080`. The backend serves the built frontend in container deployments.

**App:**

Open `WoodGate.xcodeproj` in Xcode and run on a connected iPad or simulator.

## ⚠️ Limitations

- Entra is the only supported directory sync source.

## 🤝 Contributing / PRs

We are happy to take PRs. Fork this repo, make your changes, and open a PR.
Feel free to open an [issue](https://github.com/woodleighschool/grinch) if you find any bugs or to request a feature.
