import { Outlet } from 'react-router-dom';
import { GeneralFooter } from '@/app/components/layout/footer/general-footer';
import { PublicShellHeader } from '@/app/components/layout/public-shell/public-shell-header';

export function AuthenticationLayout() {
  return (
    <div className="flex min-h-dvh flex-col bg-layer text-foreground antialiased">
      <PublicShellHeader />

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
