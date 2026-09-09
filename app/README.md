# Companion app

Native iOS client for dedicated check-in terminals. A pairing QR provides the server URL and API key, then the operator selects the terminal's location.

## 🧑‍💻 Development

Open `WoodGate.xcodeproj` and use the shared `WoodGate` scheme, or run the repository tasks from the parent directory:

```bash
mise run //app:fmt-check
mise run //app:lint
mise run //app:build
```

For standalone development, run the stateless mock from the repository root:

```bash
mise run //mock:dev
```

The mock's root response includes `review_pairing`, containing its base URL and the public API key `reviewkey`. Encode that object as a QR code, scan it from **Scan QR Code**, and select a location. Review Room accepts optional notes and requires a selfie; Reception exercises check-ins without either. The mock uses synthetic people, supports the app's `/auth/me` and `/api/v1` requests, and discards submissions. Its separate Station endpoints remain available for development.

## 📦 Releases

App releases use `app-<version>` tags and keep `MARKETING_VERSION` in `Config/Version.xcconfig`. We distribute production builds through App Store Connect as a private Custom App.

## 📄 License

Licensed under the [Apache License 2.0](../LICENSE).
