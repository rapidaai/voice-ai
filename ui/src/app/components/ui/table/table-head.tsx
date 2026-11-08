import { InputCheckbox } from '@/app/components/ui/primitives/input-checkbox';
import { cn } from '@/utils';
import {
  TableHead as CarbonTableHead,
  TableHeader as CarbonTableHeader,
  TableRow as CarbonTableRow,
} from '@carbon/react';
import { memo } from 'react';

interface TableHeadProps {
  columns: { name: string; key: string }[];
}

export function TableHead(props: TableHeadProps & { isActionable?: boolean }) {
  return (
    <CarbonTableHead>
      <CarbonTableRow className="bg-light-background bg-surface border-b">
        {props.columns.map((cl, idx) => {
          return (
            <CarbonTableHeader
              key={idx}
              className={cn(
                'whitespace-no-wrap p-0 m-0 md:px-5 px-2 py-2 text-left uppercase text-xs/6 font-medium tracking-wider relative text-muted',
              )}
            >
              {cl.name}
            </CarbonTableHeader>
          );
        })}
        {props.isActionable && (
          <CarbonTableHeader className="whitespace-no-wrap px-2 md:px-5 py-3 w-20">
            <span className="absolute w-px h-px p-0 -m-px overflow-hidden whitespace-no-wrap border-0">
              Menu
            </span>
          </CarbonTableHeader>
        )}
      </CarbonTableRow>
    </CarbonTableHead>
  );
}

function TableHederWithCheckboxComponent(
  props: TableHeadProps & { ontoggle: (checked: boolean) => void },
) {
  return (
    <CarbonTableHead className="bg-surface/30 border-b border-border-subtle">
      <CarbonTableRow>
        <CarbonTableHeader
          className={cn(
            'whitespace-no-wrap px-2 md:px-5 py-3 w-20',
            'font-semibold text-sm text-left capitalize',
          )}
        >
          <div className="flex justify-between">
            <InputCheckbox
              aria-label="Select all rows"
              onChange={event => props.ontoggle(event.currentTarget.checked)}
            />
            <span className="w-0.5 bg-layer h-4"></span>
          </div>
        </CarbonTableHeader>
        {props.columns.map((cl, idx) => {
          return (
            <CarbonTableHeader
              key={idx}
              className={cn(
                'whitespace-no-wrap px-2 md:px-5 py-3',
                'font-semibold text-sm text-left capitalize',
              )}
            >
              <div className="flex justify-between">
                {cl.name}

                {idx !== props.columns.length - 1 && (
                  <span className="w-0.5 bg-layer h-4"></span>
                )}
              </div>
            </CarbonTableHeader>
          );
        })}
      </CarbonTableRow>
    </CarbonTableHead>
  );
}

export const TableHederWithCheckbox = memo(TableHederWithCheckboxComponent);
