import {
  Children,
  isValidElement,
  type ComponentType,
  type ReactElement,
  type ReactNode,
  useId,
  useState,
} from 'react';
import {
  Button,
  ButtonSet,
  Form,
  ProgressIndicator,
  ProgressStep,
} from '@carbon/react';
import {
  StepGroup,
  StepProvider,
  useStepContext,
} from '@carbon/utilities-react';
import type { CarbonIconType } from '@carbon/icons-react';
import { Notification } from '@/app/components/ui/feedback/notification';
import { Text } from '@/app/components/ui/primitives/text';

const VisibleStepGroup = StepGroup as ComponentType<{ children: ReactNode }>;

export interface CreateFullPageStepProps {
  children: ReactNode;
  description: string;
  disableSubmit?: boolean;
  hideBackButton?: boolean;
  hideCancelButton?: boolean;
  hideHeader?: boolean;
  invalid?: boolean;
  nextButtonText?: string;
  onNext?: () => boolean | void | Promise<boolean | void>;
  subtitle?: ReactNode;
  title: string;
}

export interface CreateFullPageHeaderProps {
  description?: ReactNode;
  label?: ReactNode;
  title: ReactNode;
  titleStart?: ReactNode;
}

export interface CreateFullPageWithGlobalHeaderProps {
  children: ReactNode;
  cancelButtonText?: string;
  description: string;
  errorMessage?: string;
  eyebrow: string;
  headerIcon?: CarbonIconType;
  nextButtonText?: string;
  onCancel: () => void;
  onRequestSubmit: () => void | Promise<void>;
  submitButtonText: string;
  title: string;
}

export interface CreateFullPageWithHeaderProps {
  activeStep?: string;
  description?: string;
  errorMessage?: string;
  eyebrow?: string;
  headerIcon?: CarbonIconType;
  onChangeStep: (code: string) => void;
  steps: {
    actions: ReactElement[];
    body: ReactElement;
    code: string;
    description?: string;
    name: string;
  }[];
  title: string;
}

interface CreateFullPageFrameProps {
  children: ReactNode;
  currentIndex: number;
  description?: ReactNode;
  eyebrow?: ReactNode;
  headerIcon?: CarbonIconType;
  steps: {
    code: string;
    complete?: boolean;
    description?: string;
    invalid?: boolean;
    onClick?: () => void;
    title: string;
  }[];
  title: string;
}

export function CreateFullPageHeader({
  description,
  label,
  title,
  titleStart,
}: CreateFullPageHeaderProps) {
  return (
    <header className="shrink-0 border-b border-border-subtle bg-layer px-5 py-6 md:px-8">
      <div className="max-w-2xl">
        {label && (
          <Text
            as="div"
            className="mb-1 capitalize text-primary"
            variant="label-01"
          >
            {label}
          </Text>
        )}
        <div className="flex items-start gap-3 text-foreground">
          {titleStart && <span className="shrink-0 pt-0.5">{titleStart}</span>}
          <Text as="h1" variant="heading-04">
            {title}
          </Text>
        </div>
        {description && (
          <Text
            as="div"
            className="mt-2 text-foreground"
            variant="body-compact-01"
          >
            {description}
          </Text>
        )}
      </div>
    </header>
  );
}

function CreateFullPageFrame({
  children,
  currentIndex,
  description,
  eyebrow,
  headerIcon: HeaderIcon,
  steps,
  title,
}: CreateFullPageFrameProps) {
  return (
    <section aria-label={title} className="flex min-h-0 flex-1 flex-col">
      <CreateFullPageHeader
        description={description}
        label={eyebrow}
        title={title}
        titleStart={HeaderIcon ? <HeaderIcon size={32} /> : undefined}
      />

      <div className="flex min-h-0 flex-1 overflow-hidden bg-surface">
        <aside className="hidden w-80 shrink-0 flex-col border-r border-border-subtle bg-layer xl:flex">
          <div className="flex-1 overflow-y-auto px-8 py-6">
            <ProgressIndicator
              className="[&_.cds--progress-step]:min-h-20! [&_.cds--progress-step-button]:min-h-20!"
              currentIndex={currentIndex}
              vertical
            >
              {steps.map((step, index) => (
                <ProgressStep
                  complete={step.complete}
                  current={index === currentIndex}
                  invalid={step.invalid}
                  key={step.code}
                  label={step.title}
                  onClick={step.onClick}
                  secondaryLabel={step.description}
                />
              ))}
            </ProgressIndicator>
          </div>
        </aside>

        <div className="flex min-h-0 flex-1 flex-col bg-surface">
          {children}
        </div>
      </div>
    </section>
  );
}

export function CreateFullPageStep({
  children,
  description,
  hideHeader = false,
  subtitle,
  title,
}: CreateFullPageStepProps) {
  const titleId = useId();

  return (
    <section
      aria-label={hideHeader ? title : undefined}
      aria-labelledby={hideHeader ? undefined : titleId}
      className="flex min-h-full flex-col"
    >
      {!hideHeader && (
        <header className="border-b border-border-subtle px-6 pb-6 pt-8">
          <Text
            as="h2"
            className="text-foreground"
            id={titleId}
            variant="heading-04"
          >
            {title}
          </Text>
          {subtitle && (
            <Text
              as="div"
              className="mt-1.5 text-foreground"
              variant="heading-compact-01"
            >
              {subtitle}
            </Text>
          )}
          <Text as="div" className="mt-1.5 text-muted" variant="body-01">
            {description}
          </Text>
        </header>
      )}
      <div className="flex flex-1 flex-col">{children}</div>
    </section>
  );
}

