import React, { useState } from 'react';
import { cn } from '@/styles/media';
import { IButton } from '@/app/components/Form/Button';
import { ColumnPreferencesDialog } from '@/app/components/base/modal/column-preference-modal';
import { Settings, SlidersHorizontal } from 'lucide-react';
import TooltipPlus from '@/app/components/base/tooltip-plus';

interface TablePreferenceProps extends React.InputHTMLAttributes<HTMLElement> {
  /**
   * default page size
   */
  defaultPageSize: number[];
  /**
   *
   * The columns which is shown currently
   */
  columns: { name: string; key: string; visible: boolean }[];

  /**
   *
   * @param clmns
   * @returns
   */
  onChangeColumns: (
    clmns: { name: string; key: string; visible: boolean }[],
  ) => void;

  /**
   * Item per page
   */
  pageSize: number;

  /**
   * onChange of page
   */
  onChangePageSize: (number) => void;
}

interface TablePaginationProps extends TablePreferenceProps {
  /**
   * Current Page
   */
  currentPage: number;

  /**
   * change current page
   */
  onChangeCurrentPage: (string) => void;

  /**
   * total Page size
   */
  totalItem: number;
}

/**
 *
 * @param props
 * @returns
 */

export function TablePagination(props: TablePaginationProps) {
  /**
   * Column Perference
   */
  const [columnPreferenceModel, setColumnPreferenceModel] = useState(false);

  //   page start from 0
  const maxPage = Math.ceil(props.totalItem / props.pageSize);
  let arr = generatePageArray(props);
  return (
    <>
      <ColumnPreferencesDialog
        open={columnPreferenceModel}
        setOpen={setColumnPreferenceModel}
        {...props}
      ></ColumnPreferencesDialog>
      <ul className="flex items-center text-base">
        <li>
          <IButton
            type="button"
            onClick={() => {
              props.currentPage > 1 &&
                props.onChangeCurrentPage(props.currentPage - 1);
            }}
            disabled={props.currentPage <= 1}
            className={cn(
              'text-base! bg-transparent!',
              props.currentPage <= 1 ? 'cursor-not-allowed' : 'cursor-pointer',
            )}
          >
            <span className="sr-only">Previous</span>
            <svg
              className="w-4 h-4 rtl:rotate-180"
              aria-hidden="true"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 6 10"
            >
              <path
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M5 1 1 5l4 4"
              />
            </svg>
          </IButton>
        </li>
        {/* page count start */}
        {arr
          .filter(px => {
            return px !== undefined;
          })
          .map((pg, idx) => {
            return (
              <li key={`page-${idx}`}>
                <IButton
                  type="button"
                  className={cn(
                    'text-base! bg-transparent!',
                    pg === props.currentPage
                      ? 'text-blue-600! opacity-100'
                      : 'opacity-70',
                  )}
                  onClick={() => {
                    props.onChangeCurrentPage(pg);
                  }}
                >
                  {pg}
                </IButton>
              </li>
            );
          })}

        <li className="border-r dark:border-gray-800">
          <IButton
            type="button"
            disabled={props.currentPage >= maxPage}
            onClick={() => {
              props.currentPage < maxPage &&
                props.onChangeCurrentPage(props.currentPage + 1);
            }}
            className={cn(
              'text-base! bg-transparent!',
              props.currentPage >= maxPage
                ? 'cursor-not-allowed'
                : 'cursor-pointer',
            )}
          >
            <span className="sr-only">Next</span>
            <svg
              className="w-4 h-4 rtl:rotate-180"
              aria-hidden="true"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 6 10"
            >
              <path
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                d="m1 9 4-4-4-4"
              />
            </svg>
          </IButton>
        </li>

        {/* setting preference */}
        <li>
          <IButton
            type="button"
            onClick={() => {
              setColumnPreferenceModel(true);
            }}
          >
            <span className="sr-only">Setting</span>
            <TooltipPlus
              className="bg-white dark:bg-gray-950 border-[0.5px] rounded-[2px] px-0 py-0"
              popupContent={
                <div className="px-3 py-2 text-sm text-gray-600 dark:text-gray-500">
                  Configure column preference
                </div>
              }
            >
              <SlidersHorizontal className="w-4 h-4" strokeWidth={1.5} />
            </TooltipPlus>
          </IButton>
        </li>
      </ul>
    </>
  );
}

function generatePageArray(props: TablePaginationProps) {
  const maxPage = Math.ceil(props.totalItem / props.pageSize);

  //   props.currentPage >= maxPage;
  const prevPage = props.currentPage > 1 ? props.currentPage - 1 : undefined;
  const nextPage =
    props.currentPage < maxPage ? props.currentPage + 1 : undefined;

  // Filter out undefined values
  return [prevPage, props.currentPage, nextPage];
}

// default params for table pagination
TablePagination.defaultProps = {
  defaultPageSize: [10, 20, 50],
  currentPage: 1,
  pageSize: 20,
  totalItem: 1,
  columns: [],
  onChangeCurrentPage: () => {},
  onChangePageSize: () => {},
  onChangeColumns: ([]) => {},
};
