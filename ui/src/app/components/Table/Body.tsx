import React, { HTMLAttributes, TdHTMLAttributes } from 'react';
import ReactMarkdown from 'react-markdown';
import { cn } from '@/styles/media';

/**
 *
 * @param props
 * @returns
 */
export function TableLink(props: { text: string; to: string }) {
  return (
    <div className="font-normal flex items-center dark:text-blue-500 text-blue-600 hover:underline cursor-pointer text-left">
      {props.text}
      {props.text !== '' && (
        <span className="ml-0.5">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            stroke="currentColor"
            className="w-3 h-3"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="m4.5 19.5 15-15m0 0H8.25m11.25 0v11.25"
            />
          </svg>
        </span>
      )}
    </div>
  );
}

/**
 *
 * @param props
 * @returns
 */
export function TableMultilineText(props: { children: string }) {
  return (
    <ReactMarkdown className="prose mt-1 break-words prose-p:leading-relaxed">
      {props.children}
    </ReactMarkdown>
  );
  // {props.children}    <div
  //       className={cn('font-normal text-left max-w-[20rem]', props.className)}
  //       {...props}
  //     >

  //     </div> */}
  //   );
}

/**
 *
 * @param props
 * @returns
 */
export function TableStatus(props: { status: string; processing: boolean }) {
  return (
    <div className="font-normal flex space-x-2 items-center text-green-500">
      {props.processing ? (
        <svg
          aria-hidden="true"
          className="w-4 h-4 animate-spin fill-gray-600"
          viewBox="0 0 100 101"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            d="M100 50.5908C100 78.2051 77.6142 100.591 50 100.591C22.3858 100.591 0 78.2051 0 50.5908C0 22.9766 22.3858 0.59082 50 0.59082C77.6142 0.59082 100 22.9766 100 50.5908ZM9.08144 50.5908C9.08144 73.1895 27.4013 91.5094 50 91.5094C72.5987 91.5094 90.9186 73.1895 90.9186 50.5908C90.9186 27.9921 72.5987 9.67226 50 9.67226C27.4013 9.67226 9.08144 27.9921 9.08144 50.5908Z"
            fill="currentColor"
          />
          <path
            d="M93.9676 39.0409C96.393 38.4038 97.8624 35.9116 97.0079 33.5539C95.2932 28.8227 92.871 24.3692 89.8167 20.348C85.8452 15.1192 80.8826 10.7238 75.2124 7.41289C69.5422 4.10194 63.2754 1.94025 56.7698 1.05124C51.7666 0.367541 46.6976 0.446843 41.7345 1.27873C39.2613 1.69328 37.813 4.19778 38.4501 6.62326C39.0873 9.04874 41.5694 10.4717 44.0505 10.1071C47.8511 9.54855 51.7191 9.52689 55.5402 10.0491C60.8642 10.7766 65.9928 12.5457 70.6331 15.2552C75.2735 17.9648 79.3347 21.5619 82.5849 25.841C84.9175 28.9121 86.7997 32.2913 88.1811 35.8758C89.083 38.2158 91.5421 39.6781 93.9676 39.0409Z"
            fill="currentFill"
          />
        </svg>
      ) : (
        <span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            stroke="currentColor"
            className="w-4 h-4"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z"
            />
          </svg>
        </span>
      )}
      <span className="capitalize font-medium">{props.status}</span>
    </div>
  );
}

/**
 *
 * @param props
 * @returns
 */
export function TableFile(props: { path: string }) {
  return (
    <div className="font-normal dark:text-blue-500 truncate text-blue-600 hover:underline cursor-pointer text-left max-w-[20rem]">
      {props.path}
    </div>
  );
}

/**
 *
 * @param props
 * @returns
 */
export function TableText(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className="font-normal text-left max-w-[20rem] truncate">
      {props.children}
    </div>
  );
}

export function TableCode(props: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className="flex justify-between w-fit space-x-2 items-center rounded-[2px] bg-white/50 dark:bg-gray-950/50 backdrop-blur-sm px-2 py-1 cursor-pointer"
      onClick={() => {
        if (props.children)
          navigator.clipboard.writeText(props.children.toString());
      }}
    >
      <div className="font-normal text-left max-w-[20rem] truncate font-mono text-xs">
        {props.children}
      </div>
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        className="w-3 h-3"
      >
        <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
        <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
      </svg>
    </div>
  );
}

/**
 *
 * @param props
 * @returns
 */
export function TableState(props: { text: string; color: string }) {
  return (
    <div
      className={cn(
        'text-left w-fit text-xs rounded-[2px] font-semibold max-w-[20rem] truncate capitalize px-2 py-1 border',
        props.color,
      )}
    >
      {props.text}
    </div>
  );
}

export function TableStatusBlock(props: { status: string }) {
  if (props.status === 'draft')
    return (
      <div className="inline-flex rounded-[2px] capitalize dark:bg-gray-700 bg-gray-400 text-white text-center font-medium px-2 text-xs py-1">
        {props.status.toLowerCase()}
      </div>
    );

  if (
    props.status.toLowerCase() === 'error' ||
    props.status.toLowerCase() === 'failure'
  )
    return (
      <div className="inline-flex rounded-[2px] capitalize dark:bg-red-700 bg-red-400 text-white text-center font-medium px-2 text-xs py-1">
        {props.status.toLowerCase()}
      </div>
    );

  return (
    <div className="inline-flex rounded-[2px] capitalize dark:bg-emerald-700 bg-emerald-500 text-white text-center font-medium px-2 text-xs py-1">
      {props.status.toLowerCase()}
    </div>
  );
}

export function TD(props: React.TdHTMLAttributes<HTMLTableCellElement>) {
  return (
    <td className={cn('whitespace-no-wrap px-2 md:px-5 py-3', props.className)}>
      {props.children}
    </td>
  );
}

export function TR(props: React.HTMLAttributes<HTMLTableRowElement>) {
  return (
    <tr
      {...props}
      className={cn(
        'dark:border-gray-800 border-gray-300 border-b-[0.5px] bg-gray-50/50 dark:bg-gray-900/10 hover:bg-gray-100',
        props.className,
      )}
    >
      {props.children}
    </tr>
  );
}

export function ClickableTR(props: React.HTMLAttributes<HTMLTableRowElement>) {
  return (
    <TR
      className={cn(
        'hover:bg-gray-200/70 dark:hover:bg-gray-700/70',
        props.className,
      )}
    >
      {props.children}
    </TR>
  );
}

export function TBody(props: React.HTMLAttributes<HTMLElement>) {
  return (
    <tbody className="text-[15px]" {...props}>
      {props.children}
    </tbody>
  );
}