function CreateFullPageContent({
  children,
  cancelButtonText = 'Cancel',
  description,
  errorMessage,
  eyebrow,
  headerIcon: HeaderIcon,
  nextButtonText = 'Next',
  onCancel,
  onRequestSubmit,
  submitButtonText,
  title,
}: CreateFullPageWithGlobalHeaderProps) {
  const { currentStep, handleNext, handlePrevious, totalSteps } =
    useStepContext();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [flowError, setFlowError] = useState('');
  const steps = Children.toArray(children).filter(
    (child): child is ReactElement<CreateFullPageStepProps> =>
      isValidElement<CreateFullPageStepProps>(child),
  );
  const currentStepData = steps[currentStep - 1]?.props;
  const isLastStep = currentStep === totalSteps;
  const frameSteps = steps.map((step, index) => ({
    code: `${index}-${String(step.props.title)}`,
    complete: index < currentStep - 1,
    description: step.props.description,
    invalid: step.props.invalid,
    title: step.props.title,
  }));

  const handlePrimaryAction = async () => {
    setFlowError('');
    setIsSubmitting(true);
    try {
      if (isLastStep) {
        await onRequestSubmit();
        return;
      }

      const canContinue = await currentStepData?.onNext?.();
      if (canContinue !== false) {
        handleNext();
      }
    } catch {
      setFlowError('Unable to continue. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <CreateFullPageFrame
      currentIndex={currentStep - 1}
      description={description}
      eyebrow={eyebrow}
      headerIcon={HeaderIcon}
      steps={frameSteps}
      title={title}
    >
      <Form
        aria-label={title}
        className="flex min-h-0 flex-1 flex-col"
        onSubmit={event => event.preventDefault()}
      >
        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="border-b border-border-subtle px-6 py-4 xl:hidden">
            <Text as="div" className="text-muted" variant="label-01">
              Step {currentStep} of {totalSteps}
            </Text>
            <Text
              as="div"
              className="mt-1 text-foreground"
              variant="heading-compact-01"
            >
              {currentStepData?.title}
            </Text>
            <Text
              as="div"
              className="mt-1 text-muted"
              variant="body-compact-01"
            >
              {currentStepData?.description}
            </Text>
          </div>
          <VisibleStepGroup>{steps}</VisibleStepGroup>
        </div>

        <div className="shrink-0 border-t border-border-subtle">
          {(flowError || errorMessage) && (
            <Notification
              kind="error"
              title="Error"
              subtitle={flowError || errorMessage}
            />
          )}
          <ButtonSet className="w-full [&>button]:max-w-none! [&>button]:flex-1">
            {!currentStepData?.hideCancelButton && (
              <Button kind="ghost" onClick={onCancel} size="lg" type="button">
                {cancelButtonText}
              </Button>
            )}
            {currentStep > 1 && !currentStepData?.hideBackButton && (
              <Button
                disabled={isSubmitting}
                kind="secondary"
                onClick={() => {
                  setFlowError('');
                  handlePrevious();
                }}
                size="lg"
                type="button"
              >
                Back
              </Button>
            )}
            <Button
              disabled={currentStepData?.disableSubmit || isSubmitting}
              kind="primary"
              onClick={handlePrimaryAction}
              size="lg"
              type="button"
            >
              {isLastStep
                ? submitButtonText
                : currentStepData?.nextButtonText ?? nextButtonText}
            </Button>
          </ButtonSet>
        </div>
      </Form>
    </CreateFullPageFrame>
  );
}

export function CreateFullPageWithHeader({
  activeStep,
  description,
  errorMessage,
  eyebrow,
  headerIcon,
  onChangeStep,
  steps,
  title,
}: CreateFullPageWithHeaderProps) {
  const activeIndex = steps.findIndex(step => step.code === activeStep);
  const currentIndex = Math.max(activeIndex, 0);
  const activeStepData = steps[activeIndex];

  return (
    <CreateFullPageFrame
      currentIndex={currentIndex}
      description={description}
      eyebrow={eyebrow}
      headerIcon={headerIcon}
      steps={steps.map((step, index) => ({
        code: step.code,
        complete: index < activeIndex,
        description: step.description,
        onClick: () => onChangeStep(step.code),
        title: step.name,
      }))}
      title={title}
    >
      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        {activeStepData && (
          <div className="flex flex-1 flex-col">
            <header className="border-b border-border-subtle px-6 pb-6 pt-8">
              <Text
                as="p"
                className="mb-1.5 text-muted xl:hidden"
                variant="label-01"
              >
                Step {activeIndex + 1} of {steps.length}
              </Text>
              <Text as="h2" className="text-foreground" variant="heading-04">
                {activeStepData.name}
              </Text>
              {activeStepData.description && (
                <Text as="p" className="mt-1.5 text-muted" variant="body-01">
                  {activeStepData.description}
                </Text>
              )}
            </header>

            {activeStepData.body}
          </div>
        )}
      </div>

      {activeStepData && (
        <div className="shrink-0 border-t border-border-subtle">
          {errorMessage && (
            <Notification kind="error" title="Error" subtitle={errorMessage} />
          )}
          {activeStepData.actions.map((action, index) => (
            <div key={`action-${index}`}>{action}</div>
          ))}
        </div>
      )}
    </CreateFullPageFrame>
  );
}

export function CreateFullPageWithGlobalHeader(
  props: CreateFullPageWithGlobalHeaderProps,
) {
  return (
    <StepProvider>
      <CreateFullPageContent {...props} />
    </StepProvider>
  );
}
