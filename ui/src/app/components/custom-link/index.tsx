import React, { HTMLAttributes } from 'react';
import { Link } from 'react-router-dom';
import { cn } from '@/styles/media';
/**
 *
 * @param props
 * @returns
 */

export interface CustomLinkProps extends HTMLAttributes<HTMLAnchorElement> {
  isExternal?: boolean;
  to?: string;
}
export function CustomLink(props: CustomLinkProps) {
  if (props.isExternal)
    return (
      <a
        href={props.to}
        className={cn('focus:outline-hidden', props.className)}
        target="_blank"
        rel="noreferrer"
      >
        {props.children}
      </a>
    );
  return (
    <Link
      key={props.to}
      to={props.to ? props.to : ''}
      className={cn('focus:outline-hidden', props.className)}
    >
      {props.children}
    </Link>
  );
}
