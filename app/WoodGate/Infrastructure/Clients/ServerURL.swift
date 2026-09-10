import Foundation

enum ServerURL {
    static func parse(_ value: String) -> URL? {
        guard let url = URL(string: value),
              let scheme = url.scheme?.lowercased(),
              ["http", "https"].contains(scheme),
              let host = url.host, !host.isEmpty,
              url.user == nil, url.password == nil,
              url.query == nil, url.fragment == nil else { return nil }
        return url
    }

    static func resolve(_ value: String, session: URLSession = .shared) async throws -> URL {
        let value = value.trimmingCharacters(in: .whitespacesAndNewlines)
        if value.contains("://") {
            guard let url = parse(value) else { throw URLError(.badURL) }
            return url
        }

        guard let url = parse("http://" + value) else { throw URLError(.badURL) }
        var request = URLRequest(url: url)
        request.httpMethod = "HEAD"
        request.timeoutInterval = 10

        let response: URLResponse
        do {
            (_, response) = try await session.data(for: request)
        } catch let error as URLError where [.cannotConnectToHost, .timedOut].contains(error.code) {
            request.url = parse("https://" + value)
            (_, response) = try await session.data(for: request)
        }

        guard let resolvedURL = response.url,
              let resolved = URLComponents(url: resolvedURL, resolvingAgainstBaseURL: false),
              var components = URLComponents(url: url, resolvingAgainstBaseURL: false)
        else {
            throw URLError(.badURL)
        }

        // Redirects can lead to a web login page; retain the entered API base path.
        components.scheme = resolved.scheme
        components.host = resolved.host
        components.port = resolved.port
        guard let result = components.url.flatMap({ parse($0.absoluteString) }) else { throw URLError(.badURL) }
        return result
    }
}
