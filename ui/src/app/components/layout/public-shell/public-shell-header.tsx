import { Header as CarbonHeader, HeaderName } from '@carbon/react';
import { BrandedLogo } from '@/app/components/layout/brand/branded-logo';
import { useTheme } from '@/theme/theme-provider';

export function PublicShellHeader() {
  const { theme } = useTheme();

  return (
    <CarbonHeader
      aria-label={`${theme.brand.name} Platform`}
      className="relative! inset-auto! z-auto! h-12! min-h-12! shrink-0 border-b! border-border-subtle! bg-surface! p-0!"
    >
      <HeaderName
        aria-label={theme.brand.name}
        className="h-full px-4! sm:px-8!"
        href="/"
        prefix=""
      >
        <span className="flex items-center whitespace-nowrap">
          <BrandedLogo
            variant="full"
            className="h-6 w-auto max-w-[12rem] object-left"
          />
        </span>
      </HeaderName>
    </CarbonHeader>
  );
}
