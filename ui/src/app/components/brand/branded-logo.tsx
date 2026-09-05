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

  if (logo) {
    return (
      <img
        {...attributes}
        src={logo}
        alt={theme.brand.name}
        className={cn('block object-contain', className)}
      />
    );
  }

  return (
    <span
      className={cn(
        'block min-w-0 truncate font-semibold text-primary',
        className,
        textClassName,
      )}
    >
      {theme.brand.name}
    </span>
  );
}
