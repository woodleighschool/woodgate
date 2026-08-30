import Foundation
import UIKit

enum CheckinDirectionChoice: String, CaseIterable, Identifiable, Codable {
    case checkIn = "check_in"
    case checkOut = "check_out"

    var id: String {
        rawValue
    }
}

struct PairingPayload: Codable, Hashable {
    let baseURL: String
    let stationSecret: String

    enum CodingKeys: String, CodingKey {
        case baseURL = "base_url"
        case stationSecret = "station_secret"
    }

    static func parse(json: String) throws -> PairingPayload {
        try JSONDecoder().decode(PairingPayload.self, from: Data(json.utf8))
    }
}

struct ActiveLocation: Identifiable, Hashable {
    let id: Int64
    let name: String
    let enabled: Bool
    let notes: Bool
    let photo: Bool
    let backgroundObjectID: Int64?
    let logoObjectID: Int64?
}

struct PersonSummary: Identifiable, Hashable {
    let id: Int64
    let displayName: String
    let email: String
}

struct ActiveSession {
    let baseURLString: String
    let stationID: Int64
    let stationName: String
    let location: ActiveLocation
    let people: [PersonSummary]
    let backgroundImage: UIImage?
    let logoImage: UIImage?
    var lastSyncedAt: Date
}

struct WoodGateError: LocalizedError {
    let message: String

    var errorDescription: String? {
        message
    }
}
