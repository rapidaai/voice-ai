import type { ReactNode } from 'react';
import { useState } from 'react';
import { Filter } from '@carbon/icons-react';
import { Button, Checkbox, Popover, PopoverContent } from '@carbon/react';
import { cn } from '@/utils';

export interface FilterOption {
  id: string;
  label: string;
}

export interface TableToolbarFilterProps {
  filters: FilterOption[];
  activeFilters: Set<string>;
  onApplyFilter: (activeFilters: Set<string>) => void;
  onResetFilter: () => void;
  className?: string;
  panelClassName?: string;
  extraContent?: ReactNode;
  onApply?: () => boolean | void;
  onReset?: () => void;
  filtersTitle?: string;
}

export function TableToolbarFilter({
  filters,
  activeFilters,
  onApplyFilter,
  onResetFilter,
  className,
  panelClassName,
  extraContent,
  onApply,
  onReset,
  filtersTitle = 'Filter by type',
}: TableToolbarFilterProps) {
  const [open, setOpen] = useState(false);
  const [localFilters, setLocalFilters] = useState<Set<string>>(
    new Set(activeFilters),
  );

  const handleToggle = (id: string) => {
    setLocalFilters(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleApply = () => {
    if (onApply?.() === false) {
      return;
    }
    onApplyFilter(localFilters);
    setOpen(false);
  };

  const handleReset = () => {
    setLocalFilters(new Set());
    onResetFilter();
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
        iconDescription="Filter"
        kind="ghost"
        onClick={() => {
          setLocalFilters(new Set(activeFilters));
          setOpen(!open);
        }}
        className={cn(activeFilters.size > 0 && '!text-support-info')}
        tooltipPosition="bottom"
      />
      {open && (
        <PopoverContent className={cn('w-72 p-0', panelClassName)}>
          <div>
            {extraContent && (
              <div className="px-4 py-3 border-b border-border-subtle ">
                {extraContent}
              </div>
            )}
            {filters.length > 0 && (
              <>
                <div className="px-4 py-2 border-b border-border-subtle ">
                  <p className="text-xs font-medium uppercase tracking-wider text-muted ">
                    {filtersTitle}
                  </p>
                </div>
                <div className="max-h-64 overflow-y-auto py-2">
                  {filters.map(f => (
                    <div key={f.id} className="px-4 py-0.5">
                      <Checkbox
                        id={`filter-${f.id}`}
                        labelText={f.label}
                        checked={localFilters.has(f.id)}
                        onChange={() => handleToggle(f.id)}
                      />
                    </div>
                  ))}
                </div>
              </>
            )}
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
