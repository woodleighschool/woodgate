import SwiftUI

struct PersonPicker: View {
    @Environment(ModelData.self) private var modelData
    @Environment(\.isEnabled) private var isEnabled
    @Binding var selection: PersonSummary?
    let viewportSize: CGSize
    @State private var query = ""
    @State private var results: [PersonSummary] = []
    @State private var completedQuery = ""
    @FocusState private var isFocused: Bool
    @ScaledMetric(relativeTo: .body) private var fieldHeight = 68

    private var searchQuery: String {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        return isFocused && isEnabled && selection == nil && trimmed.count >= 2 ? trimmed : ""
    }

    private var showsResults: Bool {
        !searchQuery.isEmpty && completedQuery == searchQuery
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Name").font(.subheadline.bold()).foregroundStyle(.secondary)
            HStack(spacing: 10) {
                if let person = selection {
                    VStack(alignment: .leading, spacing: 4) {
                        Text(person.displayName).font(.body.weight(.semibold))
                        Text(person.email).font(.caption).foregroundStyle(.secondary)
                    }
                    .lineLimit(1)
                    .frame(maxWidth: .infinity, alignment: .leading)
                } else {
                    Image(systemName: "magnifyingglass").foregroundStyle(.secondary)
                    TextField("Search by name or email", text: $query)
                        .focused($isFocused)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .accessibilityLabel("Name")
                }
                if selection != nil || !query.isEmpty {
                    Button(selection == nil ? "Clear search" : "Clear selected person", systemImage: "xmark.circle.fill") {
                        selection = nil
                        query = ""
                        isFocused = true
                    }
                    .labelStyle(.iconOnly)
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                    .frame(width: 44, height: 44)
                }
            }
            .padding(.horizontal, 14)
            .frame(height: fieldHeight)
            .background(.background, in: .rect(cornerRadius: 18))
            .overlay(alignment: .top) {
                if showsResults {
                    suggestions.padding(.top, fieldHeight + 8)
                }
            }
        }
        .background {
            if isFocused {
                GeometryReader { proxy in
                    let frame = proxy.frame(in: .named("checkinForm"))
                    Color.clear
                        .frame(width: viewportSize.width, height: viewportSize.height)
                        .contentShape(Rectangle())
                        .onTapGesture { isFocused = false }
                        .offset(x: -frame.minX, y: -frame.minY)
                        .accessibilityHidden(true)
                }
            }
        }
        .task(id: searchQuery) {
            results = []
            completedQuery = ""
            let query = searchQuery
            guard !query.isEmpty else { return }
            do {
                try await Task.sleep(for: .milliseconds(180))
                results = modelData.searchPeople(matching: query)
                completedQuery = query
            } catch {}
        }
        .onChange(of: isEnabled) { _, enabled in
            if !enabled {
                isFocused = false
            }
        }
        .onChange(of: selection) { _, _ in query = "" }
    }

    private var suggestions: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                if results.isEmpty {
                    Text("No matches found").foregroundStyle(.secondary)
                        .padding(16).frame(maxWidth: .infinity, alignment: .leading)
                }
                ForEach(results) { person in
                    Button {
                        selection = person
                        isFocused = false
                    } label: {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(person.displayName).font(.body.weight(.semibold))
                            Text(person.email).font(.caption).foregroundStyle(.secondary)
                        }
                        .lineLimit(1)
                        .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                        .padding(12)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .overlay(alignment: .bottom) { Divider() }
                }
            }
        }
        .frame(height: results.isEmpty ? 56 : min(CGFloat(results.count) * 68, 260))
        .background(.regularMaterial, in: .rect(cornerRadius: 18))
        .clipShape(.rect(cornerRadius: 18))
        .shadow(color: .black.opacity(0.15), radius: 16, y: 6)
    }
}
