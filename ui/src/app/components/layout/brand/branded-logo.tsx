import { forwardRef } from 'react';
import type { HTMLAttributes, Ref } from 'react';
import { useTheme } from '@/theme/theme-provider';
import { ResolvedThemeMode } from '@/theme/types';
import { typographyClassNames } from '@/app/components/ui/primitives/text';
import { cn } from '@/utils';

type BrandedLogoVariant = 'full' | 'compact';

interface BrandedLogoProps
  extends Omit<HTMLAttributes<HTMLElement>, 'children'> {
  variant?: BrandedLogoVariant;
  colorMode?: ResolvedThemeMode;
  textClassName?: string;
}

export const BrandedLogo = forwardRef<HTMLElement, BrandedLogoProps>(
  function BrandedLogo(
    { variant = 'full', colorMode, className, textClassName, ...attributes },
    ref,
  ) {
    const { resolvedMode, theme } = useTheme();
    const logo = theme.brand.logos?.[variant][colorMode ?? resolvedMode];
    const environmentLabel =
      variant === 'full' ? theme.brand.environmentLabel : undefined;

    const label = environmentLabel ? (
      <span
        aria-label={`Environment: ${environmentLabel}`}
        className={cn(
          typographyClassNames['label-01'],
          'ml-2 shrink-0 whitespace-nowrap rounded-[3px] border border-[var(--cds-support-warning)] px-1 py-0.5 text-[var(--cds-support-warning)]',
        )}
      >
        {environmentLabel}
      </span>
    ) : null;

    if (logo) {
      return (
        <>
          <img
            {...attributes}
            ref={ref as Ref<HTMLImageElement>}
            src={logo}
            alt={theme.brand.name}
            className={cn('block object-contain', className)}
          />
          {label}
        </>
      );
    }

    return (
      <>
        <span
          {...attributes}
          ref={ref as Ref<HTMLSpanElement>}
          className={cn(
            typographyClassNames['heading-compact-02'],
            'block min-w-0 truncate text-primary',
            className,
            textClassName,
          )}
        >
          {theme.brand.name}
        </span>
        {label}
      </>
    );
  },
);
