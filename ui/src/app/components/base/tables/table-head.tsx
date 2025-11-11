import { cn } from '@/styles/media';
import { FC, memo } from 'react';

interface TableHeadProps {
  columns: { name: string; key: string }[];
}

export const TableHead: FC<
  TableHeadProps & { isActionable?: boolean }
> = props => {
  return (
    <thead className="">
      <tr className="bg-light-background dark:bg-gray-950 border-b">
        {props.columns.map((cl, idx) => {
          return (
            <th
              key={idx}
              className={cn(
                'whitespace-no-wrap px-2 md:px-5 py-3 font-medium',
                'text-sm text-left capitalize',
              )}
            >
              {cl.name}
            </th>
          );
        })}
        {props.isActionable && (
          <th className="whitespace-no-wrap px-2 md:px-5 py-3 w-20">
            <span className="absolute w-px h-px p-0 -m-px overflow-hidden whitespace-no-wrap border-0">
              Menu
            </span>
          </th>
        )}
      </tr>
    </thead>
  );
};

export const TableHederWithCheckbox: FC<
  TableHeadProps & { ontoggle: (boolean) => void }
> = memo(props => {
  return (
    <thead className="dark:bg-gray-950/30 bg-gray-100/50 border-b dark:border-gray-800">
      <tr className="">
        <th
          className={cn(
            'whitespace-no-wrap px-2 md:px-5 py-3 w-20',
            'font-semibold text-sm text-left capitalize',
          )}
        >
          <div className="flex justify-between">
            <input
              type="checkbox"
              onChange={x => props.ontoggle(x.target.checked)}
            />
            <span className="w-0.5 bg-slate-300 dark:bg-gray-800 h-4"></span>
          </div>
        </th>
        {props.columns.map((cl, idx) => {
          return (
            <th
              key={idx}
              className={cn(
                'whitespace-no-wrap px-2 md:px-5 py-3',
                'font-semibold text-sm text-left capitalize',
              )}
            >
              <div className="flex justify-between">
                {cl.name}

                {idx !== props.columns.length - 1 && (
                  <span className="w-0.5 bg-slate-300 dark:bg-gray-800 h-4"></span>
                )}
              </div>
            </th>
          );
        })}
      </tr>
    </thead>
  );
});
