import { useCallback, useEffect, useState } from 'react';
import {
  completeInterfaceTour,
  isInterfaceTourCompleted,
} from '@/context/interface-tour-storage';

const CREATE_ASSISTANT_TOUR_ID = 'createAssistant';

export type CreateAssistantInterfaceTourStep =
  | 'template'
  | 'model'
  | 'continue';

type TourProgress = {
  userId: string;
  step: CreateAssistantInterfaceTourStep | null;
};

function initialProgress(userId: string): TourProgress {
  return {
    userId,
    step:
      userId && !isInterfaceTourCompleted(CREATE_ASSISTANT_TOUR_ID)
        ? 'template'
        : null,
  };
}

export function useCreateAssistantInterfaceTour(userId: string) {
  const [progress, setProgress] = useState(() => initialProgress(userId));

  useEffect(() => {
    setProgress(current =>
      current.userId === userId ? current : initialProgress(userId),
    );
  }, [userId]);

  const showModel = useCallback(() => {
    setProgress(current =>
      current.step === 'template' ? { ...current, step: 'model' } : current,
    );
  }, []);

  const showContinue = useCallback(() => {
    setProgress(current =>
      current.step === 'model' ? { ...current, step: 'continue' } : current,
    );
  }, []);

  const dismiss = useCallback(() => {
    if (userId) {
      try {
        completeInterfaceTour(CREATE_ASSISTANT_TOUR_ID);
      } catch {
        // The in-memory state still prevents the tour from blocking this session.
      }
    }
    setProgress(current => ({ ...current, step: null }));
  }, [userId]);

  return {
    step: progress.userId === userId ? progress.step : null,
    showModel,
    showContinue,
    dismiss,
  };
}
