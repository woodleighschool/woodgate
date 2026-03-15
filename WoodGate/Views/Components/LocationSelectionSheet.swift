//
//  LocationSelectionSheet.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import SwiftUI

struct LocationSelectionSheet: View {
  // MARK: - Properties

  let selection: LocationSelectionState
  let isBusy: Bool
  let onSelect: (SessionLocation) -> Void

  // MARK: - Body

  var body: some View {
    Form {
      Section("Available Locations") {
        ForEach(selection.options) { option in
          Button {
            onSelect(option)
          } label: {
            HStack {
              Label(option.name, systemImage: "building.2")
                .foregroundStyle(.primary)

              Spacer()

              if isBusy {
                ProgressView()
              } else {
                Image(systemName: "chevron.right")
                  .font(.caption.weight(.bold))
                  .foregroundStyle(.tertiary)
              }
            }
            .contentShape(Rectangle())
          }
          .disabled(isBusy)
        }
      }
    }
    .navigationTitle("Select Location")
    .navigationBarTitleDisplayMode(.inline)
    .interactiveDismissDisabled(isBusy)
  }
}
