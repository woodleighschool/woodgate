import Foundation

// MARK: Shared API

/// Response
nonisolated struct WoodGateProblemResponse: Decodable {
    nonisolated struct FieldProblem: Decodable {
        let field: String
        let message: String
        let code: String?

        enum CodingKeys: String, CodingKey {
            case field
            case message
            case code
        }
    }

    let status: Int
    let detail: String
    let code: String
    let fieldErrors: [FieldProblem]?

    enum CodingKeys: String, CodingKey {
        case status
        case detail
        case code
        case fieldErrors = "field_errors"
    }
}

/// Response
nonisolated struct WoodGateListResponse<Row: Decodable & Sendable>: Decodable {
    let items: [Row]
    let count: Int

    enum CodingKeys: String, CodingKey {
        case items
        case count
    }
}

nonisolated struct WoodGateErrorResponse: Decodable {
    let error: String
}

// MARK: Station Configuration

/// Response
nonisolated struct WoodGateStationConfigurationResponse: Decodable {
    let stationId: Int64
    let stationName: String
    let location: WoodGateStationLocation

    enum CodingKeys: String, CodingKey {
        case stationId = "station_id"
        case stationName = "station_name"
        case location
    }
}

nonisolated struct WoodGateStationLocation: Decodable {
    let id: Int64
    let name: String
    let enabled: Bool
    let notes: Bool
    let photo: Bool
    let backgroundObjectId: Int64?
    let logoObjectId: Int64?

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case enabled
        case notes
        case photo
        case backgroundObjectId = "background_object_id"
        case logoObjectId = "logo_object_id"
    }
}

// MARK: Users

/// Response
nonisolated struct WoodGateUserResponse: Decodable {
    let id: Int64
    let name: String
    let email: String

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case email
    }
}

// MARK: Checkins

/// Response
nonisolated struct WoodGateCheckinResponse: Decodable {
    let id: Int64
    let personId: Int64
    let locationId: Int64
    let direction: CheckinDirectionChoice

    enum CodingKeys: String, CodingKey {
        case id
        case personId = "person_id"
        case locationId = "location_id"
        case direction
    }
}

// MARK: Control Plane

nonisolated struct WoodGateStationControlMessage: Decodable {
    let type: String
}
