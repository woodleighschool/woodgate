# Companion app

Native iOS client for dedicated check-in terminals. A pairing QR provides the server URL and API key, then the operator selects the terminal's location.

The app supports Single App Mode. Permit `au.edu.vic.woodleigh.WoodGateApp` for Autonomous Single App Mode on supervised devices.

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

App releases use `app-v<version>` tags and keep `MARKETING_VERSION` in `Config/Version.xcconfig`. GitHub Actions uploads builds to App Store Connect:

- A change to the app on `main` becomes an internal TestFlight build, one patch above `MARKETING_VERSION` because App Store Connect rejects uploads at or below an approved version.
- A release becomes the build we submit for review, with the tagged version.

The build number is the workflow run number. We distribute production builds through App Store Connect as a private Custom App.

## 📄 License

Licensed under the [Apache License 2.0](../LICENSE).
