import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { ConfigureEndpointPromptDialog } from '../configure-endpoint-prompt-modal';
import { EndpointArguments } from '../endpoint-trace-arguments';
import { EndpointMetadatas } from '../endpoint-trace-metadatas';
import { EndpointOptions } from '../endpoint-trace-options';
import { EndpointTraceModal } from '../endpoint-trace-modal';

jest.mock('@carbon/react', () => ({
  ProgressBar: ({ value }: any) => (
    <div aria-valuenow={value} role="progressbar" />
  ),
  SelectableTile: ({ children, onClick, selected }: any) => (
    <button
      type="button"
      aria-pressed={selected}
      data-selected={String(Boolean(selected))}
      onClick={onClick}
    >
      {children}
    </button>
  ),
  StructuredListBody: ({ children }: any) => <div>{children}</div>,
  StructuredListCell: ({ children }: any) => <span>{children}</span>,
  StructuredListRow: ({ children }: any) => <div>{children}</div>,
  StructuredListWrapper: ({ children }: any) => <section>{children}</section>,
  Table: ({ children }: any) => <table>{children}</table>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children }: any) => <td>{children}</td>,
  TableContainer: ({ children, title }: any) => (
    <section>
      {title ? <h2>{title}</h2> : null}
      {children}
    </section>
  ),
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children }: any) => <th>{children}</th>,
  TableRow: ({ children }: any) => <tr>{children}</tr>,
  Tag: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@carbon/icons-react', () => ({
  ChartLine: () => <svg data-testid="metrics-empty-icon" />,
  CheckmarkFilled: () => <svg data-testid="success-icon" />,
  DataBase: () => <svg data-testid="metadata-empty-icon" />,
  DataCheck: () => <svg data-testid="arguments-empty-icon" />,
  ErrorFilled: () => <svg data-testid="error-icon" />,
  Information: () => <svg data-testid="info-icon" />,
  ModelAlt: () => <svg data-testid="options-empty-icon" />,
}));

jest.mock('@/app/components/ui/feedback', () => ({
  CarbonStatusIndicator: ({ state }: any) => <span>Status {state}</span>,
  EmptyState: ({ title, subtitle, icon: Icon }: any) => (
    <div>
      {Icon ? <Icon /> : null}
      <h3>{title}</h3>
      <p>{subtitle}</p>
    </div>
  ),
}));

