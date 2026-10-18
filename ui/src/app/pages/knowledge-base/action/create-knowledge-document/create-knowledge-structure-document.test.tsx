import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CreateKnowledgeStructureDocumentPage } from './create-knowledge-structure-document';
import { RapidaDocumentType } from '@/utils/rapida_document';

const mockClear = jest.fn();
const mockGoBack = jest.fn();
const mockGoToKnowledge = jest.fn();
const mockHideLoader = jest.fn();
const mockOnChangeDocumentType = jest.fn();
const mockOnCreateKnowledgeDocument = jest.fn();
const mockShowLoader = jest.fn();

let mockDocumentType = RapidaDocumentType.UNSTRUCTURE;
let mockRouteId: string | undefined = 'knowledge-1';

jest.mock('react-router-dom', () => ({
  useParams: () => ({ id: mockRouteId }),
}));

jest.mock('@carbon/react', () => ({
  ButtonSet: ({ children, className }: React.ComponentProps<'div'>) => (
    <div className={className}>{children}</div>
  ),
}));

jest.mock('@rapidaai/react', () => ({}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: ({ title }: any) => <div data-testid="helmet">{title}</div>,
}));

jest.mock('@/app/components/layout/blocks/section-divider', () => ({
  SectionDivider: ({ label }: any) => <h2>{label}</h2>,
}));

jest.mock(
  '@/app/components/layout/container/message/notice-block/doc-notice-block',
  () => ({
    DocNoticeBlock: ({ children }: any) => <aside>{children}</aside>,
  }),
);

jest.mock('@/app/components/ui/primitives/button', () => ({
  PrimaryButton: ({
    children,
    isLoading,
    ...props
  }: React.ComponentProps<'button'> & { isLoading?: boolean }) => (
    <button type="button" data-loading={isLoading} {...props}>
      {children}
    </button>
  ),
  SecondaryButton: ({ children, ...props }: React.ComponentProps<'button'>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

jest.mock('@/app/components/ui/feedback/error-message', () => ({
  ErrorMessage: ({ message }: any) =>
    message ? <div role="alert">{message}</div> : null,
}));

jest.mock('@/app/components/ui/primitives/form-label', () => ({
  FormLabel: ({ children }: any) => <label>{children}</label>,
}));

jest.mock('@/app/components/ui/primitives/select', () => ({
  Select: ({ onChange, options, placeholder }: any) => (
    <select
      aria-label={placeholder}
      defaultValue=""
      onChange={event => onChange(event)}
    >
      <option value="" disabled>
        {placeholder}
      </option>
      {options.map((option: any) => (
        <option key={option.value} value={option.value}>
          {option.name}
        </option>
      ))}
    </select>
  ),
}));

jest.mock(
  '@/app/pages/knowledge-base/action/components/datasource-uploader/manual-file',
  () => ({
    ManualFile: () => <div data-testid="manual-file" />,
  }),
);

jest.mock('@/stores/knowledge/create-knowledge-document.store', () => ({
  useCreateKnowledgeDocumentPageStore: () => ({
    clear: mockClear,
    documentType: mockDocumentType,
    onChangeDocumentType: mockOnChangeDocumentType,
    onCreateKnowledgeDocument: mockOnCreateKnowledgeDocument,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: mockGoBack,
    goToKnowledge: mockGoToKnowledge,
  }),
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    loading: false,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@carbon/icons-react', () => ({
  ArrowLeft: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="arrow-left-icon"
    />
  ),
}));

describe('CreateKnowledgeStructureDocumentPage', () => {
  beforeEach(() => {
    mockDocumentType = RapidaDocumentType.UNSTRUCTURE;
    mockRouteId = 'knowledge-1';
    jest.clearAllMocks();
  });

  it('renders the back action with a Carbon icon and navigates back', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateKnowledgeStructureDocumentPage />);

    expect(screen.getByTestId('arrow-left-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );

    await user.click(
      screen.getByRole('button', { name: /back to knowledge/i }),
    );

    expect(mockGoBack).toHaveBeenCalledTimes(1);
  });

  it('blocks upload until a structured document type is selected', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateKnowledgeStructureDocumentPage />);

    await user.click(
      screen.getByRole('button', { name: /upload new document/i }),
    );

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please select document type of the file and try again.',
    );
    expect(mockShowLoader).not.toHaveBeenCalled();
    expect(mockOnCreateKnowledgeDocument).not.toHaveBeenCalled();
  });

  it('renders an invalid url message when the route id is missing', () => {
    mockRouteId = undefined;

    render(<CreateKnowledgeStructureDocumentPage />);

    expect(
      screen.getByText('Please check the url and try again.'),
    ).toBeInTheDocument();
    expect(screen.queryByTestId('arrow-left-icon')).not.toBeInTheDocument();
  });
});
