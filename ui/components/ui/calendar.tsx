"use client";

import * as React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { DayPicker } from "react-day-picker";
import { cn } from "@/lib/utils";

export type CalendarProps = React.ComponentProps<typeof DayPicker>;

function Calendar({
  className,
  classNames,
  showOutsideDays = true,
  ...props
}: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("p-4", className)}
      classNames={{
        months: "flex flex-col gap-4",
        month: "space-y-4",
        caption: "flex items-center justify-between px-2",
        caption_label: "text-sm font-medium",
        nav: "flex items-center gap-2",
        button_previous:
          "h-8 w-8 rounded-full border border-border hover:bg-muted inline-flex items-center justify-center",
        button_next:
          "h-8 w-8 rounded-full border border-border hover:bg-muted inline-flex items-center justify-center",
        table: "w-full border-collapse space-y-1",
        head_row: "flex w-full",
        head_cell: "text-muted-foreground rounded-md w-9 font-semibold text-[0.8rem]",
        row: "flex w-full mt-1",
        cell: cn(
          "relative h-9 w-9 text-center text-sm p-0",
          "[&:has([aria-selected])]:bg-foreground [&:has([aria-selected])]:text-background",
          "[&:has([aria-selected][data-selection-start])]:rounded-s-full",
          "[&:has([aria-selected][data-selection-end])]:rounded-e-full",
          classNames?.cell,
        ),
        day: cn(
          "h-9 w-9 rounded-md p-0 font-normal text-foreground",
          "hover:bg-muted aria-selected:opacity-100",
          classNames?.day,
        ),
        day_outside: "text-muted-foreground/60",
        day_disabled: "text-muted-foreground/50",
        day_today: "font-semibold",
      }}
      components={{
        Chevron: ({ orientation }) =>
          orientation === "left" ? (
            <ChevronLeft className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          ),
      }}
      {...props}
    />
  );
}
Calendar.displayName = "Calendar";

export { Calendar };
