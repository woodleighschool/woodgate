# Companion app

Native iOS client for dedicated check-in Stations. A pairing QR provides the server URL and Station secret; the server owns the Station's location.

## 🧑‍💻 Development

Open `WoodGate.xcodeproj` and use the shared `WoodGate` scheme, or run the repository tasks from the parent directory:

```bash
mise run //app:fmt-check
mise run //app:lint
mise run //app:build
```

For standalone development, run the stateless Station mock from the repository root:

```bash
mise run //mock:dev
```

On an unpaired device, tap the lower-right corner ten times and manually pair with the URL printed by Wrangler and the Station secret `testing123`. The mock uses synthetic people and discards check-ins.

## 📦 Releases

App releases use `app-<version>` tags and keep `MARKETING_VERSION` in `Config/Version.xcconfig`. We distribute production builds through App Store Connect as a private Custom App.

## 📄 License

Licensed under the [Apache License 2.0](../LICENSE).
