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
    private let stationSecret: String
    private let session: URLSession

    private static let decoder: JSONDecoder = .init()

    static let stationSubprotocol = "woodgate-station.v1"

    // MARK: - Init

    init(baseURL: URL, stationSecret: String, session: URLSession = .shared) {
        self.baseURL = baseURL
        self.stationSecret = stationSecret
        self.session = session
    }

    // MARK: - Public Methods

    func getConfiguration() async throws -> WoodGateStationConfigurationResponse {
        try await get(path: "/api/station/v1/configuration")
    }

    func listPeople() async throws -> [PersonSummary] {
        let response: WoodGateListResponse<WoodGateUserResponse> = try await get(
            path: "/api/station/v1/people"
        )
        return response.items.map { row in
            PersonSummary(id: row.id, displayName: row.name, email: row.email)
        }
    }

    func createCheckin(
        personID: Int64,
        direction: CheckinDirectionChoice,
        notes: String?,
        photoJPEGData: Data?
    ) async throws -> WoodGateCheckinResponse {
        let boundary = "WoodGateBoundary-\(UUID().uuidString)"
        var body = Data()

        appendField("person_id", value: "\(personID)", to: &body, boundary: boundary)
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

        var request = try makeRequest(path: "/api/station/v1/checkins")
        request.httpMethod = "POST"
        request.setValue(
            "multipart/form-data; boundary=\(boundary)",
            forHTTPHeaderField: "Content-Type"
        )

        let (data, response) = try await session.upload(for: request, from: body)
        return try decodeResponse(WoodGateCheckinResponse.self, data: data, response: response)
    }

    func getLocationBackground() async throws -> Data {
        try await getContent(path: "/api/station/v1/configuration/background")
    }

    func getLocationLogo() async throws -> Data {
        try await getContent(path: "/api/station/v1/configuration/logo")
    }

    private func getContent(path: String) async throws -> Data {
        let request = try makeRequest(path: path)
        let (data, response) = try await session.data(for: request)
        try validateResponse(data: data, response: response)
        return data
    }

    func makeControlTask(appBuild: String) throws -> URLSessionWebSocketTask {
        var request = try makeRequest(path: "/api/station/v1/connect")
        request.setValue(Self.stationSubprotocol, forHTTPHeaderField: "Sec-WebSocket-Protocol")
        request.setValue(appBuild, forHTTPHeaderField: "Woodgate-Station-Build")
        request.timeoutInterval = 30
        return session.webSocketTask(with: request)
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
        request.setValue("Bearer \(stationSecret)", forHTTPHeaderField: "Authorization")
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

            if let response = try? Self.decoder.decode(WoodGateErrorResponse.self, from: data) {
                throw WoodGateAPIError(statusCode: httpResponse.statusCode, detail: response.error)
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
