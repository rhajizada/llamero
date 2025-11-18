"use client";

import { format, startOfToday } from "date-fns";
import { Calendar } from "@/components/ui/calendar";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface TokenExpiryPickerProps {
  value: Date | null;
  onChange: (value: Date | null) => void;
}

const safeDate = (value: Date | null) => {
  if (!value) return undefined;
  return value;
};

export const TokenExpiryPicker = ({ value, onChange }: TokenExpiryPickerProps) => {
  const today = startOfToday();

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className="w-full justify-start text-left font-normal"
          aria-label={value ? `Expiry ${format(value, "PP")}` : "Select expiry date"}
        >
          {value ? format(value, "PP") : <span>Select a date</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <Calendar
          mode="single"
          selected={safeDate(value)}
          onSelect={(date) => onChange(date ?? null)}
          fromDate={today}
          initialFocus
        />
      </PopoverContent>
    </Popover>
  );
};
