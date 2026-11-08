import { useState } from 'react';
import { Filter } from '@carbon/icons-react';
import {
  Button,
  DatePicker as CarbonDatePicker,
  DatePickerInput,
  Popover,
  PopoverContent,
} from '@carbon/react';
import { cn } from '@/utils';

export interface DateFilterProps {
  onApply: (from: Date, to: Date) => void;
  onReset?: () => void;
  className?: string;
}

export function DateFilter({ onApply, onReset, className }: DateFilterProps) {
  const [open, setOpen] = useState(false);
  const [dates, setDates] = useState<Date[]>([]);

  const handleApply = () => {
    if (dates.length === 2) {
      onApply(dates[0], dates[1]);
    }
    setOpen(false);
  };

  const handleReset = () => {
    setDates([]);
    onReset?.();
    setOpen(false);
  };

  return (
    <Popover
      as="div"
      align="bottom-end"
      caret={false}
      className={cn('relative flex items-center', className)}
      onRequestClose={() => setOpen(false)}
      open={open}
    >
      <Button
        hasIconOnly
        renderIcon={Filter}
        iconDescription="Filter by date"
        kind="ghost"
        onClick={() => setOpen(!open)}
        className={cn(dates.length === 2 && '!text-support-info')}
        tooltipPosition="bottom"
      />
      {open && (
        <PopoverContent className="w-max p-0">
          <div>
            <div className="px-4 py-2 border-b border-border-subtle ">
              <p className="text-xs font-medium uppercase tracking-wider text-muted ">
                Filter by date
              </p>
            </div>
            <div className="p-4">
              <CarbonDatePicker
                datePickerType="range"
                dateFormat="m/d/Y"
                value={dates.length > 0 ? dates : undefined}
                onChange={(selectedDates: Date[]) => setDates(selectedDates)}
              >
                <DatePickerInput
                  id="date-filter-from"
                  placeholder="Start date"
                  labelText="From"
                  size="md"
                />
                <DatePickerInput
                  id="date-filter-to"
                  placeholder="End date"
                  labelText="To"
                  size="md"
                />
              </CarbonDatePicker>
            </div>
            <div className="flex border-t border-border-subtle ">
              <Button
                size="md"
                kind="secondary"
                className="!flex-1 !max-w-none !rounded-none"
                onClick={handleReset}
              >
                Reset
              </Button>
              <Button
                size="md"
                kind="primary"
                className="!flex-1 !max-w-none !rounded-none"
                onClick={handleApply}
              >
                Apply
              </Button>
            </div>
          </div>
        </PopoverContent>
      )}
    </Popover>
  );
}
