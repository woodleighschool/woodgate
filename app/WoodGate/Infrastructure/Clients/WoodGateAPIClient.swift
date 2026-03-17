//
//  WoodGateAPIClient.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import Foundation

struct WoodGateAPIError: LocalizedError {
  // MARK: - Properties

  let statusCode: Int
  let detail: String

  var errorDescription: String? {
    detail
  }
}

struct WoodGateAPIClient {
  // MARK: - Properties

  private let baseURL: URL
  private let apiKey: String
  private let session: URLSession

  private static let decoder: JSONDecoder = .init()

  // MARK: - Init

  init(baseURL: URL, apiKey: String, session: URLSession = .shared) {
    self.baseURL = baseURL
    self.apiKey = apiKey
    self.session = session
  }

  // MARK: - Public Methods

  func authenticate() async throws -> WoodGateAuthMeResponse {
    try await get(path: "/auth/me")
  }

  func listLocations() async throws -> [WoodGateLocationResponse] {
    let response: WoodGateListResponse<WoodGateLocationResponse> = try await get(
      path: "/api/v1/locations"
    )
    return response.rows
  }

  func getLocation(id: UUID) async throws -> WoodGateLocationResponse {
    try await get(path: "/api/v1/locations/\(id.uuidString.lowercased())")
  }

  func listPeople(locationID: UUID) async throws -> [PersonSummary] {
    var people: [PersonSummary] = []
    var offset = 0
    let limit = 250

    while true {
      let response: WoodGateListResponse<WoodGateUserResponse> = try await get(
        path: "/api/v1/users",
        queryItems: [
          URLQueryItem(name: "location_id", value: locationID.uuidString.lowercased()),
          URLQueryItem(name: "limit", value: "\(limit)"),
          URLQueryItem(name: "offset", value: "\(offset)"),
        ]
      )

      people.append(
        contentsOf: response.rows.map { row in
          PersonSummary(
            id: row.id,
            displayName: row.displayName,
            email: row.upn
          )
        }
      )

      offset += response.rows.count

      if offset >= response.total || response.rows.isEmpty { break }
    }

    return people
  }

  func createCheckin(
    locationID: UUID,
    userID: UUID,
    direction: CheckinDirectionChoice,
    notes: String?,
    photoJPEGData: Data?
  ) async throws -> WoodGateCheckinResponse {
    let boundary = "WoodGateBoundary-\(UUID().uuidString)"
    var body = Data()

    appendField("user_id", value: userID.uuidString.lowercased(), to: &body, boundary: boundary)
    appendField(
      "location_id",
      value: locationID.uuidString.lowercased(),
      to: &body,
      boundary: boundary
    )
    appendField("direction", value: direction.rawValue, to: &body, boundary: boundary)

    if let notes, notes.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false {
      appendField("notes", value: notes, to: &body, boundary: boundary)
    }

    if let photoJPEGData {
      appendFile(
        field: "photo",
        filename: "selfie.jpg",
        mimeType: "image/jpeg",
        data: photoJPEGData,
        to: &body,
        boundary: boundary
      )
    }

    body.append(Data("--\(boundary)--\r\n".utf8))

    var request = try makeRequest(path: "/api/v1/checkins")
    request.httpMethod = "POST"
    request.setValue(
      "multipart/form-data; boundary=\(boundary)",
      forHTTPHeaderField: "Content-Type"
    )

    let (data, response) = try await session.upload(for: request, from: body)
    return try decodeResponse(WoodGateCheckinResponse.self, data: data, response: response)
  }

  func getAssetContent(id: UUID) async throws -> Data {
    let request = try makeRequest(path: "/api/v1/assets/\(id.uuidString.lowercased())/content")
    let (data, response) = try await session.data(for: request)
    try validateResponse(data: data, response: response)
    return data
  }

  // MARK: - Private Helpers

  private func get<Response: Decodable>(
    path: String,
    queryItems: [URLQueryItem] = []
  ) async throws -> Response {
    let request = try makeRequest(path: path, queryItems: queryItems)
    let (data, response) = try await session.data(for: request)
    return try decodeResponse(Response.self, data: data, response: response)
  }

  private func makeRequest(path: String, queryItems: [URLQueryItem] = []) throws -> URLRequest {
    guard
      var components = URLComponents(
        url: baseURL.appending(path: path),
        resolvingAgainstBaseURL: false
      )
    else {
      throw WoodGateError(message: "The server URL is invalid.")
    }

    if queryItems.isEmpty == false {
      components.queryItems = queryItems
    }

    guard let url = components.url else {
      throw WoodGateError(message: "The server URL is invalid.")
    }

    var request = URLRequest(url: url)
    request.setValue(apiKey, forHTTPHeaderField: "X-API-Key")
    request.setValue("application/json", forHTTPHeaderField: "Accept")
    request.timeoutInterval = 30

    return request
  }

  private func decodeResponse<Response: Decodable>(
    _: Response.Type,
    data: Data,
    response: URLResponse
  ) throws -> Response {
    try validateResponse(data: data, response: response)
    return try Self.decoder.decode(Response.self, from: data)
  }

  private func validateResponse(
    data: Data,
    response: URLResponse
  ) throws {
    guard let httpResponse = response as? HTTPURLResponse else {
      throw WoodGateError(message: "The server response was invalid.")
    }

    guard (200 ... 299).contains(httpResponse.statusCode) else {
      if let problem = try? Self.decoder.decode(WoodGateProblemResponse.self, from: data) {
        throw WoodGateAPIError(statusCode: httpResponse.statusCode, detail: problem.detail)
      }

      throw WoodGateAPIError(
        statusCode: httpResponse.statusCode,
        detail: HTTPURLResponse.localizedString(forStatusCode: httpResponse.statusCode).capitalized
      )
    }
  }

  private func appendField(
    _ name: String,
    value: String,
    to body: inout Data,
    boundary: String
  ) {
    body.append(Data("--\(boundary)\r\n".utf8))
    body.append(Data("Content-Disposition: form-data; name=\"\(name)\"\r\n\r\n".utf8))
    body.append(Data("\(value)\r\n".utf8))
  }

  private func appendFile(
    field: String,
    filename: String,
    mimeType: String,
    data: Data,
    to body: inout Data,
    boundary: String
  ) {
    body.append(Data("--\(boundary)\r\n".utf8))
    body.append(
      Data(
        "Content-Disposition: form-data; name=\"\(field)\"; filename=\"\(filename)\"\r\n".utf8
      )
    )
    body.append(Data("Content-Type: \(mimeType)\r\n\r\n".utf8))
    body.append(data)
    body.append(Data("\r\n".utf8))
  }
}
