//
//  WoodGateModels.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

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
  let rows: [Row]
  let total: Int

  enum CodingKeys: String, CodingKey {
    case rows
    case total
  }
}

// MARK: Authentication

/// Response
nonisolated struct WoodGateAuthMeResponse: Decodable {
  nonisolated struct Principal: Decodable {
    let type: String
    let id: String
    let displayName: String?
    let email: String?
    let name: String?

    enum CodingKeys: String, CodingKey {
      case type
      case id
      case displayName = "display_name"
      case email
      case name
    }
  }

  let principal: Principal
  let admin: Bool
  let access: [WoodGatePermissionGrant]

  enum CodingKeys: String, CodingKey {
    case principal
    case admin
    case access
  }
}

nonisolated struct WoodGatePermissionGrant: Decodable {
  let resource: String
  let action: String
  let locationId: UUID?

  enum CodingKeys: String, CodingKey {
    case resource
    case action
    case locationId = "location_id"
  }
}

// MARK: Locations

/// Response
nonisolated struct WoodGateLocationResponse: Decodable {
  let id: UUID
  let name: String
  let enabled: Bool
  let notes: Bool
  let photo: Bool
  let backgroundAssetId: UUID?
  let logoAssetId: UUID?
  let groupIds: [UUID]

  enum CodingKeys: String, CodingKey {
    case id
    case name
    case enabled
    case notes
    case photo
    case backgroundAssetId = "background_asset_id"
    case logoAssetId = "logo_asset_id"
    case groupIds = "group_ids"
  }
}

// MARK: Users

/// Response
nonisolated struct WoodGateUserResponse: Decodable {
  let id: UUID
  let upn: String
  let displayName: String

  enum CodingKeys: String, CodingKey {
    case id
    case upn
    case displayName = "display_name"
  }
}

// MARK: Checkins

/// Response
nonisolated struct WoodGateCheckinResponse: Decodable {
  let id: UUID
  let userId: UUID
  let locationId: UUID
  let direction: CheckinDirectionChoice

  enum CodingKeys: String, CodingKey {
    case id
    case userId = "user_id"
    case locationId = "location_id"
    case direction
  }
}
