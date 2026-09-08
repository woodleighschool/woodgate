import SwiftUI

struct ContentView: View {
    // MARK: - Properties

    @Environment(ModelData.self) private var modelData
    @Environment(\.scenePhase) private var scenePhase

    @State private var isPairingPresented = false
    @State private var isSecretMenuPresented = false

    // MARK: - Computed Properties

    private var pairingPresentationBinding: Binding<Bool> {
        Binding(
            get: { isPairingPresented || modelData.locationSelection != nil },
            set: { presented in
                isPairingPresented = presented
                if !presented {
                    modelData.cancelLocationSelection()
                }
            }
        )
    }

    private var alertBinding: Binding<AlertItem?> {
        Binding(
            get: { modelData.alert },
            set: { newValue in
                modelData.alert = newValue
            }
        )
    }

    // MARK: - Body

    var body: some View {
        NavigationStack {
            ZStack {
                backgroundView
                rootView
            }
            .overlay(alignment: .bottomTrailing) {
                Color.clear
                    .frame(width: 100, height: 100)
                    .contentShape(Rectangle())
                    .onTapGesture(count: 10) {
                        isSecretMenuPresented = true
                    }
            }
        }
        .onChange(of: scenePhase, initial: true) { _, newValue in
            guard newValue == .active else { return }

            Task {
                await modelData.handleSceneActive()
            }
        }
        .sheet(isPresented: pairingPresentationBinding) {
            pairingSheet
        }
        .sheet(isPresented: $isSecretMenuPresented) {
            SecretMenuSheet(session: modelData.currentSession)
        }
        .alert(item: alertBinding) { alert in
            Alert(
                title: Text(alert.title),
                message: Text(alert.message),
                dismissButton: .default(Text("OK"))
            )
        }
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
                        "The server can't be reached right now. You can try refreshing, and this device will keep trying in the background."
                    )
                case .authorization:
                    UnavailableCardView(
                        title: "This Device Is No Longer Authorized",
                        systemImage: "key.slash.fill",
                        message: "This device can no longer accept check-ins with its current pairing."
                    )
                case .locationDisabled:
                    UnavailableCardView(
                        title: "This Location Is Not Currently Accepting Check-Ins",
                        systemImage: "mappin.slash.circle.fill",
                        message: "Please see a staff member if you need help."
                    )
                }
            } else {
                CheckinHomeView(session: session)
                    .id(session.location.id)
            }
        } else if AppSettings.shared.hasPairing {
            UnavailableCardView(
                title: "Can’t Connect Right Now", systemImage: "wifi.exclamationmark",
                message: "The saved configuration is unavailable. This device will keep trying in the background."
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

    private var pairingSheet: some View {
        NavigationStack {
            Group {
                if let selection = modelData.locationSelection {
                    LocationSelectionSheet(selection: selection, isBusy: modelData.isBusy) { option in
                        Task {
                            await modelData.selectLocation(option)
                            if modelData.locationSelection == nil {
                                isPairingPresented = false
                            }
                        }
                    }
                } else {
                    PairingScannerSheet(onPayload: modelData.beginPairing)
                }
            }
            .alert(item: alertBinding) { alert in
                Alert(title: Text(alert.title), message: Text(alert.message))
            }
        }
        .interactiveDismissDisabled(modelData.isBusy)
        .presentationDetents([.large])
        .presentationDragIndicator(.visible)
    }
}
