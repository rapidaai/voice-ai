import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { HelpToggletip } from '../help-toggletip';

jest.mock('@carbon/icons-react', () => ({
  Information: () => <svg data-testid="information-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Toggletip: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
  ToggletipButton: ({
    children,
    label,
  }: {
    children: React.ReactNode;
    label: string;
  }) => (
    <button aria-label={label} type="button">
      {children}
    </button>
  ),
  ToggletipContent: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

describe('HelpToggletip', () => {
  it('renders an accessible information trigger and content', () => {
    render(
      <HelpToggletip
        label="Application name"
        helpText="Identify the application that will use this key."
      />,
    );

    expect(
      screen.getByRole('button', { name: 'Application name information' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Identify the application that will use this key.'),
    ).toBeInTheDocument();
    expect(screen.getByTestId('information-icon')).toBeInTheDocument();
  });

  it('renders nothing when help content is unavailable', () => {
    const { container } = render(<HelpToggletip label="Application name" />);

    expect(container).toBeEmptyDOMElement();
  });
});
