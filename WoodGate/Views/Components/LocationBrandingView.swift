//
//  LocationBrandingView.swift
//  WoodGate
//
//  Created by Alexander Hyde on 15/3/2026.
//

import SwiftUI
import UIKit

struct LocationBackgroundView: View {
  let image: UIImage?

  var body: some View {
    GeometryReader { proxy in
      Group {
        if let image {
          Image(uiImage: image)
            .resizable()
            .scaledToFill()
        } else {
          Image("background")
            .resizable()
            .scaledToFill()
        }
      }
      .frame(width: proxy.size.width, height: proxy.size.height)
      .clipped()
    }
    .ignoresSafeArea()
  }
}

struct LocationLogoView: View {
  let image: UIImage?

  var body: some View {
    Group {
      if let image {
        Image(uiImage: image)
          .resizable()
          .scaledToFit()
      }
    }
  }
}
