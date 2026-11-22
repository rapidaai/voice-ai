import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { AssistantSideNav } from '../assistant-side-nav';
import { typographyClassNames } from '@/app/components/ui/primitives/text';

const assistant = {
  hasAssistantprovidermodel: () => true,
  getPhonedeployment: () => ({}),
} as any;

describe('AssistantSideNav', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  it('renders playground links without a modal page overlay', () => {
    const { container } = render(
      <MemoryRouter
        initialEntries={['/deployment/assistant/assistant-1/playground/chat']}
      >
        <AssistantSideNav
          assistantId="assistant-1"
          assistant={assistant}
          expanded
          onToggle={jest.fn()}
        />
      </MemoryRouter>,
    );

    expect(screen.getByRole('link', { name: 'Chat' })).toHaveAttribute(
      'href',
      '/deployment/assistant/assistant-1/playground/chat',
    );
    expect(screen.getByRole('link', { name: 'Phone call' })).toHaveAttribute(
      'href',
      '/deployment/assistant/assistant-1/playground/phone',
    );
    expect(container.querySelector('.cds--side-nav__overlay')).toBeNull();
  });

  it('matches the global sidebar surface and collapse control', async () => {
    const user = userEvent.setup();
    const onToggle = jest.fn();
    render(
      <MemoryRouter initialEntries={['/deployment/assistant/assistant-1']}>
        <AssistantSideNav
          assistantId="assistant-1"
          assistant={assistant}
          expanded
          onToggle={onToggle}
        />
      </MemoryRouter>,
    );

    const navigation = screen.getByLabelText('Assistant actions');
    expect(navigation.parentElement).toHaveClass('bg-shell', 'text-foreground');
    expect(navigation.parentElement).toHaveStyle({ width: '13.5rem' });
    expect(navigation).toHaveClass('app-side-nav', 'bg-transparent!');
    expect(navigation).toHaveClass(
      ...typographyClassNames['body-compact-01'].split(' '),
    );

    const toggle = screen.getByRole('button', {
      name: 'Collapse assistant navigation',
    });
    expect(toggle).toHaveClass('cds--btn--ghost');
    expect(toggle).toHaveClass('app-side-navigation-collapse-control');
    expect(toggle.querySelector('svg')).toBeInTheDocument();
    expect(toggle).toHaveTextContent('Collapse');

    await user.click(toggle);
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it('keeps navigation and toggle icons visible in the collapsed rail', () => {
    render(
      <MemoryRouter initialEntries={['/deployment/assistant/assistant-1']}>
        <AssistantSideNav
          assistantId="assistant-1"
          assistant={assistant}
          expanded={false}
          onToggle={jest.fn()}
        />
      </MemoryRouter>,
    );

    const navigation = screen.getByLabelText('Assistant actions');
    expect(navigation.parentElement).toHaveStyle({ width: '3rem' });
    expect(navigation).toHaveClass('transform-none!');
    expect(navigation).not.toHaveClass('translate-x-0!');
    expect(
      navigation.querySelector('.cds--side-nav__icon svg'),
    ).toBeInTheDocument();

    const toggle = screen.getByRole('button', {
      name: 'Expand assistant navigation',
    });
    expect(toggle.querySelector('svg')).toBeInTheDocument();
    expect(toggle).toHaveTextContent('Expand');
  });
});
