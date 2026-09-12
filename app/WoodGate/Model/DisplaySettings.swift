import Foundation

struct DisplaySettings: Codable, Equatable {
    var keepAwake = true
    var schedule = DisplaySchedule()

    var preventsSleep: Bool {
        keepAwake || schedule.enabled
    }
}

struct DisplaySchedule: Codable, Equatable {
    var enabled = false
    var days: [Weekday] = [.monday, .tuesday, .wednesday, .thursday, .friday]
    var start = "08:00"
    var end = "16:00"

    enum Weekday: String, Codable, CaseIterable, Identifiable {
        case monday = "mon"
        case tuesday = "tue"
        case wednesday = "wed"
        case thursday = "thu"
        case friday = "fri"
        case saturday = "sat"
        case sunday = "sun"

        var id: Self {
            self
        }

        var calendarWeekday: Int {
            switch self {
            case .sunday: 1
            case .monday: 2
            case .tuesday: 3
            case .wednesday: 4
            case .thursday: 5
            case .friday: 6
            case .saturday: 7
            }
        }
    }

    var validationMessage: String? {
        guard !days.isEmpty else { return "Select at least one day." }
        guard Self.minute(start) != nil, Self.minute(end) != nil else {
            return "Use 24-hour times in HH:mm format."
        }
        guard start != end else { return "Start and end times must be different." }
        return nil
    }

    func isOpen(at date: Date, calendar: Calendar = .current) -> Bool {
        guard enabled, validationMessage == nil,
              let startMinute = Self.minute(start), let endMinute = Self.minute(end)
        else { return true }

        let minute = calendar.component(.hour, from: date) * 60 + calendar.component(.minute, from: date)
        let weekday = calendar.component(.weekday, from: date)
        if startMinute < endMinute {
            return includes(weekday) && minute >= startMinute && minute < endMinute
        }
        // Overnight hours belong to the day on which the interval starts.
        let previousWeekday = weekday == 1 ? 7 : weekday - 1
        return (includes(weekday) && minute >= startMinute)
            || (includes(previousWeekday) && minute < endMinute)
    }

    private func includes(_ weekday: Int) -> Bool {
        days.contains { $0.calendarWeekday == weekday }
    }

    private static func minute(_ time: String) -> Int? {
        let parts = time.split(separator: ":", omittingEmptySubsequences: false)
        guard parts.count == 2, parts.allSatisfy({ $0.count == 2 && $0.utf8.allSatisfy { (48 ... 57).contains($0) } }),
              let hour = Int(parts[0]), let minute = Int(parts[1]),
              (0 ... 23).contains(hour), (0 ... 59).contains(minute)
        else { return nil }
        return hour * 60 + minute
    }
}
