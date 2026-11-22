import { useCallback, useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { useCurrentCredential } from '@/hooks/use-credential';
import {
  completeInterfaceTour,
  isInterfaceTourCompleted,
} from '@/context/interface-tour-storage';

const DASHBOARD_TOUR_ID = 'dashboard';

export type DashboardInterfaceTourStep =
  | 'context'
  | 'theme'
  | 'account'
  | 'navigation'
  | 'sidebar-toggle';

export type DashboardInterfaceTour = {
  step: DashboardInterfaceTourStep | null;
  next: () => void;
  dismiss: () => void;
};

type TourProgress = {
  userId: string;
  step: DashboardInterfaceTourStep | null;
};

function hasCompletedTour(userId: string) {
  if (!userId) return true;
  return isInterfaceTourCompleted(DASHBOARD_TOUR_ID);
}

function initialProgress(userId: string): TourProgress {
  return {
    userId,
    step: hasCompletedTour(userId) ? null : 'context',
  };
}

export function useDashboardInterfaceTour({
  hasThemeModeAction,
}: {
  hasThemeModeAction: boolean;
}): DashboardInterfaceTour {
  const { pathname } = useLocation();
  const { authId } = useCurrentCredential();
  const [progress, setProgress] = useState(() => initialProgress(authId));

  useEffect(() => {
    setProgress(current =>
      current.userId === authId ? current : initialProgress(authId),
    );
  }, [authId]);

  useEffect(() => {
    if (hasThemeModeAction) return;
    setProgress(current =>
      current.step === 'theme' ? { ...current, step: 'account' } : current,
    );
  }, [hasThemeModeAction]);

  const dismiss = useCallback(() => {
    if (authId) {
      try {
        completeInterfaceTour(DASHBOARD_TOUR_ID);
      } catch {
        // The in-memory state still prevents the tour from blocking this session.
      }
    }
    setProgress(current => ({ ...current, step: null }));
  }, [authId]);

  const next = useCallback(() => {
    if (progress.step === 'context') {
      setProgress(current => ({
        ...current,
        step: hasThemeModeAction ? 'theme' : 'account',
      }));
      return;
    }
    if (progress.step === 'theme') {
      setProgress(current => ({ ...current, step: 'account' }));
      return;
    }
    if (progress.step === 'account') {
      setProgress(current => ({ ...current, step: 'navigation' }));
      return;
    }
    if (progress.step === 'navigation') {
      setProgress(current => ({ ...current, step: 'sidebar-toggle' }));
      return;
    }
    if (progress.step === 'sidebar-toggle') dismiss();
  }, [dismiss, hasThemeModeAction, progress.step]);

  const isDashboard = pathname === '/dashboard' || pathname === '/dashboard/';

  return {
    step: isDashboard && progress.userId === authId ? progress.step : null,
    next,
    dismiss,
  };
}
