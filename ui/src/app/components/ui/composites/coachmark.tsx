import {
  Component,
  lazy,
  Suspense,
  type ComponentProps,
  type ReactElement,
  type RefObject,
} from 'react';
import type { Coachmark as CarbonCoachmark } from '@carbon/ibm-products/lib/components/Coachmark/next/Coachmark/Coachmark';
import { cn } from '@/utils';

const LazyCoachmarkContent = lazy(() =>
  import('./coachmark-content').then(module => ({
    default: module.CoachmarkContent,
  })),
);

class CoachmarkErrorBoundary extends Component<
  { children: ReactElement; fallback: ReactElement },
  { hasError: boolean }
> {
  state = { hasError: false };

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  render() {
    return this.state.hasError ? this.props.fallback : this.props.children;
  }
}

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
  targetWrapperClassName,
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
  targetWrapperClassName?: string;
  title: string;
}) {
  if (!open) return children;

  return (
    <CoachmarkErrorBoundary fallback={children}>
      <Suspense fallback={children}>
        <LazyCoachmarkContent
          actionLabel={actionLabel}
          align={align}
          description={description}
          launcherButtonRef={launcherButtonRef}
          onAction={onAction}
          onClose={onClose}
          targetClassName={cn(
            '[&>.cds--popover-container]:w-full',
            targetClassName,
          )}
          targetWrapperClassName={targetWrapperClassName}
          title={title}
        >
          {children}
        </LazyCoachmarkContent>
      </Suspense>
    </CoachmarkErrorBoundary>
  );
}
