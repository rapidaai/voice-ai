import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import {
  CreateFullPageHeader,
  CreateFullPageStep,
  CreateFullPageWithHeader,
  CreateFullPageWithGlobalHeader,
} from '../create-full-page';

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    kind: _kind,
    size: _size,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement> & {
    kind?: string;
    size?: string;
  }) => <button {...props}>{children}</button>,
  ButtonSet: ({ children, ...props }: React.HTMLAttributes<HTMLDivElement>) => (
    <div {...props}>{children}</div>
  ),
  Form: ({ children, ...props }: React.FormHTMLAttributes<HTMLFormElement>) => (
    <form {...props}>{children}</form>
  ),
  Link: ({
    children,
    href,
    ...props
  }: React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
  ProgressIndicator: ({
    children,
    currentIndex,
  }: React.PropsWithChildren<{ currentIndex: number }>) => (
    <ol data-current-index={currentIndex}>{children}</ol>
  ),
  ProgressStep: ({
    complete,
    label,
    onClick,
    secondaryLabel,
  }: {
    complete?: boolean;
    label: React.ReactNode;
    onClick?: () => void;
    secondaryLabel?: React.ReactNode;
  }) => (
    <li data-complete={complete}>
      <button onClick={onClick} type="button">
        {label}
      </button>
      <span>{secondaryLabel}</span>
    </li>
  ),
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle }: { subtitle?: string }) => (
    <div role="alert">{subtitle}</div>
  ),
}));

const renderFlow = (props?: {
  onNext?: () => boolean | Promise<boolean>;
  onSubmit?: () => void | Promise<void>;
}) =>
  render(
    <CreateFullPageWithGlobalHeader
      title="Create resource"
      description="Create a new resource in two steps."
      errorMessage="Resolve the highlighted value."
      eyebrow="Resource"
      onCancel={jest.fn()}
      onRequestSubmit={props?.onSubmit ?? jest.fn()}
      submitButtonText="Create resource"
    >
      <CreateFullPageStep
        title="Configure"
        description="Configure the resource."
        onNext={props?.onNext}
      >
        <div>Configuration fields</div>
      </CreateFullPageStep>
      <CreateFullPageStep title="Review" description="Review the resource.">
        <div>Review fields</div>
      </CreateFullPageStep>
    </CreateFullPageWithGlobalHeader>,
  );

describe('CreateFullPageHeader', () => {
  it('renders the Carbon tearsheet header content contract', () => {
    render(
      <CreateFullPageHeader
        description="Create a new resource in two steps."
        label="resource"
        title="Create resource"
        titleStart={<svg aria-label="Resource icon" />}
      />,
    );

    expect(screen.getByText('resource')).toHaveClass(
      'font-sans',
      'text-[0.75rem]',
      'leading-[1rem]',
      'tracking-[0.32px]',
      'text-primary',
      'capitalize',
    );
    expect(screen.getByLabelText('Resource icon')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Create resource', level: 1 }),
    ).toHaveClass(
      'font-sans',
      'text-[1.75rem]',
      'font-normal',
      'leading-[2.25rem]',
      'tracking-[0]',
    );
    expect(screen.getByText('Create a new resource in two steps.')).toHaveClass(
      'font-sans',
      'text-[0.875rem]',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
    );
    expect(screen.getByRole('banner')).toHaveClass('bg-layer');
  });
});

