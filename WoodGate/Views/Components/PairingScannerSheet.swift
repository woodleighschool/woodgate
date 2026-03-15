//
//  PairingScannerSheet.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import SwiftUI
import Vision
import VisionKit

struct PairingScannerSheet: View {
  // MARK: - Properties

  let onPayload: (String) -> Void

  // MARK: - Body

  var body: some View {
    VStack(spacing: 24) {
      scannerView
      copyView
      Spacer()
    }
    .padding(24)
    .navigationTitle("Pair Device")
    .navigationBarTitleDisplayMode(.inline)
  }

  // MARK: - View Builders

  private var scannerView: some View {
    QRScannerView { payload in
      onPayload(payload)
    }
    .frame(maxWidth: .infinity)
    .frame(height: 420)
    .clipShape(RoundedRectangle(cornerRadius: 28, style: .continuous))
    .shadow(color: .black.opacity(0.1), radius: 10, y: 4)
  }

  private var copyView: some View {
    VStack(alignment: .leading, spacing: 12) {
      Text("Scan Configuration QR")
        .font(.title2.weight(.bold))

      Text(
        "This screen is only used to pair the app with a location. The app will return to the check-in screen automatically once configured."
      )
      .font(.callout)
      .foregroundStyle(.secondary)
    }
    .frame(maxWidth: .infinity, alignment: .leading)
  }
}

// MARK: - Private Components

private struct QRScannerView: UIViewControllerRepresentable {
  // MARK: - Properties

  let onPayload: (String) -> Void

  // MARK: - UIViewControllerRepresentable

  func makeCoordinator() -> Coordinator {
    Coordinator(onPayload: onPayload)
  }

  func makeUIViewController(context: Context) -> DataScannerViewController {
    let controller = DataScannerViewController(
      recognizedDataTypes: [.barcode(symbologies: [.qr])],
      qualityLevel: .balanced,
      recognizesMultipleItems: false,
      isHighFrameRateTrackingEnabled: false,
      isHighlightingEnabled: true
    )
    controller.delegate = context.coordinator
    return controller
  }

  func updateUIViewController(_ controller: DataScannerViewController, context _: Context) {
    guard !controller.isScanning else { return }

    try? controller.startScanning()
  }

  static func dismantleUIViewController(
    _ controller: DataScannerViewController,
    coordinator _: Coordinator
  ) {
    controller.stopScanning()
  }

  // MARK: - Coordinator

  final class Coordinator: NSObject, DataScannerViewControllerDelegate {
    // MARK: - Properties

    private let onPayload: (String) -> Void
    private var hasScanned = false

    // MARK: - Lifecycle

    init(onPayload: @escaping (String) -> Void) {
      self.onPayload = onPayload
    }

    // MARK: - DataScannerViewControllerDelegate

    func dataScanner(
      _ dataScanner: DataScannerViewController,
      didAdd addedItems: [RecognizedItem],
      allItems _: [RecognizedItem]
    ) {
      guard !hasScanned else { return }

      for item in addedItems {
        guard case let .barcode(barcode) = item,
              let payload = barcode.payloadStringValue
        else {
          continue
        }

        hasScanned = true
        dataScanner.stopScanning()
        onPayload(payload)
        return
      }
    }
  }
}
