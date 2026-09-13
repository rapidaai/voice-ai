import { ImgHTMLAttributes } from 'react';
import { useTheme } from '@/theme/theme-provider';
import { ResolvedThemeMode } from '@/theme/types';
import { cn } from '@/utils';

type BrandedLogoVariant = 'full' | 'compact';

interface BrandedLogoProps
  extends Omit<ImgHTMLAttributes<HTMLImageElement>, 'src' | 'alt'> {
  variant?: BrandedLogoVariant;
  colorMode?: ResolvedThemeMode;
  textClassName?: string;
}

export function BrandedLogo({
  variant = 'full',
  colorMode,
  className,
  textClassName,
  ...attributes
}: BrandedLogoProps) {
  const { resolvedMode, theme } = useTheme();
  const logo = theme.brand.logos?.[variant][colorMode ?? resolvedMode];
  const environmentLabel =
    variant === 'full' ? theme.brand.environmentLabel : undefined;

  const label = environmentLabel ? (
    <span
      aria-label={`Environment: ${environmentLabel}`}
      className="ml-2 rounded-[3px] border border-[var(--cds-support-warning)] px-1 py-0.5 text-[10px] font-semibold leading-none text-[var(--cds-support-warning)]"
    >
      {environmentLabel}
    </span>
  ) : null;

  if (logo) {
    return (
      <>
        <img
          {...attributes}
          src={logo}
          alt={theme.brand.name}
          className={cn('block object-contain', className)}
        />
        {label}
      </>
    );
  }

  return (
    <span className="flex min-w-0 items-center">
      <span
        className={cn(
          'block min-w-0 truncate font-semibold text-primary',
          className,
          textClassName,
        )}
      >
        {theme.brand.name}
      </span>
      {label}
    </span>
  );
}
