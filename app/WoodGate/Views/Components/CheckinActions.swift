import SwiftUI

struct CheckinActions: View {
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var direction: CheckinDirectionChoice?
    let canSubmit: Bool
    let submit: (CheckinDirectionChoice) async throws -> String
    let onFailure: (Error) -> Void

    private struct Outcome {
        let message: String
        let error: Error?
    }

    @State private var outcome: Outcome?
    @State private var expanded = false
    @State private var expansionFinished = false
    @State private var isVisible = false
    @ScaledMetric(relativeTo: .headline) private var buttonHeight = 88

    private var isVertical: Bool {
        horizontalSizeClass == .compact
    }

    private var showsOutcome: Bool {
        expansionFinished && outcome != nil
    }

    private var animation: Animation? {
        reduceMotion ? nil : .spring(response: 0.6, dampingFraction: 0.88)
    }

    var body: some View {
        GeometryReader { geometry in
            let width = geometry.size.width
            let idleWidth = isVertical ? width : max(0, (width - 12) / 2)
            ZStack(alignment: .topLeading) {
                ForEach(CheckinDirectionChoice.allCases) { action in
                    let isActive = direction == action
                    let fillsArea = isActive && expanded
                    Button {
                        start(action)
                    } label: {
                        label(for: action)
                            .animation(.easeInOut(duration: 0.15), value: showsOutcome)
                            .frame(maxWidth: .infinity, maxHeight: .infinity)
                            .padding(.horizontal, 12)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.white)
                    .frame(
                        width: fillsArea ? width : idleWidth,
                        height: fillsArea ? geometry.size.height : buttonHeight
                    )
                    .background(
                        action == .checkIn ? Color.green : Color.red,
                        in: .rect(cornerRadius: 22)
                    )
                    .offset(
                        x: !isVertical && action == .checkOut && !fillsArea ? idleWidth + 12 : 0,
                        y: isVertical && action == .checkOut && !fillsArea ? buttonHeight + 12 : 0
                    )
                    .opacity(direction == nil ? (canSubmit ? 1 : 0.55) : (isActive ? 1 : (expanded ? 0 : 1)))
                    .zIndex(isActive ? 1 : 0)
                    .disabled(direction != nil || !canSubmit)
                    .accessibilityHidden(direction != nil && !isActive)
                    .accessibilityRemoveTraits(isActive ? .isButton : [])
                }
            }
        }
        .frame(height: isVertical ? buttonHeight * 2 + 12 : buttonHeight)
        .onAppear { isVisible = true }
        .onDisappear {
            isVisible = false
            direction = nil
            expanded = false
            expansionFinished = false
            outcome = nil
        }
        .onChange(of: showsOutcome) { _, shown in
            if shown, let outcome {
                AccessibilityNotification.Announcement(outcome.message).post()
            }
        }
        .task(id: direction) {
            guard let direction else { return }
            do {
                let message = try await submit(direction)
                try Task.checkCancellation()
                outcome = Outcome(message: message, error: nil)
            } catch {
                guard !Task.isCancelled else { return }
                outcome = Outcome(message: error.localizedDescription, error: error)
            }
        }
        .task(id: showsOutcome) {
            guard showsOutcome else { return }
            do {
                try await Task.sleep(for: .seconds(1.8))
            } catch { return }
            let error = outcome?.error
            expansionFinished = false
            withAnimation(animation, completionCriteria: .removed) {
                expanded = false
            } completion: {
                guard isVisible else { return }
                outcome = nil
                direction = nil
                if let error {
                    onFailure(error)
                }
            }
        }
    }

    @ViewBuilder
    private func label(for action: CheckinDirectionChoice) -> some View {
        if direction == action, showsOutcome, let outcome {
            Label(outcome.message, systemImage: outcome.error == nil ? "checkmark.circle.fill" : "exclamationmark.circle.fill")
                .font(.headline)
                .multilineTextAlignment(.center)
                .lineLimit(3)
                .minimumScaleFactor(0.7)
                .accessibilityLabel(outcome.error == nil ? "Success: \(outcome.message)" : "Error: \(outcome.message)")
        } else {
            HStack(spacing: 10) {
                if direction == action, expansionFinished {
                    ProgressView().tint(.white)
                } else {
                    Image(systemName: action == .checkIn ? "figure.walk.arrival" : "figure.walk.departure")
                }
                Text(direction == action && expansionFinished
                    ? (action == .checkIn ? "Checking in..." : "Checking out...")
                    : (action == .checkIn ? "Check In" : "Check Out"))
            }
            .font(.title3.bold())
        }
    }

    private func start(_ action: CheckinDirectionChoice) {
        direction = action
        withAnimation(animation, completionCriteria: .removed) {
            expanded = true
        } completion: {
            guard isVisible else { return }
            expansionFinished = true
        }
    }
}
