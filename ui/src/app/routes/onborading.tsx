import { Navigate, Outlet, Route, Routes, useLocation } from 'react-router-dom';
import { GeneralFooter } from '@/app/components/layout/footer/general-footer';
import { PublicShellHeader } from '@/app/components/layout/public-shell/public-shell-header';
import { ProtectedBox } from '@/app/components/layout/container/protected-box';
import { Text } from '@/app/components/ui/primitives/text';
import {
  OnboardingCreateOrganizationPage,
  OnboardingCreateProjectPage,
} from '@/app/pages/user-onboarding';

const STEPS = [
  { path: 'organization', label: 'Organization' },
  { path: 'project', label: 'Project' },
];

export function OnboardingLayout() {
  const location = useLocation();
  const currentStepIndex = STEPS.findIndex(step =>
    location.pathname.includes(step.path),
  );
  const currentStep = currentStepIndex >= 0 ? currentStepIndex + 1 : 1;

  return (
    <div className="flex min-h-dvh flex-col bg-layer text-foreground antialiased">
      <PublicShellHeader />

      <main
        aria-label="Onboarding"
        className="flex min-h-0 flex-1 items-center justify-center overflow-auto bg-layer p-4 sm:p-10"
        data-testid="onboarding-main"
      >
        <section
          aria-label="Workspace setup"
          className="my-auto w-full max-w-[37rem] border-0 bg-surface p-6 sm:p-8"
        >
          <div className="mb-10 flex items-start justify-between gap-4">
            <div>
              <Text as="p" variant="label-01" className="text-muted">
                Workspace setup
              </Text>
              <Text as="p" variant="heading-compact-01" className="mt-1">
                {STEPS[currentStep - 1].label}
              </Text>
            </div>
            <Text
              as="p"
              variant="body-compact-01"
              className="shrink-0 text-muted"
            >
              Step {currentStep} of {STEPS.length}
            </Text>
          </div>

          <Outlet />
        </section>
      </main>

      <GeneralFooter />
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
