import React from 'react';
export function TextImage(props: {
  size?: 4 | 5 | 6 | 7 | 8 | 9 | 10;
  name: string;
}) {
  const dimensionClz = (size?: number) => {
    if (size === 10) {
      return 'w-10 h-10';
    }
    if (size === 7) {
      return 'w-7 h-7 text-sm!';
    }

    if (size === 8) {
      return 'w-8 h-8 text-sm!';
    }

    if (size === 6) {
      return 'w-6 h-6 text-sm!';
    }
    if (size === 5) {
      return 'w-5 h-5 text-sm!';
    }
    if (size === 4) {
      return 'w-4 h-4 text-xs!';
    }

    return 'h-9 w-9';
  };

  const randomBgColor = () => {
    return [
      'bg-rose-800 dark:bg-rose-400 text-rose-100 dark:text-rose-900',
      'bg-red-800 dark:bg-red-400 text-red-100 dark:text-red-900',
      'bg-green-800 dark:bg-green-400 text-green-100 dark:text-green-900',
      'bg-sky-800 dark:bg-sky-400 text-sky-100 dark:text-sky-900',
      'bg-indigo-800 dark:bg-indigo-400 text-indigo-100 dark:text-indigo-900',
      'bg-orange-800 dark:bg-orange-400 text-orange-100 dark:text-orange-900',
      'bg-amber-800 dark:bg-amber-400 text-amber-100 dark:text-amber-900',
      'bg-yellow-800 dark:bg-yellow-400 text-yellow-100 dark:text-yellow-900',
      'bg-lime-800 dark:bg-lime-400 text-lime-100 dark:text-lime-900',
      'bg-teal-800 dark:bg-teal-400 text-teal-100 dark:text-teal-900',
      'bg-emerald-800 dark:bg-emerald-400 text-emerald-100 dark:text-emerald-900',
      'bg-cyan-800 dark:bg-cyan-400 text-cyan-100 dark:text-cyan-900',
      'bg-violet-800 dark:bg-violet-400 text-violet-100 dark:text-violet-900',
      'bg-purple-800 dark:bg-purple-400 text-purple-100 dark:text-purple-900',
      'bg-fuchsia-800 dark:bg-fuchsia-400 text-fuchsia-100 dark:text-fuchsia-900',
      'bg-pink-800 dark:bg-pink-400 text-pink-100 dark:text-pink-900',
    ].find((_, i, ar) => Math.random() < 1 / (ar.length - i));
  };

  return (
    <div
      className={`${dimensionClz(
        props.size,
      )} ${randomBgColor()} rounded-[2px] uppercase flex justify-center items-center text-center text-lg shrink-0`}
    >
      <span>{props.name.charAt(0)}</span>
    </div>
  );
}
