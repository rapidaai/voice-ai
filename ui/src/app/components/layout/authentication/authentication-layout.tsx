import { Outlet } from 'react-router-dom';
import { Header as CarbonHeader, HeaderName } from '@carbon/react';
import { BrandedLogo } from '@/app/components/layout/brand/branded-logo';
import { GeneralFooter } from '@/app/components/layout/footer/general-footer';
import { useTheme } from '@/theme/theme-provider';

function AuthenticationHeader() {
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

export function AuthenticationLayout() {
  return (
    <div className="flex min-h-dvh flex-col bg-layer text-foreground antialiased">
      <AuthenticationHeader />

      <main
        aria-label="Authentication"
        className="flex min-h-0 flex-1 items-center justify-center overflow-auto bg-layer p-4 sm:p-10"
        data-testid="auth-main"
      >
        <section
          aria-label="Account access"
          className="mx-auto flex w-full max-w-[37rem] justify-center"
        >
          <Outlet />
        </section>
      </main>

      <GeneralFooter />
    </div>
  );
}
