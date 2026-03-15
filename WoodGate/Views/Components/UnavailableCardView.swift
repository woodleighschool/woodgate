//
//  UnavailableCardView.swift
//  WoodGate
//
//  Created by Alexander Hyde on 15/3/2026.
//

import SwiftUI

struct UnavailableCardView: View {
  let title: LocalizedStringKey
  let systemImage: String
  let message: LocalizedStringKey

  var body: some View {
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
    .glassEffect(in: .rect(cornerRadius: 28))
    .padding(16)
  }
}
