import SwiftUI

struct DisplaySettingsSection: View {
    private var settings: AppSettings {
        .shared
    }

    var body: some View {
        Section {
            Toggle("Keep Screen Awake", isOn: Binding(
                get: { settings.displaySettings.preventsSleep },
                set: { settings.displaySettings.keepAwake = $0 }
            ))
            .disabled(settings.displaySettings.schedule.enabled)
            Toggle("Schedule Display", isOn: binding(\.schedule.enabled))

            if settings.displaySettings.schedule.enabled {
                DatePicker("Opens", selection: timeBinding(\.start), displayedComponents: .hourAndMinute)
                DatePicker("Closes", selection: timeBinding(\.end), displayedComponents: .hourAndMinute)
                ForEach(DisplaySchedule.Weekday.allCases) { day in
                    Toggle(Calendar.current.weekdaySymbols[day.calendarWeekday - 1], isOn: dayBinding(day))
                }
                if let message = settings.displaySettings.schedule.validationMessage {
                    Text(message).foregroundStyle(.red)
                }
            }
        } header: {
            Text("Display")
        } footer: {
            VStack(alignment: .leading, spacing: 6) {
                Text("Schedules use this device's time zone and apply after pairing. Outside opening hours, the screen is black and dimmed. The app stays awake and resumes automatically. Overnight hours start on the selected day.")
                Text("Tap the bottom-right corner ten times to open this menu, including while the screen is dimmed.")
            }
        }
    }

    private func binding<Value>(_ keyPath: WritableKeyPath<DisplaySettings, Value>) -> Binding<Value> {
        Binding(
            get: { settings.displaySettings[keyPath: keyPath] },
            set: { settings.displaySettings[keyPath: keyPath] = $0 }
        )
    }

    private func dayBinding(_ day: DisplaySchedule.Weekday) -> Binding<Bool> {
        Binding(
            get: { settings.displaySettings.schedule.days.contains(day) },
            set: { selected in
                var days = settings.displaySettings.schedule.days
                days.removeAll { $0 == day }
                if selected {
                    days.append(day)
                }
                settings.displaySettings.schedule.days = days
            }
        )
    }

    private func timeBinding(_ keyPath: WritableKeyPath<DisplaySchedule, String>) -> Binding<Date> {
        Binding(
            get: {
                let parts = settings.displaySettings.schedule[keyPath: keyPath].split(separator: ":")
                let hour = parts.first.flatMap { Int($0) } ?? 8
                let minute = parts.last.flatMap { Int($0) } ?? 0
                return Calendar.current.date(from: DateComponents(year: 2001, month: 1, day: 1, hour: hour, minute: minute)) ?? .now
            },
            set: { date in
                let hour = Calendar.current.component(.hour, from: date)
                let minute = Calendar.current.component(.minute, from: date)
                settings.displaySettings.schedule[keyPath: keyPath] = String(format: "%02d:%02d", hour, minute)
            }
        )
    }
}
