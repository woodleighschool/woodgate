# woodgate

Internal check-in system with a Go API, a React administration interface, and a native companion app for dedicated Stations.

Users and groups can sync from Microsoft Entra. Roles grant `none`, `view`, or `edit` access to each resource, and can be assigned directly to users or through groups.

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

The companion app is configured by opening a Station configuration QR code or entering its server URL and key manually. The server binds each Station to a location. We distribute it as a private Custom App through Apple School Manager and our MDM.

## ⚙️ Configuration

Runtime configuration uses `WOODGATE_` environment variables. Start with [`.env.example`](.env.example) and persist PostgreSQL and the selected storage backend.

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
