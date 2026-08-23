import React, { FC, HTMLAttributes, memo } from 'react';
import { Header as CarbonHeader } from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { RapidaTextIcon } from '@/app/components/Icon/RapidaText';
import { cn } from '@/utils';

interface HeaderProps extends HTMLAttributes<HTMLElement> {}

export const Header: FC<HeaderProps> = memo(
  ({ className, 'aria-label': ariaLabel, ...attributes }) => {
    const { resolvedMode, theme } = useTheme();
    const fullLogo = theme.brand.logos?.full[resolvedMode];

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
          {fullLogo ? (
            <img
              src={fullLogo}
              alt={theme.brand.name}
              className="block h-6 w-auto max-w-[12rem] object-contain object-left"
            />
          ) : (
            <div className="flex items-center gap-2 text-primary">
              <RapidaIcon className="h-6 w-6 shrink-0" />
              <RapidaTextIcon className="h-[18px] w-auto" />
            </div>
          )}
        </div>
      </CarbonHeader>
    );
  },
);

Header.displayName = 'Header';
