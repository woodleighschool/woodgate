//
//  WelcomeView.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import SwiftUI

struct WelcomeView: View {
  // MARK: - Properties

  let isBusy: Bool
  let onScan: () -> Void
  let onDemo: () -> Void

  // MARK: - Body

  var body: some View {
    VStack {
      Spacer()

      VStack(spacing: 20) {
        pairingCard
        demoCard
      }
      .padding()

      Spacer()
    }
  }

  // MARK: - View Builders

  private var pairingCard: some View {
    VStack(alignment: .leading, spacing: 16) {
      Label("Pair This Device", systemImage: "qrcode.viewfinder")
        .font(.system(size: 22, weight: .bold, design: .rounded))

      Text(
        "Use the API key pairing QR to connect this device to WoodGate and then choose the location it should stay on."
      )
      .foregroundStyle(.secondary)
      .fixedSize(horizontal: false, vertical: true)

      Button(action: onScan) {
        Label("Scan QR Code", systemImage: "camera.viewfinder")
          .font(.system(size: 17, weight: .semibold, design: .rounded))
          .frame(maxWidth: .infinity)
      }
      .disabled(isBusy)
    }
    .padding(24)
    .glassEffect(in: .rect(cornerRadius: 28))
  }

  private var demoCard: some View {
    VStack(alignment: .leading, spacing: 16) {
      Label("Demo Mode", systemImage: "sparkles.rectangle.stack")
        .font(.system(size: 22, weight: .bold, design: .rounded))

      Text("Run the whole check-in flow on device without a live server.")
        .foregroundStyle(.secondary)
        .fixedSize(horizontal: false, vertical: true)

      Button(action: onDemo) {
        Label("Try Demo", systemImage: "play.fill")
          .font(.system(size: 17, weight: .semibold, design: .rounded))
          .frame(maxWidth: .infinity)
      }
      .disabled(isBusy)
    }
    .padding(24)
    .glassEffect(in: .rect(cornerRadius: 28))
  }
}
