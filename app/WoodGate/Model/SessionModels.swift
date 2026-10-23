import Foundation
import UIKit

enum CheckinDirectionChoice: String, CaseIterable, Identifiable, Codable {
    case checkIn = "check_in"
    case checkOut = "check_out"

    var id: String {
        rawValue
    }
}

struct PairingPayload: Hashable {
    let baseURL: String
    let stationKey: String

    static func parse(urlString: String) throws -> PairingPayload {
        guard let url = URL(string: urlString) else {
            throw WoodGateError(message: "Scan a valid WoodGate configuration QR code.")
        }
        return try parse(url: url)
    }

    static func parse(url: URL) throws -> PairingPayload {
        guard
            url.scheme?.lowercased() == "woodgate",
            url.host?.lowercased() == "pair",
            let components = URLComponents(url: url, resolvingAgainstBaseURL: false),
            let server = queryValue(named: "server", in: components),
            let key = queryValue(named: "key", in: components)
        else {
            throw WoodGateError(message: "Scan a valid WoodGate configuration QR code.")
        }
        return PairingPayload(baseURL: server, stationKey: key)
    }

    private static func queryValue(named name: String, in components: URLComponents) -> String? {
        let values = components.queryItems?
            .filter { $0.name == name }
            .compactMap(\.value) ?? []
        guard values.count == 1 else { return nil }

        let value = values[0].trimmingCharacters(in: .whitespacesAndNewlines)
        return value.isEmpty ? nil : value
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
