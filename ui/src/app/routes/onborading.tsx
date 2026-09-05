import { ProtectedBox } from '@/app/components/container/protected-box';
import { Microphone, Globe, ChartLine } from '@carbon/icons-react';
import React from 'react';
import { Outlet, Route, Routes, useLocation } from 'react-router-dom';
import {
  OnboardingCreateOrganizationPage,
  OnboardingCreateProjectPage,
} from '@/app/pages/user-onboarding';
import { ProgressIndicator, ProgressStep, Tag } from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';
import { BrandedLogo } from '@/app/components/brand/branded-logo';

// ── Step definitions ──────────────────────────────────────────────────────────

const STEPS = [
  {
    path: 'organization',
    label: 'Create organization',
    description: 'Define ownership and access',
    step: 1,
  },
  {
    path: 'project',
    label: 'Create project',
    description: 'Separate clients, brands, or teams',
    step: 2,
  },
];

// ── Feature highlights ────────────────────────────────────────────────────────

const FEATURES = [
  {
    icon: Microphone,
    text: 'Own your stack, credentials, and deployment model',
  },
  { icon: Globe, text: 'Run separate brands, clients, or business units' },
  {
    icon: ChartLine,
    text: 'Scale with governance, observability, and audit trails',
  },
];

// ── Layout ────────────────────────────────────────────────────────────────────

export function OnboardingLayout() {
  const location = useLocation();
  const { theme } = useTheme();

  const currentStep =
    STEPS.find(s => location.pathname.includes(s.path))?.step ?? 1;

  return (
    <div className="min-h-[100dvh] flex bg-surface text-foreground">
      {/* ── Left brand panel ───────────────────────────────────────── */}
      <aside className="carbon-theme-g100 hidden lg:flex w-[400px] xl:w-[460px] flex-shrink-0 bg-surface text-foreground flex-col relative overflow-hidden">
        {/* Dot-grid decoration */}
        <div
          className="absolute inset-0 opacity-[0.04] pointer-events-none"
          style={{
            backgroundImage:
              'radial-gradient(circle, currentColor 1px, transparent 1px)',
            backgroundSize: '24px 24px',
          }}
        />

        {/* Logo */}
        <div className="relative px-10 pt-10">
          <BrandedLogo
            variant="full"
            colorMode="dark"
            className="h-8"
            textClassName="text-lg text-foreground"
          />
        </div>

        {/* Tagline + feature highlights */}
        <div className="relative flex-1 flex flex-col justify-center px-10 pb-8">
          <p className="text-[10px] font-semibold tracking-[0.16em] uppercase text-muted mb-4">
            Built for control
          </p>
          <h2 className="text-2xl font-light text-foreground mb-3 leading-snug">
            Build voice operations your agency or enterprise team actually
            controls.
          </h2>
          <p className="text-sm text-muted leading-relaxed mb-8">
            Use {theme.brand.name} to run branded assistants, isolate client or
            business-unit workloads, and scale with governance from day one.
          </p>

          {/* Feature list */}
          <div className="flex flex-col gap-3">
            {FEATURES.map(({ icon: Icon, text }) => (
              <div key={text} className="flex items-center gap-3">
                <div className="w-7 h-7 flex items-center justify-center bg-layer-hover shrink-0">
                  <Icon size={16} className="text-muted" />
                </div>
                <span className="text-sm text-foreground leading-5">
                  {text}
                </span>
              </div>
            ))}
          </div>
        </div>
      </aside>

      {/* ── Right form panel ───────────────────────────────────────── */}
      <main className="flex-1 flex flex-col min-w-0">
        {/* Mobile header */}
        <div className="lg:hidden flex items-center justify-between px-6 py-4 border-b border-border-subtle">
          <BrandedLogo
            variant="compact"
            className="h-7 w-7"
            textClassName="text-sm"
          />
          <Tag size="sm" type="blue">
            Step {currentStep} of {STEPS.length}
          </Tag>
        </div>

        {/* Form area */}
        <div className="flex-1 flex flex-col items-center justify-center px-6 sm:px-12 py-10">
          <div className="w-full max-w-md">
            {/* Progress indicator — horizontal, above form */}
            <div className="hidden lg:block mb-14">
              <ProgressIndicator currentIndex={currentStep - 1} spaceEqually>
                {STEPS.map(step => (
                  <ProgressStep
                    key={step.path}
                    label={step.label}
                    description={step.description}
                    secondaryLabel={`Step ${step.step}`}
                  />
                ))}
              </ProgressIndicator>
            </div>
            <Outlet />
          </div>
        </div>
      </main>
    </div>
  );
}

// ── Route ─────────────────────────────────────────────────────────────────────

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
