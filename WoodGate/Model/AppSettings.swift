//
//  AppSettings.swift
//  WoodGate
//
//  Created by Alexander Hyde on 16/2/2026.
//

import Foundation
import SwiftData
import SwiftUI

@MainActor
@Observable
final class AppSettings {
  // MARK: - Properties

  static let shared = AppSettings()

  var baseURLString: String {
    didSet {
      defaults.set(baseURLString, forKey: Key.baseURLString)
    }
  }

  var apiKey: String {
    didSet {
      KeychainHelper.shared.save(apiKey, key: Key.apiKey)
    }
  }

  var locationID: UUID? {
    didSet {
      defaults.set(locationID?.uuidString.lowercased(), forKey: Key.locationID)
    }
  }

  var locationName: String {
    didSet {
      defaults.set(locationName, forKey: Key.locationName)
    }
  }

  var notes: Bool {
    didSet {
      defaults.set(notes, forKey: Key.notes)
    }
  }

  var photo: Bool {
    didSet {
      defaults.set(photo, forKey: Key.photo)
    }
  }

  var backgroundAssetID: UUID? {
    didSet {
      defaults.set(backgroundAssetID?.uuidString.lowercased(), forKey: Key.backgroundAssetID)
    }
  }

  var logoAssetID: UUID? {
    didSet {
      defaults.set(logoAssetID?.uuidString.lowercased(), forKey: Key.logoAssetID)
    }
  }

  var lastSyncedAt: Date? {
    didSet {
      defaults.set(lastSyncedAt, forKey: Key.lastSyncedAt)
    }
  }

  var hasPairing: Bool {
    locationID != nil && baseURLString.isEmpty == false && apiKey.isEmpty == false
  }

  // MARK: - Private

  private let defaults = UserDefaults.standard

  private enum Key {
    static let baseURLString = "baseURLString"
    static let apiKey = "apiKey"
    static let locationID = "locationID"
    static let locationName = "locationName"
    static let notes = "notes"
    static let photo = "photo"
    static let backgroundAssetID = "backgroundAssetID"
    static let logoAssetID = "logoAssetID"
    static let lastSyncedAt = "lastSyncedAt"
  }

  // MARK: - Initialization

  private init() {
    baseURLString = defaults.string(forKey: Key.baseURLString) ?? ""
    apiKey = KeychainHelper.shared.read(key: Key.apiKey) ?? ""
    locationID = Self.uuid(forKey: Key.locationID, defaults: defaults)
    locationName = defaults.string(forKey: Key.locationName) ?? ""
    notes = defaults.bool(forKey: Key.notes)
    photo = defaults.bool(forKey: Key.photo)
    backgroundAssetID = Self.uuid(forKey: Key.backgroundAssetID, defaults: defaults)
    logoAssetID = Self.uuid(forKey: Key.logoAssetID, defaults: defaults)
    lastSyncedAt = defaults.object(forKey: Key.lastSyncedAt) as? Date
  }

  // MARK: - Public Methods

  func clear(removeAPIKey: Bool = true) {
    baseURLString = ""
    locationID = nil
    locationName = ""
    notes = false
    photo = false
    backgroundAssetID = nil
    logoAssetID = nil
    lastSyncedAt = nil
    if removeAPIKey {
      apiKey = ""
    }
  }

  // MARK: - Helpers

  private static func uuid(forKey key: String, defaults: UserDefaults) -> UUID? {
    guard let rawValue = defaults.string(forKey: key) else {
      return nil
    }

    return UUID(uuidString: rawValue)
  }
}

// MARK: - Client Helpers

extension AppSettings {
  func woodGateClient(session: URLSession = .shared) -> WoodGateAPIClient? {
    guard let baseURL = URL(string: baseURLString), apiKey.isEmpty == false else {
      return nil
    }

    return WoodGateAPIClient(baseURL: baseURL, apiKey: apiKey, session: session)
  }

  func woodGateClient(
    baseURLString: String,
    apiKey: String,
    session: URLSession = .shared
  ) -> WoodGateAPIClient? {
    guard let baseURL = URL(string: baseURLString), apiKey.isEmpty == false else {
      return nil
    }

    return WoodGateAPIClient(baseURL: baseURL, apiKey: apiKey, session: session)
  }
}
