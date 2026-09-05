import SwiftUI
import UIKit

struct LocationBackgroundView: View {
    let image: UIImage?

    var body: some View {
        if let image {
            GeometryReader { proxy in
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
                    .frame(width: proxy.size.width, height: proxy.size.height)
                    .clipped()
            }
            .ignoresSafeArea()
        }
    }
}

struct WallpaperCard<Content: View>: View {
    let hasBackground: Bool
    @ViewBuilder let content: Content

    var body: some View {
        if hasBackground {
            content
                .glassEffect(in: .rect(cornerRadius: 28))
        } else {
            content
        }
    }
}
