import React, { FC, HTMLAttributes, memo } from 'react';
import { Header as CarbonHeader } from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';
import { BrandedLogo } from '@/app/components/brand/branded-logo';
import { cn } from '@/utils';

interface HeaderProps extends HTMLAttributes<HTMLElement> {}

export const Header: FC<HeaderProps> = memo(
  ({ className, 'aria-label': ariaLabel, ...attributes }) => {
    const { theme } = useTheme();

    return (
      <CarbonHeader
        {...attributes}
        aria-label={ariaLabel ?? `${theme.brand.name} Platform`}
        className={cn(
          'bg-shell! border-b! border-border-subtle! px-3!',
          className,
        )}
      >
        <div className="flex h-full min-w-0 items-center">
          <BrandedLogo
            variant="full"
            className="h-6 w-auto max-w-[12rem] object-left"
            textClassName="text-base"
          />
        </div>
      </CarbonHeader>
    );
  },
);

Header.displayName = 'Header';
