import SwiftUI

struct WelcomeView: View {
    let isBusy: Bool
    let onScan: () -> Void

    var body: some View {
        ContentUnavailableView {
            Label("Pair This Device", systemImage: "qrcode.viewfinder")
        } description: {
            Text("Scan a Station pairing code to continue.")
        } actions: {
            Button(action: onScan) {
                Label("Scan QR Code", systemImage: "camera.viewfinder")
            }
            .buttonStyle(.borderedProminent)
            .disabled(isBusy)
        }
    }
}
