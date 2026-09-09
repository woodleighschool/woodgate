import { format } from "date-fns";
import { CalendarDays, X } from "lucide-react";
import type { DateRange } from "react-day-picker";

import { Button } from "@components/ui/button";
import { ButtonGroup } from "@components/ui/button-group";
import { Calendar } from "@components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@components/ui/popover";
import { Separator } from "@components/ui/separator";

export function DateRangePicker({
  value,
  onValueChange,
  label = "Date Range",
  disabled,
  defaultMonth,
  valueLabel,
  onToday,
}: {
  value?: DateRange;
  onValueChange: (value: DateRange | undefined) => void;
  label?: string;
  disabled?: React.ComponentProps<typeof Calendar>["disabled"];
  defaultMonth?: Date;
  valueLabel?: string;
  onToday?: () => void;
}) {
  const selected = value?.from !== undefined;

  return (
    <ButtonGroup>
      <Popover>
        <PopoverTrigger
          render={<Button variant="outline" size="sm" className="h-8 border-dashed font-normal" />}
        >
          <CalendarDays data-icon="inline-start" />
          {label}
          {selected ? (
            <>
              <Separator orientation="vertical" className="mx-0.5 h-4" />
              {valueLabel ?? formatRange(value)}
            </>
          ) : null}
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="start">
          {onToday ? (
            <div className="border-b p-2">
              <Button variant="ghost" size="sm" onClick={onToday}>
                Today
              </Button>
            </div>
          ) : null}
          <Calendar
            mode="range"
            defaultMonth={value?.from ?? defaultMonth}
            selected={value}
            onSelect={onValueChange}
            numberOfMonths={2}
            disabled={disabled}
          />
        </PopoverContent>
      </Popover>
      {selected ? (
        <Button
          type="button"
          variant="outline"
          size="icon-sm"
          className="size-8"
          aria-label={`Clear ${label.toLowerCase()}`}
          onClick={() => onValueChange(undefined)}
        >
          <X />
        </Button>
      ) : null}
    </ButtonGroup>
  );
}

function formatRange(value: DateRange | undefined): string {
  if (!value?.from) return "";
  if (!value.to) return format(value.from, "MMM d, yyyy");
  return `${format(value.from, "MMM d, yyyy")} - ${format(value.to, "MMM d, yyyy")}`;
}
