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

## 📱 Single App Mode

Tap the bottom-right corner ten times to open **Device Menu**, including before pairing. Use **Enable Single App Mode** or **Disable Single App Mode** to control the session.

[AutonomousSingleAppMode.mobileconfig](Config/AutonomousSingleAppMode.mobileconfig) grants a supervised device permission for this app to enter and exit Single App Mode. MDM delivers this Restrictions payload (`com.apple.applicationaccess`) with the app in `autonomousSingleAppModePermittedAppIDs`. The profile grants permission; **Enable Single App Mode** in Device Menu starts the session. No app entitlement is required. An MDM-forced App Lock must be removed through MDM before the app can own entry and exit. Apple's [`requestGuidedAccessSession` documentation](<https://developer.apple.com/documentation/uikit/uiaccessibility/requestguidedaccesssession(enabled:completionhandler:)>) describes the supervision and MDM requirements.

## 📦 Releases

App releases use `app-<version>` tags and keep `MARKETING_VERSION` in `Config/Version.xcconfig`. We distribute production builds through App Store Connect as a private Custom App.

## 📄 License

Licensed under the [Apache License 2.0](../LICENSE).
