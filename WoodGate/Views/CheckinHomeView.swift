//
//  CheckinHomeView.swift
//  WoodGate
//
//  Created by Alexander Hyde on 13/3/2026.
//

import SwiftUI
import UIKit

struct CheckinHomeView: View {
  // MARK: - Properties

  @Environment(ModelData.self) private var modelData

  let session: ActiveSession

  private struct FormState {
    var query = ""
    var selectedPersonID: UUID?
    var notes = ""
    var selfie: CapturedSelfie?
  }

  @State private var form = FormState()
  @State private var isSelfieCapturePresented = false
  @State private var isSearchPopoverPresented = false
  @FocusState private var isSearchFocused: Bool
  @FocusState private var isNotesFocused: Bool

  // MARK: - Computed Properties

  private var selectedPerson: PersonSummary? {
    session.people.first(where: { $0.id == form.selectedPersonID })
  }

  private var searchResults: [PersonSummary] {
    guard !form.query.isEmpty else { return [] }
    return modelData.searchPeople(matching: form.query)
  }

  private var trimmedNotes: String {
    form.notes.trimmingCharacters(in: .whitespacesAndNewlines)
  }

  private var canSubmit: Bool {
    guard selectedPerson != nil else { return false }
    guard !session.location.notes || !trimmedNotes.isEmpty else { return false }
    guard !modelData.isBusy else { return false }

    if session.location.photo {
      return form.selfie != nil
    }

    return true
  }

  private var submissionHint: String? {
    if selectedPerson == nil {
      return "Search and choose your name to continue."
    }

    if session.location.notes, trimmedNotes.isEmpty {
      return "Add notes to continue."
    }

    if session.location.photo, form.selfie == nil {
      return "Add a selfie to continue."
    }

    return nil
  }

  // MARK: - Body

  var body: some View {
    VStack(spacing: 0) {
      VStack {
        Spacer()
        logoSection
        Spacer()
      }
      .frame(maxWidth: .infinity, maxHeight: .infinity)

      checkinCard
        .padding(.horizontal)

      Color.clear
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
    .ignoresSafeArea(.keyboard, edges: .bottom)
    .onTapGesture {
      isSearchFocused = false
      isNotesFocused = false
    }
    .sheet(isPresented: $isSelfieCapturePresented) {
      SelfieCaptureSheet { selfie in
        form.selfie = selfie
      }
      .presentationDetents([.large])
    }
    .onChange(of: session.location.id, initial: true) { _, _ in
      resetForm()
    }
    .onChange(of: form.query) { _, newValue in
      guard let selectedPerson, newValue != selectedPerson.displayName else { return }
      form = FormState(query: newValue)
    }
    .onChange(of: form.query) { _, _ in
      updateSearchPopoverPresentation()
    }
    .onChange(of: isSearchFocused) { _, _ in
      updateSearchPopoverPresentation()
    }
  }

  // MARK: - View Builders

  private var checkinCard: some View {
    VStack(alignment: .leading, spacing: 20) {
      headerSection
      personSelectionSection
      detailsSection
      submissionHintSection
      submitButtons
    }
    .padding(24)
    .glassEffect(in: .rect(cornerRadius: 28))
    .contentShape(Rectangle())
  }

  private var logoSection: some View {
    LocationLogoView(image: session.logoImage)
      .frame(maxWidth: 320, maxHeight: 120)
      .shadow(color: .black.opacity(0.15), radius: 12, y: 4)
  }

  private var headerSection: some View {
    HStack(alignment: .center, spacing: 12) {
      Text(session.location.name)
        .font(.system(size: 34, weight: .bold, design: .rounded))
        .lineLimit(1)
        .minimumScaleFactor(0.55)
        .allowsTightening(true)

      Spacer(minLength: 0)

      if session.isDemo {
        HStack(spacing: 16) {
          Toggle(
            "Notes",
            isOn: Binding(
              get: { session.location.notes },
              set: { _ in
                modelData.toggleDemoNotes()
                if !session.location.notes { form.notes = "" }
              }
            )
          )

          Toggle(
            "Selfie",
            isOn: Binding(
              get: { session.location.photo },
              set: { _ in
                modelData.toggleDemoPhoto()
                if !session.location.photo { form.selfie = nil }
              }
            )
          )
        }
        .font(.system(size: 13, weight: .medium, design: .rounded))
        .fixedSize()
      }
    }
  }

  private var personSelectionSection: some View {
    VStack(alignment: .leading, spacing: 12) {
      Text("Person")
        .font(.system(size: 15, weight: .bold, design: .rounded))
        .foregroundStyle(.secondary)

      HStack(spacing: 10) {
        Image(systemName: "magnifyingglass")
          .foregroundStyle(.secondary)

        TextField("Search by name or email", text: $form.query)
          .focused($isSearchFocused)

        if !form.query.isEmpty {
          Button {
            form = FormState()
            isSearchFocused = true
          } label: {
            Image(systemName: "xmark.circle.fill")
              .foregroundStyle(.secondary)
          }
          .buttonStyle(.plain)
        }
      }
      .padding(.horizontal, 14)
      .padding(.vertical, 12)
      .background(
        RoundedRectangle(cornerRadius: 18, style: .continuous)
          .fill(.thickMaterial)
      )
      .popover(
        isPresented: $isSearchPopoverPresented,
        attachmentAnchor: .rect(.bounds),
        arrowEdge: .top
      ) {
        searchResultsPopover
          .presentationCompactAdaptation(.none)
      }
    }
  }

  private var searchResultsPopover: some View {
    List(searchResults) { person in
      searchResultRow(person)
    }
    .listStyle(.plain)
    .frame(minWidth: 340, maxWidth: 420, minHeight: 125, maxHeight: 360)
    .overlay {
      if searchResults.isEmpty {
        Text("No matches found")
          .font(.system(size: 15, weight: .medium, design: .rounded))
          .foregroundStyle(.secondary)
      }
    }
  }

  private var detailsSection: some View {
    VStack(alignment: .leading, spacing: 16) {
      if session.location.notes {
        notesSection
      }

      if session.location.photo {
        selfieSection
      }
    }
  }

  private var notesSection: some View {
    VStack(alignment: .leading, spacing: 8) {
      Text("Notes")
        .font(.system(size: 15, weight: .bold, design: .rounded))
        .foregroundStyle(.secondary)

      TextField("Add notes...", text: $form.notes, axis: .vertical)
        .focused($isNotesFocused)
        .lineLimit(3 ... 6)
        .padding(12)
        .background(
          RoundedRectangle(cornerRadius: 18, style: .continuous)
            .fill(.thickMaterial)
        )
    }
  }

  private var selfieSection: some View {
    VStack(alignment: .leading, spacing: 8) {
      Text("Selfie")
        .font(.system(size: 15, weight: .bold, design: .rounded))
        .foregroundStyle(.secondary)

      Button {
        isSearchFocused = false
        isNotesFocused = false
        isSearchPopoverPresented = false
        isSelfieCapturePresented = true
      } label: {
        if let selfie = form.selfie, let uiImage = UIImage(data: selfie.jpegData) {
          Image(uiImage: uiImage)
            .resizable()
            .scaledToFill()
            .frame(width: 88, height: 88)
            .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
            .overlay(alignment: .bottom) {
              Text("Retake")
                .font(.system(size: 11, weight: .bold, design: .rounded))
                .foregroundStyle(.white)
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Capsule().fill(.black.opacity(0.6)))
                .padding(.bottom, 6)
            }
        } else {
          VStack(spacing: 6) {
            Image(systemName: "camera.fill")
              .font(.system(size: 24))
            Text("Add Photo")
              .font(.system(size: 12, weight: .bold, design: .rounded))
          }
          .foregroundStyle(.secondary)
          .frame(width: 88, height: 88)
          .background(
            RoundedRectangle(cornerRadius: 18, style: .continuous)
              .fill(.thickMaterial)
          )
        }
      }
      .buttonStyle(.plain)
    }
  }

