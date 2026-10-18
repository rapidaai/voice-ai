import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { CreateKnowledge } from './create-assistant-knowledge';
import { UpdateKnowledge } from './update-assistant-knowledge';

const mockGetAssistantKnowledge = jest.fn();
const mockGoBack = jest.fn();
const mockHideLoader = jest.fn();
const mockShowLoader = jest.fn();

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

jest.mock('react-router-dom', () => ({
  useParams: () => ({ assistantKnowledgeId: 'assistant-knowledge-1' }),
}));

jest.mock('@rapidaai/react', () => ({
  CreateAssistantKnowledge: jest.fn(),
  GetAssistantKnowledge: (...args: any[]) => mockGetAssistantKnowledge(...args),
  UpdateAssistantKnowledge: jest.fn(),
}));

jest.mock('@/app/components/domain/dropdowns/knowledge-dropdown', () => ({
  KnowledgeDropdown: () => <div data-testid="knowledge-dropdown" />,
}));

jest.mock('@/app/components/domain/providers/reranker', () => ({
  GetDefaultRerankerConfigIfInvalid: (_provider: string, current: unknown[]) =>
    current,
  RerankerProvider: () => <div data-testid="reranker-provider" />,
}));

jest.mock('@/app/components/layout/blocks/page-action-button-block', () => ({
  PageActionButtonBlock: ({ children, errorMessage }: any) => (
    <div>
      {errorMessage ? <div role="alert">{errorMessage}</div> : null}
      {children}
    </div>
  ),
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  GhostButton: ({ children, ...props }: React.ComponentProps<'button'>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
  PrimaryButton: ({ children, ...props }: React.ComponentProps<'button'>) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/ui/primitives/card', () => ({
  Card: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/primitives/checkbox-card', () => ({
  __esModule: true,
  default: ({ children, ...props }: React.ComponentProps<'input'>) => (
    <label>
      <input {...props} />
      {children}
    </label>
  ),
}));

jest.mock('@/app/components/ui/primitives/fieldset', () => ({
  FieldSet: ({ children, className }: any) => (
    <fieldset className={className}>{children}</fieldset>
  ),
}));

jest.mock('@/app/components/ui/primitives/form-label', () => ({
  FormLabel: ({ children, htmlFor }: any) => (
    <label htmlFor={htmlFor}>{children}</label>
  ),
}));

jest.mock('@/app/components/ui/primitives/input', () => ({
  Input: (props: React.ComponentProps<'input'>) => <input {...props} />,
}));

jest.mock('@/app/components/ui/primitives/input-helper', () => ({
  InputHelper: ({ children }: any) => <p>{children}</p>,
}));

jest.mock('@/app/components/ui/primitives/slider', () => ({
  Slider: ({ onSlide, value }: any) => (
    <input
      aria-label="Slider"
      type="range"
      value={value}
      onChange={event => onSlide(Number(event.target.value))}
    />
  ),
}));

jest.mock('@/app/components/ui/primitives/switch', () => ({
  SwitchWithLabel: ({ label }: any) => <div>{label}</div>,
}));

jest.mock('@/app/components/ui/primitives/tooltip', () => ({
  Tooltip: ({ children, icon }: any) => (
    <span>
      {icon}
      {children}
    </span>
  ),
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    ConfirmDialogComponent: () => null,
    showDialog: (callback: () => void) => callback(),
  }),
}));

jest.mock('@/configs', () => ({
  connectionConfig: {},
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    hideLoader: mockHideLoader,
    showLoader: mockShowLoader,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    projectId: 'project-1',
    token: 'token-1',
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: mockGoBack,
    goToConfigureAssistantKnowledge: jest.fn(),
  }),
}));

jest.mock('@carbon/icons-react', () => ({
  DataBase: ({ className }: any) => (
    <svg className={className} data-testid="database-icon" />
  ),
  Information: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="information-icon"
    />
  ),
  ModelAlt: ({ className }: any) => (
    <svg className={className} data-testid="model-icon" />
  ),
  Search: ({ className }: any) => (
    <svg className={className} data-testid="search-icon" />
  ),
}));

describe('assistant knowledge configuration forms', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetAssistantKnowledge.mockImplementation(
      (_config, _assistantId, _id, callback) => {
        callback(null, {
          getData: () => ({
            getAssistantknowledgererankeroptionsList: () => [],
            getKnowledgeid: () => 'knowledge-1',
            getRerankerenable: () => false,
            getRetrievalmethod: () => 'hybrid',
            getScorethreshold: () => 0.5,
            getTopk: () => 5,
          }),
        });
      },
    );
  });

  it('uses Carbon information icons in the create form tooltips', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<CreateKnowledge assistantId="assistant-1" />);

    const icons = screen.getAllByTestId('information-icon');
    expect(icons).toHaveLength(2);
    icons.forEach(icon => {
      expect(icon).not.toHaveAttribute('data-stroke-width');
    });

    await user.click(
      screen.getByRole('button', { name: /connect knowledge/i }),
    );

    expect(screen.getByRole('alert')).toHaveTextContent(
      'Please select a valid knowledge.',
    );
  });

  it('uses Carbon information icons in the update form tooltips', () => {
    render(<UpdateKnowledge assistantId="assistant-1" />);

    const icons = screen.getAllByTestId('information-icon');
    expect(icons).toHaveLength(2);
    icons.forEach(icon => {
      expect(icon).not.toHaveAttribute('data-stroke-width');
    });
    expect(mockGetAssistantKnowledge).toHaveBeenCalled();
  });
});
