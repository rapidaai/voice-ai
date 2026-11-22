import {
  lazy,
  Suspense,
  type ComponentProps,
  type ReactElement,
  type RefObject,
} from 'react';
import type { Coachmark as CarbonCoachmark } from '@carbon/ibm-products/lib/components/Coachmark/next/Coachmark/Coachmark';

const LazyCoachmarkContent = lazy(() =>
  import('./coachmark-content').then(module => ({
    default: module.CoachmarkContent,
  })),
);

export function Coachmark({
  actionLabel,
  align,
  children,
  description,
  launcherButtonRef,
  onAction,
  onClose,
  open,
  targetClassName,
  title,
}: {
  actionLabel: string;
  align: ComponentProps<typeof CarbonCoachmark>['align'];
  children: ReactElement;
  description: string;
  launcherButtonRef: RefObject<HTMLElement>;
  onAction: () => void;
  onClose: () => void;
  open: boolean;
  targetClassName?: string;
  title: string;
}) {
  if (!open) return children;

  return (
    <Suspense fallback={children}>
      <LazyCoachmarkContent
        actionLabel={actionLabel}
        align={align}
        description={description}
        launcherButtonRef={launcherButtonRef}
        onAction={onAction}
        onClose={onClose}
        targetClassName={targetClassName}
        title={title}
      >
        {children}
      </LazyCoachmarkContent>
    </Suspense>
  );
}
