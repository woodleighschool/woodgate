//
//  ContentView.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import SwiftUI

struct ContentView: View {
  // MARK: - Properties

  @Environment(ModelData.self) private var modelData
  @Environment(\.scenePhase) private var scenePhase

  @State private var isScannerPresented = false
  @State private var isSecretMenuPresented = false

  // MARK: - Computed Properties

  private var locationSelectionBinding: Binding<LocationSelectionState?> {
    Binding(
      get: { modelData.locationSelection },
      set: { newValue in
        if newValue == nil {
          modelData.cancelLocationSelection()
        }
      }
    )
  }

  private var alertBinding: Binding<AlertItem?> {
    Binding(
      get: { modelData.alert },
      set: { newValue in
        modelData.alert = newValue
      }
    )
  }

  // MARK: - Body

  var body: some View {
    NavigationStack {
      ZStack {
        backgroundView
        rootView
      }
      .overlay(alignment: .bottomTrailing) {
        Color.clear
          .frame(width: 100, height: 100)
          .contentShape(Rectangle())
          .onTapGesture(count: 10) {
            isSecretMenuPresented = true
          }
      }
    }
    .onChange(of: scenePhase, initial: true) { _, newValue in
      guard newValue == .active else { return }

      Task {
        await modelData.handleSceneActive()
      }
    }
    .sheet(isPresented: $isScannerPresented) {
      scannerSheet
    }
    .sheet(isPresented: $isSecretMenuPresented) {
      SecretMenuSheet(session: modelData.currentSession)
    }
    .sheet(item: locationSelectionBinding) { selection in
      locationSelectionSheet(selection: selection)
    }
    .alert(item: alertBinding) { alert in
      Alert(
        title: Text(alert.title),
        message: Text(alert.message),
        dismissButton: .default(Text("OK"))
      )
    }
  }

  // MARK: - View Builders

  private var backgroundView: some View {
    LocationBackgroundView(
      image: modelData.currentSession?.backgroundImage
    )
  }

  @ViewBuilder
  private var rootView: some View {
    if let session = modelData.currentSession {
      if let unavailableState = modelData.unavailableState, !session.isDemo {
        switch unavailableState {
        case .connectivity:
          unavailableView
        case .authorization:
          UnavailableCardView(
            title: "This Device Is No Longer Authorized",
            systemImage: "key.slash.fill",
            message: "This device can no longer accept check-ins with its current pairing."
          )
        case .locationDisabled:
          UnavailableCardView(
            title: "This Location Is Not Currently Accepting Check-Ins",
            systemImage: "mappin.slash.circle.fill",
            message: "Please see a staff member if you need help."
          )
        }
      } else {
        CheckinHomeView(session: session)
      }
    } else if AppSettings.shared.hasPairing {
      Color.clear
    } else {
      WelcomeView(
        isBusy: modelData.isBusy,
        onScan: {
          isScannerPresented = true
        },
        onDemo: {
          modelData.beginDemoMode()
        }
      )
    }
  }

  private var scannerSheet: some View {
    NavigationStack {
      PairingScannerSheet(
        onPayload: { payload in
          isScannerPresented = false

          Task {
            await modelData.beginPairing(with: payload)
          }
        }
      )
    }
    .presentationDetents([.large])
    .presentationDragIndicator(.visible)
  }

  private var refreshButton: some View {
    Button("Refresh") {
      Task {
        await modelData.refreshSession()
      }
    }
    .buttonStyle(.borderedProminent)
  }

  // MARK: - Private Helpers

  private func locationSelectionSheet(selection: LocationSelectionState) -> some View {
    NavigationStack {
      LocationSelectionSheet(
        selection: selection,
        isBusy: modelData.isBusy,
        onSelect: { option in
          Task {
            await modelData.selectLocation(option)
          }
        }
      )
      .alert(item: alertBinding) { alert in
        Alert(
          title: Text(alert.title),
          message: Text(alert.message),
          dismissButton: .default(Text("OK"))
        )
      }
    }
    .presentationDetents([.large])
    .presentationDragIndicator(.visible)
  }

  private var unavailableView: some View {
    VStack(alignment: .leading, spacing: 22) {
      VStack(alignment: .leading, spacing: 10) {
        Label("Can't Connect Right Now", systemImage: "wifi.exclamationmark")
          .font(.system(size: 28, weight: .bold, design: .rounded))

        Text(
          "The server can't be reached right now. You can try refreshing, and this device will keep trying in the background."
        )
        .font(.system(size: 17, weight: .medium, design: .rounded))
        .foregroundStyle(.secondary)
        .fixedSize(horizontal: false, vertical: true)
      }

      ViewThatFits(in: .horizontal) {
        HStack(spacing: 12) {
          refreshButton
        }

        VStack(spacing: 12) {
          refreshButton
        }
      }
    }
    .padding(24)
    .frame(maxWidth: 620)
    .glassEffect(in: .rect(cornerRadius: 28))
    .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
    .safeAreaPadding(.horizontal, 16)
    .safeAreaPadding(.vertical, 16)
  }
}
