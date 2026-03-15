//
//  SessionModels.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import Foundation
import UIKit

enum SessionMode: String {
  case paired
  case demo
}

enum CheckinDirectionChoice: String, CaseIterable, Identifiable, Codable {
  case checkIn = "check_in"
  case checkOut = "check_out"

  var id: String {
    rawValue
  }

  var title: String {
    switch self {
    case .checkIn:
      "Check In"
    case .checkOut:
      "Check Out"
    }
  }

  var symbolName: String {
    switch self {
    case .checkIn:
      "arrow.right.circle.fill"
    case .checkOut:
      "arrow.left.circle.fill"
    }
  }
}

struct PairingPayload: Codable, Hashable {
  let baseURL: String
  let apiKey: String

  enum CodingKeys: String, CodingKey {
    case baseURL = "base_url"
    case apiKey = "api_key"
  }

  static func parse(json: String) throws -> PairingPayload {
    try JSONDecoder().decode(PairingPayload.self, from: Data(json.utf8))
  }
}

struct SessionLocation: Identifiable, Hashable {
  let id: UUID
  let name: String
}

struct ActiveLocation: Identifiable, Hashable {
  let id: UUID
  let name: String
  let notes: Bool
  let photo: Bool
  let backgroundAssetID: UUID?
  let logoAssetID: UUID?
}

struct PersonSummary: Identifiable, Hashable {
  let id: UUID
  let displayName: String
  let email: String
}

struct ActiveSession {
  let mode: SessionMode
  let baseURLString: String
  var location: ActiveLocation
  var people: [PersonSummary]
  var backgroundImage: UIImage?
  var logoImage: UIImage?
  var lastSyncedAt: Date

  var isDemo: Bool {
    mode == .demo
  }
}

struct LocationSelectionState: Identifiable {
  let id = UUID()
  let options: [SessionLocation]
  let payload: PairingPayload
}

struct WoodGateError: LocalizedError {
  let message: String

  var errorDescription: String? {
    message
  }
}
