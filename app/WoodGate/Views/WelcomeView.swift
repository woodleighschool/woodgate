import SwiftUI

struct WelcomeView: View {
    let isBusy: Bool
    let onPair: () -> Void

    var body: some View {
        ScrollView {
            VStack(spacing: 24) {
                Image("Logo")
                    .resizable()
                    .scaledToFit()
                    .frame(width: 96, height: 96)
                    .foregroundStyle(.tint)
                    .accessibilityHidden(true)

                VStack(spacing: 12) {
                    Text("Welcome to WoodGate")
                        .font(.title.bold())
                        .accessibilityAddTraits(.isHeader)

                    Text("To get started, create a new app key on your server.")
                        .foregroundStyle(.secondary)
                }
                .multilineTextAlignment(.center)

                Button("Set Up", action: onPair)
                    .buttonStyle(.borderedProminent)
                    .controlSize(.large)
                    .disabled(isBusy)
            }
            .frame(maxWidth: 360)
            .padding(24)
            .frame(maxWidth: .infinity)
        }
        .defaultScrollAnchor(.center)
    }
}

#Preview("Setup") {
    WelcomeView(isBusy: false, onPair: {})
}

#Preview("Setup — Busy") {
    WelcomeView(isBusy: true, onPair: {})
}
