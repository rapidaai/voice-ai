import { Link, ProgressBar } from '@carbon/react';
import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom';
import { BrandedLogo } from '@/app/components/layout/brand/branded-logo';
import { ProtectedBox } from '@/app/components/layout/container/protected-box';
import {
  OnboardingCreateOrganizationPage,
  OnboardingCreateProjectPage,
} from '@/app/pages/user-onboarding';
import { useTheme } from '@/theme/theme-provider';

const STEPS = [
  { path: 'organization', label: 'Organization' },
  { path: 'project', label: 'Project' },
];

export function OnboardingLayout() {
  const location = useLocation();
  const { theme } = useTheme();
  const currentStepIndex = STEPS.findIndex(step =>
    location.pathname.includes(step.path),
  );
  const currentStep = currentStepIndex >= 0 ? currentStepIndex + 1 : 1;

  return (
    <div className="flex min-h-dvh flex-col bg-layer text-foreground">
      <header className="flex h-12 shrink-0 items-center border-b border-border-subtle bg-surface px-4 sm:px-8">
        <BrandedLogo
          variant="full"
          className="h-6 w-auto max-w-48 object-left"
          textClassName="text-base"
        />
      </header>

      <main
        aria-label="Onboarding"
        className="flex flex-1 justify-center px-4 py-8 sm:px-8 sm:py-12 md:px-12 lg:px-16 lg:py-16"
      >
        <section
          aria-label="Workspace setup"
          className="my-auto w-full max-w-[40rem] border border-border-subtle bg-surface px-4 py-8 sm:px-10 sm:py-10 lg:px-12 lg:py-12"
        >
          <div className="mx-auto w-full max-w-[30rem]">
            <div className="mb-10">
              <div className="mb-3 flex items-end justify-between gap-4">
                <div>
                  <p className="text-xs text-muted">Workspace setup</p>
                  <p className="mt-1 text-sm font-medium">
                    {STEPS[currentStep - 1].label}
                  </p>
                </div>
                <p className="shrink-0 text-sm text-muted">
                  Step {currentStep} of {STEPS.length}
                </p>
              </div>
              <ProgressBar
                hideLabel
                label="Onboarding progress"
                max={STEPS.length}
                size="small"
                value={currentStep}
              />
            </div>

            <Outlet />
          </div>
        </section>
      </main>

      <footer className="shrink-0 border-t border-border-subtle bg-surface px-4 py-5 sm:px-8">
        <nav
          aria-label="Onboarding resources"
          className="flex flex-wrap justify-center gap-x-6 gap-y-2 text-xs"
        >
          <Link href={theme.links.terms}>Terms</Link>
          <Link href={theme.links.privacy}>Privacy</Link>
          <Link href={theme.links.documentation}>Documentation</Link>
          <Link href={theme.links.support}>Support</Link>
        </nav>
      </footer>
    </div>
  );
}

export function OnbaordingRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <OnboardingLayout />
          </ProtectedBox>
        }
      >
        <Route index element={<Navigate to="organization" replace />} />
        <Route
          key="organization"
          path="organization"
          element={<OnboardingCreateOrganizationPage />}
        />
        <Route
          key="project"
          path="project"
          element={<OnboardingCreateProjectPage />}
        />
      </Route>
    </Routes>
  );
}
