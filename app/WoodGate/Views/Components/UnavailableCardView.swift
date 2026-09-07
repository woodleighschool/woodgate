import SwiftUI

struct UnavailableCardView: View {
    let title: LocalizedStringKey
    let systemImage: String
    let message: LocalizedStringKey
    let hasBackground: Bool

    var body: some View {
        WallpaperCard(hasBackground: hasBackground) {
            VStack(spacing: 16) {
                Image(systemName: systemImage)
                    .font(.system(size: 56))
                    .foregroundStyle(.secondary)

                VStack(spacing: 8) {
                    Text(title)
                        .font(.title2.weight(.bold))

                    Text(message)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                        .multilineTextAlignment(.center)
                }
            }
            .padding(32)
            .frame(maxWidth: 620)
        }
        .padding(.horizontal, 16)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

#Preview("Unavailable — Plain") {
    UnavailableCardView(
        title: "Can't Connect Right Now",
        systemImage: "wifi.exclamationmark",
        message: "The server can't be reached right now.",
        hasBackground: false
    )
}

#Preview("Unavailable — Wallpaper") {
    ZStack {
        LocationBackgroundView(image: PreviewFixtures.brandedSession.backgroundImage)
        UnavailableCardView(
            title: "This Location Is Not Currently Accepting Check-Ins",
            systemImage: "mappin.slash.circle.fill",
            message: "Please see a staff member if you need help.",
            hasBackground: true
        )
    }
}
