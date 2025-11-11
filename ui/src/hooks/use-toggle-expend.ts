import { useState } from 'react';

export const useToggleExpend = (ref: React.RefObject<HTMLDivElement>) => {
  const [isExpand, setIsExpand] = useState(false);
  const wrapClassName =
    isExpand &&
    'fixed z-50 left-0 right-0 top-0 bottom-0 rounded-lg backdrop-blur-xl px-10 py-10 h-full w-full';
  return {
    wrapClassName,
    isExpand,
    setIsExpand,
  };
};
