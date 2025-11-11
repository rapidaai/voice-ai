import { CustomLink } from '@/app/components/custom-link';
import { RightArrowIcon } from '@/app/components/Icon/RightArrow';
import React from 'react';

/**
 *
 * Document link
 * @returns
 */
export function DocumentLink() {
  return (
    <CustomLink
      isExternal={true}
      to="https://docs.rapida.ai"
      className="text-blue-600 dark:text-blue-400 text-sm flex items-center"
    >
      Read the support documentation
      <RightArrowIcon />
    </CustomLink>
  );
}
