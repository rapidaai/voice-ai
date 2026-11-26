import {
  useRef,
  type ComponentProps,
  type ReactElement,
  type RefObject,
} from 'react';
import { Button } from '@carbon/react';
import { Coachmark as CarbonCoachmark } from '@carbon/ibm-products/lib/components/Coachmark/next/Coachmark/Coachmark';

export function CoachmarkContent({
  actionLabel,
  align,
  children,
  description,
  launcherButtonRef,
  onAction,
  onClose,
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
  targetClassName?: string;
  title: string;
}) {
  const hasClosedRef = useRef(false);
  const restoreLauncherFocus = () => {
    window.requestAnimationFrame(() => {
      if (
        document.activeElement !== document.body &&
        document.activeElement !== document.documentElement
      ) {
        return;
      }
      launcherButtonRef.current?.focus();
    });
  };
  const handleClose = () => {
    if (hasClosedRef.current) return;
    hasClosedRef.current = true;
    onClose();
    restoreLauncherFocus();
  };
  const handleAction = () => {
    onAction();
    restoreLauncherFocus();
  };

  return (
    <CarbonCoachmark
      align={align}
      className={targetClassName}
      dropShadow
      launcherButtonRef={launcherButtonRef}
      onClose={handleClose}
      open
      preventCloseOnClickOutside
      selectorPrimaryFocus=".app-coachmark__action"
    >
      {children}
      <CarbonCoachmark.Content
        aria-label={`${title} coachmark`}
        className="app-coachmark__content"
      >
        <CarbonCoachmark.ContentHeader closeIconDescription="Close tour" />
        <CarbonCoachmark.ContentBody>
          <h2>{title}</h2>
          <p>{description}</p>
          <Button
            className="app-coachmark__action"
            onClick={handleAction}
            size="sm"
          >
            {actionLabel}
          </Button>
        </CarbonCoachmark.ContentBody>
      </CarbonCoachmark.Content>
    </CarbonCoachmark>
  );
}
