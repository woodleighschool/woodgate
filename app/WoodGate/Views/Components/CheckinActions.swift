import SwiftUI

struct CheckinActions: View {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Binding var direction: CheckinDirectionChoice?
    let canSubmit: Bool
    let submit: (CheckinDirectionChoice) async throws -> String
    let onFailure: (Error) -> Void

    private struct Outcome {
        let message: String
        let error: Error?
    }

    private enum Phase {
        case idle
        case submitting
        case outcome(Outcome)
        case collapsing

        var isExpanded: Bool {
            switch self {
            case .submitting, .outcome: true
            case .idle, .collapsing: false
            }
        }

        var outcome: Outcome? {
            guard case let .outcome(outcome) = self else { return nil }
            return outcome
        }
    }

    @State private var phase = Phase.idle
    @ScaledMetric(relativeTo: .headline) private var buttonHeight = 88

    private var showsOutcome: Bool {
        phase.outcome != nil
    }

    private var animation: Animation? {
        reduceMotion ? nil : .spring(response: 0.6, dampingFraction: 0.88)
    }

    var body: some View {
        GeometryReader { geometry in
            let width = geometry.size.width
            let idleWidth = max(0, (width - 12) / 2)
            ZStack(alignment: .topLeading) {
                ForEach(CheckinDirectionChoice.allCases) { action in
                    let isActive = direction == action
                    let fillsArea = isActive && phase.isExpanded
                    Button {
                        start(action)
                    } label: {
                        ZStack {
                            label(for: action)
                                .transition(.opacity)
                        }
                        .foregroundStyle(.white)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                        .padding(.horizontal, 12)
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .frame(
                        width: fillsArea ? width : idleWidth,
                        height: fillsArea ? geometry.size.height : buttonHeight
                    )
                    .background(
                        action == .checkIn ? Color.green : Color.red,
                        in: .rect(cornerRadius: 22)
                    )
                    .offset(
                        x: action == .checkOut && !fillsArea ? idleWidth + 12 : 0
                    )
                    .opacity(phase.isExpanded ? (isActive ? 1 : 0) : (canSubmit ? 1 : 0.55))
                    .zIndex(isActive ? 1 : 0)
                    .disabled(direction != nil || !canSubmit)
                    .accessibilityHidden(direction != nil && !isActive)
                    .accessibilityRemoveTraits(isActive ? .isButton : [])
                }
            }
        }
        .frame(height: buttonHeight)
        .onDisappear {
            direction = nil
            phase = .idle
        }
        .onChange(of: showsOutcome) { _, shown in
            if shown, let outcome = phase.outcome {
                AccessibilityNotification.Announcement(outcome.message).post()
            }
        }
        .task(id: direction) {
            guard let direction else { return }
            do {
                let message = try await submit(direction)
                try Task.checkCancellation()
                withAnimation(animation) {
                    phase = .outcome(Outcome(message: message, error: nil))
                }
            } catch {
                guard !Task.isCancelled else { return }
                withAnimation(animation) {
                    phase = .outcome(Outcome(message: error.localizedDescription, error: error))
                }
            }
        }
        .task(id: showsOutcome) {
            guard showsOutcome else { return }
            do {
                try await Task.sleep(for: .seconds(1.8))
            } catch { return }
            let error = phase.outcome?.error
            withAnimation(animation, completionCriteria: .removed) {
                phase = .collapsing
            } completion: {
                guard case .collapsing = phase else { return }
                phase = .idle
                direction = nil
                if let error {
                    onFailure(error)
                }
            }
        }
    }

    @ViewBuilder
    private func label(for action: CheckinDirectionChoice) -> some View {
        if direction == action, let outcome = phase.outcome {
            Label(outcome.message, systemImage: outcome.error == nil ? "checkmark.circle.fill" : "exclamationmark.circle.fill")
                .font(.headline)
                .multilineTextAlignment(.center)
                .lineLimit(3)
                .minimumScaleFactor(0.7)
                .accessibilityLabel(outcome.error == nil ? "Success: \(outcome.message)" : "Error: \(outcome.message)")
        } else if direction == action, phase.isExpanded {
            HStack(spacing: 10) {
                ProgressView().tint(.white)
                Text(action == .checkIn ? "Checking in..." : "Checking out...")
            }
            .font(.title3.bold())
        } else {
            Label(
                action == .checkIn ? "Check In" : "Check Out",
                systemImage: action == .checkIn ? "figure.walk.arrival" : "figure.walk.departure"
            )
            .font(.title3.bold())
        }
    }

    private func start(_ action: CheckinDirectionChoice) {
        guard direction == nil, canSubmit else { return }
        withAnimation(animation) {
            direction = action
            phase = .submitting
        }
    }
}
