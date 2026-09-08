import SwiftUI
import UIKit

struct CheckinHomeView: View {
    @Environment(ModelData.self) private var modelData
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    let session: ActiveSession
    @State private var selectedPerson: PersonSummary?
    @State private var notes = ""
    @State private var selfie: CapturedSelfie?
    @State private var selfieImage: UIImage?
    @State private var isSelfieCapturePresented = false
    @State private var submissionDirection: CheckinDirectionChoice?
    @FocusState private var isNotesFocused: Bool

    private var trimmedNotes: String {
        notes.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private var canSubmit: Bool {
        selectedPerson != nil && !modelData.isBusy
            && (!session.location.notes || !trimmedNotes.isEmpty)
            && (!session.location.photo || selfie != nil)
    }

    var body: some View {
        GeometryReader { viewport in
            ScrollView {
                VStack(spacing: 24) {
                    if let image = session.logoImage {
                        Image(uiImage: image)
                            .resizable()
                            .scaledToFit()
                            .frame(maxWidth: 320, maxHeight: 120)
                            .accessibilityHidden(true)
                    }
                    WallpaperCard {
                        VStack(alignment: .leading, spacing: 20) {
                            Text(session.location.name)
                                .font(.largeTitle.bold())
                                .accessibilityAddTraits(.isHeader)
                            PersonPicker(selection: $selectedPerson, viewportSize: viewport.size)
                                .disabled(submissionDirection != nil)
                                .zIndex(1)
                            if session.location.notes {
                                notesSection.disabled(submissionDirection != nil)
                            }
                            if session.location.photo {
                                photoSection.disabled(submissionDirection != nil)
                            }
                            CheckinActions(
                                direction: $submissionDirection,
                                canSubmit: canSubmit,
                                submit: submit,
                                onFailure: modelData.handleSubmissionFailure
                            )
                        }
                        .padding(horizontalSizeClass == .compact ? 20 : 24)
                    }
                }
                .frame(maxWidth: 720)
                .padding(.horizontal, horizontalSizeClass == .compact ? 16 : 24)
                .padding(.vertical, 24)
                .frame(maxWidth: .infinity)
            }
            .defaultScrollAnchor(.center, for: .alignment)
            .scrollDismissesKeyboard(.interactively)
        }
        .coordinateSpace(name: "checkinForm")
        .sheet(isPresented: $isSelfieCapturePresented) {
            SelfieCaptureSheet { capture in
                selfie = capture
                selfieImage = UIImage(data: capture.jpegData)
            }
            .presentationDetents([.large])
        }
    }

    private var notesSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Note").font(.subheadline.bold()).foregroundStyle(.secondary)
            TextField("Add a note...", text: $notes, axis: .vertical)
                .focused($isNotesFocused)
                .lineLimit(3, reservesSpace: true)
                .padding(14)
                .background(.background, in: .rect(cornerRadius: 18))
        }
    }

    private var photoSection: some View {
        HStack(spacing: 0) {
            Button {
                isNotesFocused = false
                isSelfieCapturePresented = true
            } label: {
                HStack(spacing: 12) {
                    Group {
                        if let selfieImage {
                            Image(uiImage: selfieImage).resizable().scaledToFill()
                        } else {
                            Image(systemName: "camera.fill").font(.title2)
                        }
                    }
                    .frame(width: 52, height: 52)
                    .clipShape(.rect(cornerRadius: 12))
                    Text(selfie == nil ? "Take Photo" : "Photo added")
                        .font(.headline)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
                .padding(12)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityHint(selfie == nil ? "" : "Take a replacement photo")
            if selfie != nil {
                Button("Remove photo", systemImage: "xmark.circle.fill") {
                    selfie = nil
                    selfieImage = nil
                }
                .labelStyle(.iconOnly)
                .buttonStyle(.plain)
                .frame(width: 44, height: 44)
                .padding(.trailing, 8)
            }
        }
        .background(.background, in: .rect(cornerRadius: 18))
    }

    private func submit(_ direction: CheckinDirectionChoice) async throws -> String {
        let person = selectedPerson!
        isNotesFocused = false
        try await modelData.submitCheckin(session: session, person: person, direction: direction, notes: trimmedNotes, selfie: selfie)
        try Task.checkCancellation()
        selectedPerson = nil
        notes = ""
        selfie = nil
        selfieImage = nil
        return "\(person.displayName) was \(direction == .checkIn ? "checked in" : "checked out")."
    }
}

private struct CheckinHomePreview: View {
    let session: ActiveSession
    @State private var modelData: ModelData
    init(session: ActiveSession) {
        self.session = session
        _modelData = State(initialValue: PreviewFixtures.modelData(session: session))
    }

    var body: some View {
        ZStack {
            LocationBackgroundView(image: session.backgroundImage)
            CheckinHomeView(session: session)
        }
        .environment(modelData)
    }
}

#Preview("Check-in - Plain") {
    CheckinHomePreview(session: PreviewFixtures.plainSession)
}

#Preview("Check-in - Wallpaper and Logo") {
    CheckinHomePreview(session: PreviewFixtures.brandedSession)
}
