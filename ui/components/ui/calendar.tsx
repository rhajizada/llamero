"use client";

import * as React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { DayPicker } from "react-day-picker";
import { cn } from "@/lib/utils";

export type CalendarProps = React.ComponentProps<typeof DayPicker>;

function Calendar({ className, classNames, showOutsideDays = true, ...props }: CalendarProps) {
  return (
    <DayPicker
      showOutsideDays={showOutsideDays}
      className={cn("p-4", className)}
      classNames={{
        months: "grid grid-cols-1 gap-4",
        month: "space-y-4",
        caption: "flex items-center justify-between px-2",
        caption_label: "text-sm font-medium",
        nav: "flex items-center gap-2",
        button_previous: "h-8 w-8 rounded-full border border-border hover:bg-muted",
        button_next: "h-8 w-8 rounded-full border border-border hover:bg-muted",
        table: "w-full border-collapse",
        head_row: "grid grid-cols-7 text-center text-[0.7rem] text-muted-foreground",
        head_cell: "font-normal",
        row: "grid grid-cols-7 text-center text-sm",
        cell: cn(
          "relative h-10 w-10 text-center",
          "[&:has([aria-selected])]:rounded-full [&:has([aria-selected])]:bg-foreground [&:has([aria-selected])]:text-background",
          classNames?.cell,
        ),
        day: cn(
          "inline-flex h-10 w-10 items-center justify-center rounded-full text-sm font-normal text-foreground",
          "hover:bg-muted aria-selected:opacity-100",
          classNames?.day,
        ),
        day_outside: "text-muted-foreground opacity-50",
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
