import SwiftUI

struct WelcomeView: View {
    let isBusy: Bool
    let onScan: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("Pair This Device", systemImage: "qrcode.viewfinder")
        } description: {
            Text("Scan the API key pairing QR code, then choose this device’s location.")
        } actions: {
            Button(action: onScan) {
                Label("Scan QR Code", systemImage: "camera.viewfinder")
            }
            .buttonStyle(.borderedProminent)
            .disabled(isBusy)
        }
    }
}

#Preview("Setup") {
    WelcomeView(isBusy: false, onScan: {})
}

#Preview("Setup — Busy") {
    WelcomeView(isBusy: true, onScan: {})
}
