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

    var stationSecret: String {
        didSet {
            KeychainHelper.shared.save(stationSecret, key: Key.stationSecret)
        }
    }

    var stationID: Int64? {
        didSet {
            defaults.set(stationID, forKey: Key.stationID)
        }
    }

    var stationName: String {
        didSet {
            defaults.set(stationName, forKey: Key.stationName)
        }
    }

    var locationID: Int64? {
        didSet {
            defaults.set(locationID, forKey: Key.locationID)
        }
    }

    var locationName: String {
        didSet {
            defaults.set(locationName, forKey: Key.locationName)
        }
    }

    var locationEnabled: Bool {
        didSet {
            defaults.set(locationEnabled, forKey: Key.locationEnabled)
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

    var backgroundObjectID: Int64? {
        didSet {
            defaults.set(backgroundObjectID, forKey: Key.backgroundObjectID)
        }
    }

    var logoObjectID: Int64? {
        didSet {
            defaults.set(logoObjectID, forKey: Key.logoObjectID)
        }
    }

    var lastSyncedAt: Date? {
        didSet {
            defaults.set(lastSyncedAt, forKey: Key.lastSyncedAt)
        }
    }

    var hasPairing: Bool {
        baseURLString.isEmpty == false && stationSecret.isEmpty == false
    }

    // MARK: - Private

    private let defaults = UserDefaults.standard

    private enum Key {
        static let baseURLString = "baseURLString"
        static let stationSecret = "stationSecret"
        static let stationID = "stationID"
        static let stationName = "stationName"
        static let locationID = "locationID"
        static let locationName = "locationName"
        static let locationEnabled = "locationEnabled"
        static let notes = "notes"
        static let photo = "photo"
        static let backgroundObjectID = "backgroundObjectID"
        static let logoObjectID = "logoObjectID"
        static let lastSyncedAt = "lastSyncedAt"
    }

    // MARK: - Initialization

    private init() {
        baseURLString = defaults.string(forKey: Key.baseURLString) ?? ""
        stationSecret = KeychainHelper.shared.read(key: Key.stationSecret) ?? ""
        stationID = Self.int64(forKey: Key.stationID, defaults: defaults)
        stationName = defaults.string(forKey: Key.stationName) ?? ""
        locationID = Self.int64(forKey: Key.locationID, defaults: defaults)
        locationName = defaults.string(forKey: Key.locationName) ?? ""
        locationEnabled = defaults.bool(forKey: Key.locationEnabled)
        notes = defaults.bool(forKey: Key.notes)
        photo = defaults.bool(forKey: Key.photo)
        backgroundObjectID = Self.int64(forKey: Key.backgroundObjectID, defaults: defaults)
        logoObjectID = Self.int64(forKey: Key.logoObjectID, defaults: defaults)
        lastSyncedAt = defaults.object(forKey: Key.lastSyncedAt) as? Date
    }

    // MARK: - Public Methods

    func clear(removeStationSecret: Bool = true) {
        baseURLString = ""
        stationID = nil
        stationName = ""
        locationID = nil
        locationName = ""
        locationEnabled = false
        notes = false
        photo = false
        backgroundObjectID = nil
        logoObjectID = nil
        lastSyncedAt = nil
        if removeStationSecret {
            stationSecret = ""
        }
    }

    // MARK: - Helpers

    private static func int64(forKey key: String, defaults: UserDefaults) -> Int64? {
        guard let number = defaults.object(forKey: key) as? NSNumber else {
            return nil
        }
        return number.int64Value
    }
}

// MARK: - Client Helpers

extension AppSettings {
    func woodGateClient(session: URLSession = .shared) -> WoodGateAPIClient? {
        guard let baseURL = validServerURL(baseURLString), stationSecret.isEmpty == false else {
            return nil
        }

        return WoodGateAPIClient(baseURL: baseURL, stationSecret: stationSecret, session: session)
    }

    func woodGateClient(
        baseURLString: String,
        stationSecret: String,
        session: URLSession = .shared
    ) -> WoodGateAPIClient? {
        guard let baseURL = validServerURL(baseURLString), stationSecret.isEmpty == false else {
            return nil
        }

        return WoodGateAPIClient(baseURL: baseURL, stationSecret: stationSecret, session: session)
    }
}

private func validServerURL(_ value: String) -> URL? {
    guard
        let url = URL(string: value.trimmingCharacters(in: .whitespacesAndNewlines)),
        let scheme = url.scheme?.lowercased(),
        scheme == "http" || scheme == "https",
        url.host != nil,
        url.user == nil,
        url.password == nil,
        url.query == nil,
        url.fragment == nil
    else {
        return nil
    }
    return url
}
