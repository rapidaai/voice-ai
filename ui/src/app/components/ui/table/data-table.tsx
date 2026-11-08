import { TableRow } from './table-row';
import { InputCheckbox } from '@/app/components/ui/primitives/input-checkbox';
import {
  Table as CarbonTable,
  TableBody as CarbonTableBody,
  TableHead as CarbonTableHead,
  TableHeader as CarbonTableHeader,
} from '@carbon/react';
import { useState, useCallback, useRef, useEffect } from 'react';
import type { HTMLAttributes, MouseEvent as ReactMouseEvent } from 'react';

interface ScrollableResizableTableProps
  extends HTMLAttributes<HTMLTableElement> {
  clms: { name: string; key: string; width?: number }[];
  isActionable?: boolean;
  isExpandable?: boolean;
  ontoggle?: (checked: boolean) => void;
  isOptionable?: boolean;
  optionLabel?: string;
}

export function ScrollableResizableTable({
  clms,
  isActionable = true,
  isExpandable = false,
  ontoggle,
  children,
  isOptionable,
  optionLabel,
}: ScrollableResizableTableProps) {
  const [columns, setColumns] = useState<
    { key: string; width: number; name: string }[]
  >([]);

  useEffect(() => {
    setColumns(
      clms.map(x => ({
        name: x.name,
        key: x.key,
        width: x.width ?? 200,
      })),
    );
  }, [clms]);
  const tableRef = useRef<HTMLDivElement | null>(null);
  const [tableWidth, setTableWidth] = useState(0);

  useEffect(() => {
    const updateTableWidth = () => {
      if (tableRef.current) {
        const newWidth = columns.reduce((sum, column) => sum + column.width, 0);
        setTableWidth(newWidth);
      }
    };
    updateTableWidth();
  }, [columns]);

  const handleResize = useCallback((index: number, newWidth: number) => {
    setColumns(previousColumns =>
      previousColumns.map((column, columnIndex) =>
        columnIndex === index
          ? { ...column, width: Math.max(100, newWidth) }
          : column,
      ),
    );
  }, []);

  return (
    <div className="w-full flex flex-col flex-1 overflow-x-auto">
      <div
        ref={tableRef}
        style={{
          minWidth: `${tableWidth}px`,
          //   minWidth: '100%',
          maxWidth: '100% !important',
        }}
      >
        <CarbonTable className="w-full border-collapse bg-surface border-y ">
          <CarbonTableHead className="uppercase">
            <TableRow className="bg-layer ">
              {isActionable && (
                <CarbonTableHeader className="w-8 h-8 ">
                  <div className="w-8 h-8 flex justify-center items-center">
                    <InputCheckbox
                      aria-label="Select all rows"
                      onChange={event =>
                        ontoggle?.(event.currentTarget.checked)
                      }
                    />
                  </div>
                </CarbonTableHeader>
              )}

              {isExpandable && (
                <CarbonTableHeader className="w-8 h-8 ">
                  <div className="w-8 h-8 flex justify-center items-center"></div>
                </CarbonTableHeader>
              )}

              {columns.map((column, index) => (
                <CarbonTableHeader
                  key={column.key}
                  className="px-2 py-2 text-left uppercase text-xs/6 font-medium tracking-wider relative text-muted"
                  style={{ width: column.width }}
                >
                  {column.name}
                  {index !== columns.length - 1 && (
                    <div
                      className="absolute top-1 -right-[2px] bottom-1 w-[2px] cursor-col-resize hover:bg-primary"
                      onMouseDown={(e: ReactMouseEvent<HTMLDivElement>) => {
                        e.preventDefault();
                        const startX = e.pageX;
                        const startWidth = column.width;

                        const onMouseMove = (event: MouseEvent) => {
                          const newWidth = startWidth + event.pageX - startX;
                          handleResize(index, newWidth);
                        };

                        const onMouseUp = () => {
                          document.removeEventListener(
                            'mousemove',
                            onMouseMove,
                          );
                          document.removeEventListener('mouseup', onMouseUp);
                        };

                        document.addEventListener('mousemove', onMouseMove);
                        document.addEventListener('mouseup', onMouseUp);
                      }}
                    ></div>
                  )}
                </CarbonTableHeader>
              ))}

              {isOptionable && (
                <CarbonTableHeader className="w-10 px-2 py-2 text-left uppercase text-xs/6 font-medium tracking-wider text-muted">
                  {optionLabel}
                </CarbonTableHeader>
              )}
            </TableRow>
          </CarbonTableHead>
          <CarbonTableBody>{children}</CarbonTableBody>
        </CarbonTable>
      </div>
    </div>
  );
}