jest.mock('@/app/components/domain/indicators/source', () => ({
  SourceIndicator: ({ source }: any) => <span>Source {source}</span>,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Modal: ({ children, open }: any) =>
    open ? <section>{children}</section> : null,
  ModalBody: ({ children }: any) => <main>{children}</main>,
  ModalFooter: ({ children }: any) => <footer>{children}</footer>,
  ModalHeader: ({ label, title }: any) => (
    <header>
      <p>{label}</p>
      <h2>{title}</h2>
    </header>
  ),
  PrimaryButton: ({ children, disabled, onClick }: any) => (
    <button type="button" disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
  SecondaryButton: ({ children, onClick }: any) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  Tabs: ({ children, tabs }: any) => (
    <div>
      <nav>{tabs.join(', ')}</nav>
      {children}
    </div>
  ),
  Tooltip: ({ children, icon }: any) => (
    <span>
      {icon}
      {children}
    </span>
  ),
}));

jest.mock('@/app/components/dialogs/shared/right-side-modal', () => ({
  RightSideModal: ({ children, label, modalOpen, title }: any) =>
    modalOpen ? (
      <aside>
        <p>{label}</p>
        <h2>{title}</h2>
        {children}
      </aside>
    ) : null,
}));

const makeArgument = (name: string, value: string) =>
  ({
    getName: () => name,
    getValue: () => value,
  }) as any;

const makeMetadata = (key: string, value: string) =>
  ({
    getKey: () => key,
    getValue: () => value,
  }) as any;

const makeTrace = (createdDate: any = null) =>
  ({
    getArgumentsList: () => [makeArgument('customer_id', '123')],
    getCreateddate: () => createdDate,
    getEndpointprovidermodelid: () => 'model-1',
    getId: () => 'trace-1',
    getMetadataList: () => [makeMetadata('region', 'us-east')],
    getMetricsList: () => [
      {
        getDescription: () => 'Total request latency',
        getName: () => 'latency',
        getValue: () => '245',
      },
    ],
    getOptionsList: () => [makeMetadata('temperature', '0.7')],
    getSource: () => 'api',
    getStatus: () => 'success',
    getTimetaken: () => 12000000,
  }) as any;

describe('endpoint dialogs', () => {
  it('renders empty trace panel states', () => {
    render(
      <>
        <EndpointArguments args={[]} />
        <EndpointMetadatas metadata={[]} />
        <EndpointOptions options={[]} />
      </>,
    );

    expect(screen.getByText('No arguments found')).toBeInTheDocument();
    expect(screen.getByTestId('arguments-empty-icon')).toBeInTheDocument();
    expect(screen.getByText('No metadata found')).toBeInTheDocument();
    expect(screen.getByTestId('metadata-empty-icon')).toBeInTheDocument();
    expect(screen.getByText('No model options found')).toBeInTheDocument();
    expect(screen.getByTestId('options-empty-icon')).toBeInTheDocument();
  });

  it('renders endpoint trace arguments, metadata, and options with Carbon structures', () => {
    render(
      <>
        <EndpointArguments args={[makeArgument('prompt', 'hello')]} />
        <EndpointMetadatas metadata={[makeMetadata('route', 'support')]} />
        <EndpointOptions options={[makeMetadata('temperature', '0.5')]} />
      </>,
    );

    expect(
      screen.getByRole('columnheader', { name: 'Name' }),
    ).toBeInTheDocument();
    expect(screen.getByText('prompt')).toBeInTheDocument();
    expect(screen.getByText('hello')).toBeInTheDocument();
    expect(screen.getByText('route')).toBeInTheDocument();
    expect(screen.getByText('support')).toBeInTheDocument();
    expect(screen.getByText('temperature')).toBeInTheDocument();
    expect(screen.getByText('0.5')).toBeInTheDocument();
  });

  it('shows the trace overview and panel content in the drawer', () => {
    render(
      <EndpointTraceModal
        modalOpen
        setModalOpen={jest.fn()}
        currentTrace={makeTrace()}
      />,
    );

    expect(screen.getByText('Trace')).toBeInTheDocument();
    expect(screen.getByText('trace-1')).toBeInTheDocument();
    expect(
      screen.getByText('Overview, Metrics, Metadata, Options, Arguments'),
    ).toBeInTheDocument();
    expect(screen.getByText('Status success')).toBeInTheDocument();
    expect(screen.getByText('Source api')).toBeInTheDocument();
    expect(screen.getByText('vrsn_model-1')).toBeInTheDocument();
    expect(screen.getByText('12ms')).toBeInTheDocument();
    expect(screen.getByText('latency')).toBeInTheDocument();
    expect(screen.getByText('customer_id')).toBeInTheDocument();
  });

  it('shows a trace creation timestamp when it is available', () => {
    render(
      <EndpointTraceModal
        modalOpen
        setModalOpen={jest.fn()}
        currentTrace={makeTrace({
          getNanos: () => 0,
          getSeconds: () => 1704067200,
        })}
      />,
    );

    expect(screen.getByText('Created')).toBeInTheDocument();
  });

  it('does not render trace content without a selected trace', () => {
    const { container } = render(
      <EndpointTraceModal
        modalOpen
        setModalOpen={jest.fn()}
        currentTrace={null}
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it('selects an endpoint template and returns it', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onSelectTemplate = jest.fn();
    const setModalOpen = jest.fn();

    render(
      <ConfigureEndpointPromptDialog
        modalOpen
        setModalOpen={setModalOpen}
        onSelectTemplate={onSelectTemplate}
      />,
    );

    expect(screen.getByRole('button', { name: 'Use template' })).toBeDisabled();

    await user.click(
      screen.getByRole('button', {
        name: /Churn Risk Detection/,
      }),
    );
    await user.click(screen.getByRole('button', { name: 'Use template' }));

    expect(onSelectTemplate).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Churn Risk Detection' }),
    );
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('closes after selecting a template when no selection callback is provided', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const setModalOpen = jest.fn();

    render(
      <ConfigureEndpointPromptDialog modalOpen setModalOpen={setModalOpen} />,
    );

    await user.click(
      screen.getByRole('button', {
        name: /Churn Risk Detection/,
      }),
    );
    await user.click(screen.getByRole('button', { name: 'Use template' }));

    expect(setModalOpen).toHaveBeenCalledWith(false);
  });

  it('closes the template dialog without selecting a template', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onSelectTemplate = jest.fn();
    const setModalOpen = jest.fn();

    render(
      <ConfigureEndpointPromptDialog
        modalOpen
        setModalOpen={setModalOpen}
        onSelectTemplate={onSelectTemplate}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(onSelectTemplate).not.toHaveBeenCalled();
    expect(setModalOpen).toHaveBeenCalledWith(false);
  });
});