describe('CreateFullPageWithHeader', () => {
  const steps = [
    {
      code: 'details',
      name: 'Details',
      description: 'Add the required details.',
      body: <div>Details body</div>,
      actions: [<button key="next">Next</button>],
    },
    {
      code: 'review',
      name: 'Review',
      description: 'Review the configuration.',
      body: <div>Review body</div>,
      actions: [<button key="save">Save</button>],
    },
  ];

  it('renders controlled steps with the shared header and footer', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeStep = jest.fn();

    render(
      <CreateFullPageWithHeader
        activeStep="details"
        description="Configure a reusable resource."
        errorMessage="Resolve the highlighted value."
        eyebrow="Resource"
        onChangeStep={onChangeStep}
        steps={steps}
        title="Configure resource"
      />,
    );

    expect(
      screen.getByRole('heading', { name: 'Configure resource' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Details body')).toBeInTheDocument();
    expect(screen.queryByText('Review body')).not.toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Resolve the highlighted value.',
    );

    await user.click(screen.getByRole('button', { name: 'Review' }));
    expect(onChangeStep).toHaveBeenCalledWith('review');
  });

  it('marks prior controlled steps complete', () => {
    render(
      <CreateFullPageWithHeader
        activeStep="review"
        description="Configure a reusable resource."
        onChangeStep={jest.fn()}
        steps={steps}
        title="Configure resource"
      />,
    );

    expect(screen.getByText('Details').closest('li')).toHaveAttribute(
      'data-complete',
      'true',
    );
    expect(screen.getByText('Review body')).toBeInTheDocument();
  });
});

describe('CreateFullPageStep', () => {
  it('uses the shared Carbon typesets for step content', () => {
    render(
      <CreateFullPageStep
        description="Configure the resource."
        subtitle="Required settings"
        title="Configure"
      >
        <div>Configuration fields</div>
      </CreateFullPageStep>,
    );

    expect(
      screen.getByRole('heading', { name: 'Configure', level: 2 }),
    ).toHaveClass(
      'font-sans',
      'text-[1.75rem]',
      'font-normal',
      'leading-[2.25rem]',
      'tracking-[0]',
    );
    expect(screen.getByText('Required settings')).toHaveClass(
      'font-sans',
      'text-[0.875rem]',
      'font-[600]',
      'leading-[1.125rem]',
      'tracking-[0.16px]',
    );
    expect(screen.getByText('Configure the resource.')).toHaveClass(
      'font-sans',
      'text-[0.875rem]',
      'font-normal',
      'leading-[1.25rem]',
      'tracking-[0.16px]',
    );
  });

  it('can omit repeated step heading content while retaining its fields', () => {
    render(
      <CreateFullPageStep
        description="Configure the resource."
        hideHeader
        title="Configure"
      >
        <div>Configuration fields</div>
      </CreateFullPageStep>,
    );

    expect(screen.getByText('Configuration fields')).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: 'Configure' }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('heading', { name: 'Configure' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText('Configure the resource.'),
    ).not.toBeInTheDocument();
  });
});

describe('CreateFullPageWithGlobalHeader', () => {
  it('moves forward, returns to the prior step, and submits', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onNext = jest.fn(() => true);
    const onSubmit = jest.fn();
    renderFlow({ onNext, onSubmit });

    expect(screen.getByText('Configuration fields')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Create resource' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Resource')).toBeInTheDocument();
    expect(
      screen.getByText('Create a new resource in two steps.'),
    ).toBeInTheDocument();
    expect(screen.getAllByText('Configure the resource.')).toHaveLength(3);
    expect(screen.getByText('Step 1 of 2')).toBeInTheDocument();
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
    expect(screen.queryByText('Review fields')).not.toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Resolve the highlighted value.',
    );

    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(onNext).toHaveBeenCalledTimes(1);
    expect(screen.getByText('Review fields')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByText('Configuration fields')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Next' }));
    await user.click(screen.getByRole('button', { name: 'Create resource' }));
    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it('stays on the current step when step validation fails', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onNext = jest.fn(() => false);
    renderFlow({ onNext });

    await user.click(screen.getByRole('button', { name: 'Next' }));

    expect(onNext).toHaveBeenCalledTimes(1);
    expect(screen.getByText('Configuration fields')).toBeInTheDocument();
    expect(screen.queryByText('Review fields')).not.toBeInTheDocument();
  });

  it('guards the submit action until the request resolves', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    let resolveSubmit: (() => void) | undefined;
    const onSubmit = jest.fn(
      () =>
        new Promise<void>(resolve => {
          resolveSubmit = resolve;
        }),
    );
    renderFlow({ onSubmit });

    await user.click(screen.getByRole('button', { name: 'Next' }));
    const submitButton = screen.getByRole('button', {
      name: 'Create resource',
    });
    await user.click(submitButton);

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(submitButton).toBeDisabled();
    await user.click(submitButton);
    expect(onSubmit).toHaveBeenCalledTimes(1);

    resolveSubmit?.();
    await waitFor(() => expect(submitButton).toBeEnabled());
  });

  it('uses step-specific navigation actions after an irreversible step', async () => {
    const user = userEvent.setup({ writeToClipboard: false });

    render(
      <CreateFullPageWithGlobalHeader
        title="Create resource"
        description="Create a new resource in three steps."
        eyebrow="Resource"
        onCancel={jest.fn()}
        onRequestSubmit={jest.fn()}
        submitButtonText="Finish"
      >
        <CreateFullPageStep title="Configure" description="Configure it.">
          <div>Configuration fields</div>
        </CreateFullPageStep>
        <CreateFullPageStep
          title="Create"
          description="Create the resource."
          hideBackButton
          hideCancelButton
          nextButtonText="Create and continue"
        >
          <div>Creation fields</div>
        </CreateFullPageStep>
        <CreateFullPageStep title="Finish" description="Finish setup.">
          <div>Finish fields</div>
        </CreateFullPageStep>
      </CreateFullPageWithGlobalHeader>,
    );

    await user.click(screen.getByRole('button', { name: 'Next' }));

    expect(
      screen.getByRole('button', { name: 'Create and continue' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull();
    expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull();
  });
});
