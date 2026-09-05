import SwiftUI

struct ContentView: View {
    // MARK: - Properties

    @Environment(ModelData.self) private var modelData
    @Environment(\.scenePhase) private var scenePhase

    @State private var isPairingPresented = false
    @State private var isSecretMenuPresented = false

    // MARK: - Body

    var body: some View {
        NavigationStack {
            ZStack {
                backgroundView
                rootView
            }
            .overlay(alignment: .bottomTrailing) {
                if AppSettings.shared.hasPairing {
                    Color.clear
                        .frame(width: 100, height: 100)
                        .contentShape(Rectangle())
                        .onTapGesture(count: 10) {
                            isSecretMenuPresented = true
                        }
                }
            }
        }
        .onChange(of: scenePhase, initial: true) { _, newValue in
            guard newValue == .active else { return }

            Task {
                await modelData.handleSceneActive()
            }
        }
        .onOpenURL { url in
            Task {
                if await modelData.beginPairing(with: url) {
                    isPairingPresented = false
                }
            }
        }
        .sheet(isPresented: $isPairingPresented) {
            PairingSheet()
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $isSecretMenuPresented) {
            SecretMenuSheet(session: modelData.currentSession)
        }
        .modelAlert(isEnabled: !isPairingPresented && !isSecretMenuPresented)
    }

    // MARK: - View Builders

    private var backgroundView: some View {
        LocationBackgroundView(
            image: modelData.currentSession?.backgroundImage
        )
    }

    @ViewBuilder
    private var rootView: some View {
        if let session = modelData.currentSession {
            if let unavailableState = modelData.unavailableState {
                switch unavailableState {
                case .connectivity:
                    UnavailableCardView(
                        title: "Can't Connect Right Now",
                        systemImage: "wifi.exclamationmark",
                        message:
                        "The server can't be reached right now. You can try refreshing, and this device will keep trying in the background.",
                        hasBackground: session.backgroundImage != nil
                    )
                case .authorization:
                    UnavailableCardView(
                        title: "This Device Is No Longer Authorized",
                        systemImage: "key.slash.fill",
                        message: "This device can no longer accept check-ins with its current pairing.",
                        hasBackground: session.backgroundImage != nil
                    )
                case .locationDisabled:
                    UnavailableCardView(
                        title: "This Location Is Not Currently Accepting Check-Ins",
                        systemImage: "mappin.slash.circle.fill",
                        message: "Please see a staff member if you need help.",
                        hasBackground: session.backgroundImage != nil
                    )
                }
            } else {
                CheckinHomeView(session: session)
            }
        } else if AppSettings.shared.hasPairing {
            UnavailableCardView(
                title: "Can't Connect Right Now",
                systemImage: "wifi.exclamationmark",
                message: "The saved Station configuration is unavailable. This device will keep trying in the background.",
                hasBackground: false
            )
        } else {
            WelcomeView(
                isBusy: modelData.isBusy,
                onPair: {
                    isPairingPresented = true
                }
            )
        }
    }
}
