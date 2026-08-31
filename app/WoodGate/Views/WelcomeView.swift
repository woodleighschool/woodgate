import SwiftUI

struct WelcomeView: View {
    let isBusy: Bool
    let onPair: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("Pair This Device", systemImage: "qrcode.viewfinder")
        } description: {
            Text("Scan a Station pairing code or enter its details manually.")
        } actions: {
            Button(action: onPair) {
                Label("Pair Device", systemImage: "link")
            }
            .buttonStyle(.borderedProminent)
            .disabled(isBusy)
        }
    }
}

#Preview("Setup") {
    WelcomeView(isBusy: false, onPair: {})
}

#Preview("Setup — Busy") {
    WelcomeView(isBusy: true, onPair: {})
}
