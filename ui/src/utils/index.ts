import { generate } from 'random-words';
import { ResourceRole } from '@/models/common';
import { RETRIEVE_METHOD } from '@/models/datasets';
import {
  DEBUGGER_SOURCE,
  RAPIDA_APP_SOURCE,
  RapidaSource,
} from '@/utils/rapida_source';

import clsx from 'clsx';
import { getBrowser } from '@/utils/browser';
import { RapidaEnvironment } from '@/utils/rapida_environment';

/**
 *
 * @param ms
 * @returns
 */
export const sleep = (ms: number) => {
  return new Promise(resolve => setTimeout(resolve, ms));
};

/**
 *
 * @param fn
 * @returns
 */
export async function asyncRunSafe<T = any>(
  fn: Promise<T>,
): Promise<[Error] | [null, T]> {
  try {
    return [null, await fn];
  } catch (e) {
    if (e instanceof Error) return [e];
    return [new Error('unknown error')];
  }
}

/**
 *
 * @param text
 * @param font
 * @returns
 */
export const getTextWidthWithCanvas = (text: string, font?: string) => {
  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  if (ctx) {
    ctx.font =
      font ??
      '12px Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji"';
    return Number(ctx.measureText(text).width.toFixed(2));
  }
  return 0;
};

const chars =
  '0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-_';

/**
 *
 * @param length
 * @returns
 */
export function randomString(length: number) {
  let result = '';
  for (let i = length; i > 0; --i)
    result += chars[Math.floor(Math.random() * chars.length)];
  return result;
}

export const getPurifyHref = (href: string) => {
  if (!href) return '';

  return href
    .replace(/javascript:/gi, '')
    .replace(/vbscript:/gi, '')
    .replace(/data:/gi, '');
};

export const toDateString = (d: Date) => {
  const postgresDateString = d.toLocaleDateString('en-CA');
  return postgresDateString;
};

/**
 *
 * @param rl
 * @returns
 */
export const isOrgResource = (rl: ResourceRole): boolean => {
  if (
    rl === ResourceRole.owner ||
    rl === ResourceRole.projectMember ||
    rl === ResourceRole.organizationMember
  )
    return true;
  return false;
};

/**
 *
 * @param rl
 * @returns
 */
export const isProjectResource = (rl: ResourceRole): boolean => {
  if (rl === ResourceRole.owner || rl === ResourceRole.projectMember)
    return true;
  return false;
};

/**
 *
 * @param rl
 * @returns
 */
export const isOwnerResource = (rl: ResourceRole): boolean => {
  if (rl === ResourceRole.owner) return true;
  return false;
};

/**
 *
 * @param method
 * @returns
 */
export function retrieveMethodFromString(method: string): RETRIEVE_METHOD {
  switch (method) {
    case RETRIEVE_METHOD.semantic:
      return RETRIEVE_METHOD.semantic;
    case RETRIEVE_METHOD.fullText:
      return RETRIEVE_METHOD.fullText;
    case RETRIEVE_METHOD.hybrid:
      return RETRIEVE_METHOD.hybrid;
    case RETRIEVE_METHOD.invertedIndex:
      return RETRIEVE_METHOD.invertedIndex;
    default:
      return RETRIEVE_METHOD.semantic; // Or throw an error if appropriate
  }
}

/**
 *
 * @returns
 */
export const isElectron = (): boolean => {
  return window.isElectron;
};

/**
 *
 * @param str
 * @returns
 */
export const toTitleCase = (str: any) => {
  return str
    .toLowerCase()
    .split(' ')
    .map((word: any) => {
      return word.charAt(0).toUpperCase() + word.slice(1);
    })
    .join(' ');
};

/**
 *
 * @param string
 * @returns
 */
export function ValidHttpUrl(string) {
  let url;
  try {
    url = new URL(string);
  } catch (_) {
    return false;
  }
  return url.protocol === 'http:' || url.protocol === 'https:';
}

/**
 * Calls all functions in the order they were chained with the same arguments.
 * @internal
 */
export function chain(...callbacks: any[]): (...args: any[]) => void {
  return (...args: any[]) => {
    for (const callback of callbacks) {
      if (typeof callback === 'function') {
        try {
          callback(...args);
        } catch (e) {
          console.error(e);
        }
      }
    }
  };
}