  private var submissionHintSection: some View {
    Group {
      if let submissionHint {
        Label(submissionHint, systemImage: "info.circle.fill")
          .font(.system(size: 14, weight: .medium, design: .rounded))
          .foregroundStyle(.secondary)
      }
    }
  }

  private var submitButtons: some View {
    HStack(spacing: 12) {
      actionButton(
        title: "Check In",
        systemImage: "figure.walk.arrival",
        tint: .green,
        direction: .checkIn
      )
      actionButton(
        title: "Check Out",
        systemImage: "figure.walk.departure",
        tint: .red,
        direction: .checkOut
      )
    }
  }

  // MARK: - Private Helpers

  private func searchResultRow(_ person: PersonSummary) -> some View {
    Button(person.displayName) {
      form.selectedPersonID = person.id
      form.query = person.displayName
      isSearchFocused = false
      isSearchPopoverPresented = false
    }
  }

  private func actionButton(
    title: String,
    systemImage: String,
    tint: Color,
    direction: CheckinDirectionChoice
  ) -> some View {
    Button {
      Task {
        await submit(direction)
      }
    } label: {
      HStack(spacing: 10) {
        if modelData.isBusy {
          ProgressView()
            .tint(.white)
        } else {
          Image(systemName: systemImage)
            .font(.system(size: 22, weight: .bold))
        }

        Text(title)
          .font(.system(size: 22, weight: .bold, design: .rounded))
      }
      .frame(maxWidth: .infinity)
      .frame(minHeight: 72)
      .foregroundStyle(.white)
      .background(
        RoundedRectangle(cornerRadius: 22, style: .continuous)
          .fill(tint)
      )
      .opacity(canSubmit ? 1 : 0.45)
    }
    .buttonStyle(.plain)
    .disabled(!canSubmit)
  }

  private func submit(_ direction: CheckinDirectionChoice) async {
    guard let selectedPerson else { return }

    do {
      try await modelData.submitCheckin(
        person: selectedPerson,
        direction: direction,
        notes: form.notes,
        selfie: form.selfie
      )

      resetForm()
    } catch {
      modelData.handleSubmissionFailure(error)
    }
  }

  private func resetForm() {
    form = FormState()
    isSearchFocused = false
    isNotesFocused = false
    isSearchPopoverPresented = false
    isSelfieCapturePresented = false
  }

  private func updateSearchPopoverPresentation() {
    let trimmedQuery = form.query.trimmingCharacters(in: .whitespacesAndNewlines)
    isSearchPopoverPresented = isSearchFocused && !trimmedQuery.isEmpty
  }
}
