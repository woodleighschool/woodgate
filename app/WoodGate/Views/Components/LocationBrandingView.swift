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
        } else {
            Color(uiColor: .systemGroupedBackground)
                .ignoresSafeArea()
        }
    }
}

struct WallpaperCard<Content: View>: View {
    @ViewBuilder let content: Content

    var body: some View {
        content
            .background(
                RoundedRectangle(cornerRadius: 28, style: .continuous)
                    .fill(.regularMaterial)
                    .strokeBorder(.primary.opacity(0.12), lineWidth: 1)
            )
    }
}
