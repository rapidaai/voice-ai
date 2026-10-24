import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AssistantSideNav } from '../assistant-side-nav';

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
});