interface Props {
  [key: string]: any;
}

// taken from: https://stackoverflow.com/questions/51603250/typescript-3-parameter-list-intersection-type/51604379#51604379
type TupleTypes<T> = { [P in keyof T]: T[P] } extends { [key: number]: infer V }
  ? V
  : never;
type UnionToIntersection<U> = (U extends any ? (k: U) => void : never) extends (
  k: infer I,
) => void
  ? I
  : never;

/**
 * Merges multiple props objects together. Event handlers are chained,
 * classNames are combined, and ids are deduplicated - different ids
 * will trigger a side-effect and re-render components hooked up with `useId`.
 * For all other props, the last prop object overrides all previous ones.
 * @param args - Multiple sets of props to merge together.
 * @internal
 */
export function mergeProps<T extends Props[]>(
  ...args: T
): UnionToIntersection<TupleTypes<T>> {
  // Start with a base clone of the first argument. This is a lot faster than starting
  // with an empty object and adding properties as we go.
  const result: Props = { ...args[0] };
  for (let i = 1; i < args.length; i++) {
    const props = args[i];
    for (const key in props) {
      const a = result[key];
      const b = props[key];

      // Chain events
      if (
        typeof a === 'function' &&
        typeof b === 'function' &&
        // This is a lot faster than a regex.
        key[0] === 'o' &&
        key[1] === 'n' &&
        key.charCodeAt(2) >= /* 'A' */ 65 &&
        key.charCodeAt(2) <= /* 'Z' */ 90
      ) {
        result[key] = chain(a, b);

        // Merge classnames, sometimes classNames are empty string which eval to false, so we just need to do a type check
      } else if (
        (key === 'className' || key === 'UNSAFE_className') &&
        typeof a === 'string' &&
        typeof b === 'string'
      ) {
        result[key] = clsx(a, b);
      } else {
        result[key] = b !== undefined ? b : a;
      }
    }
  }

  return result as UnionToIntersection<TupleTypes<T>>;
}

export function isSafari(): boolean {
  return getBrowser()?.name === 'Safari';
}

export const GetEnvironment = (): RapidaEnvironment => {
  return process.env.NODE_ENV !== 'development'
    ? RapidaEnvironment.PRODUCTION
    : RapidaEnvironment.DEVELOPMENT;
};

/**
 *
 * @returns
 */
export const GetSource = (): RapidaSource => {
  if (isElectron()) return RAPIDA_APP_SOURCE;
  return DEBUGGER_SOURCE;
};

/**
 *
 * @param prefix
 * @returns
 */
export const randomMeaningfullName = (prefix?: string): string => {
  const name = generate({ exactly: 3, join: '-' });
  return prefix ? `${prefix}-${name}` : name;
};

/**
 * Converts nanoseconds to milliseconds.
 * @param nano - The number of nanoseconds or undefined.
 * @returns The equivalent number of milliseconds, or undefined if input is undefined.
 */
export function nanoToMilli(
  nano: number | string | undefined,
): number | undefined {
  if (nano === undefined) return undefined;
  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  return Number((nanoNumber / 1_000_000).toFixed(2));
}

export function nanoToMinute(
  nano: number | string | undefined,
): number | undefined {
  if (nano === undefined) return undefined;
  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  return Number((nanoNumber / 60_000_000_000).toFixed(2));
}

export function formatNanoToReadableMinute(
  nano: number | string | undefined,
): string {
  if (nano === undefined || isNaN(Number(nano))) {
    return 'n/a';
  }

  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  const totalSeconds = Math.floor(nanoNumber / 1_000_000_000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (totalSeconds <= 0) {
    return 'n/a';
  }

  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  } else {
    return `${seconds}s`;
  }
}
export function formatNanoToReadableMilli(
  nano: number | string | undefined,
  fraction = 2,
): string {
  if (nano === undefined || isNaN(Number(nano))) {
    return 'n/a';
  }

  const nanoNumber = typeof nano === 'string' ? parseFloat(nano) : nano;
  const totalMilliSeconds = nanoNumber / 1_000_000; // No Math.floor, keep fractions

  if (totalMilliSeconds <= 0) {
    return 'n/a';
  }

  return `${totalMilliSeconds.toFixed(fraction)} ms`; // ToFixed ensures readability for fractions
}
