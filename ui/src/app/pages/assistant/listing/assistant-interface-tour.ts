import { useCallback, useEffect, useState } from 'react';
import {
  completeInterfaceTour,
  isInterfaceTourCompleted,
} from '@/context/interface-tour-storage';

const ASSISTANT_TOUR_ID = 'assistant';

export type AssistantInterfaceTourStep = 'create' | 'types';

export type AssistantInterfaceTour = {
  step: AssistantInterfaceTourStep | null;
  showTypes: () => void;
  dismiss: () => void;
};

type TourProgress = {
  userId: string;
  step: AssistantInterfaceTourStep | null;
};

function initialProgress(userId: string): TourProgress {
  return {
    userId,
    step:
      userId && !isInterfaceTourCompleted(ASSISTANT_TOUR_ID) ? 'create' : null,
  };
}

export function useAssistantInterfaceTour(
  userId: string,
): AssistantInterfaceTour {
  const [progress, setProgress] = useState(() => initialProgress(userId));

  useEffect(() => {
    setProgress(current =>
      current.userId === userId ? current : initialProgress(userId),
    );
  }, [userId]);

  const showTypes = useCallback(() => {
    setProgress(current =>
      current.step === 'create' ? { ...current, step: 'types' } : current,
    );
  }, []);

  const dismiss = useCallback(() => {
    if (userId) {
      try {
        completeInterfaceTour(ASSISTANT_TOUR_ID);
      } catch {
        // The in-memory state still prevents the tour from blocking this session.
      }
    }
    setProgress(current => ({ ...current, step: null }));
  }, [userId]);

  return {
    step: progress.userId === userId ? progress.step : null,
    showTypes,
    dismiss,
  };
}
