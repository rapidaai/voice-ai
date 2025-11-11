import React from 'react';
import { cn } from '@/styles/media';
import { CustomLink } from '@/app/components/custom-link';

/**
 *
 */
interface LinkButtonProps
  extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  to: string;
  text: string;
  iconsize: string;
  gapclass: string;
}

/**
 *
 * @param props
 * @returns
 */
export function AnimatedLinkButton(props: LinkButtonProps) {
  return (
    <CustomLink
      isExternal={props.target === '_blank'}
      className={cn(
        props.className,
        'text-gray-700 dark:text-gray-200 font-medium block text-base',
        'group transition duration-300 w-fit',
      )}
      to={props.to}
    >
      {props.text}
      <svg
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
        strokeWidth={1.5}
        stroke="currentColor"
        className={cn(props.iconsize, 'inline-block ml-2')}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M8.25 4.5l7.5 7.5-7.5 7.5"
        />
      </svg>
      <span
        className={cn(
          props.gapclass,
          'block max-w-0 group-hover:max-w-full transition-all duration-500 h-px bg-gray-700 dark:bg-gray-50',
        )}
      ></span>
    </CustomLink>
  );
}

AnimatedLinkButton.defaultProps = {
  className: 'text-[18px] md:text-[20px] my-10',
  target: '_blank',
  rel: 'noreferrer',
  iconsize: 'w-6 h-6',
  gapclass: 'mt-2',
};
