import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AccessSecurityPage } from './index';

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: ({ title }: any) => <div data-testid="helmet">{title}</div>,
}));

jest.mock('@/app/components/layout/heading/descriptive-heading', () => ({
  DescriptiveHeading: ({ heading }: any) => <h1>{heading}</h1>,
}));

jest.mock('@/theme/documentation-url', () => ({
  useDocumentationUrl: () => 'https://docs.example.test/security',
}));

jest.mock('@carbon/react', () => ({
  Link: ({ children, href, renderIcon: Icon, size }: any) => (
    <a href={href} data-design-system-link-size={size}>
      {children}
      {Icon ? <Icon /> : null}
    </a>
  ),
  Toggle: ({
    disabled,
    hideLabel,
    labelText,
    onToggle,
    size,
    toggled,
  }: any) => (
    <button
      type="button"
      role="switch"
      aria-checked={toggled}
      aria-label={hideLabel ? labelText : undefined}
      data-design-system-toggle-size={size}
      disabled={disabled}
      onClick={() => onToggle(!toggled)}
    >
      {!hideLabel ? labelText : null}
    </button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  ArrowRight: () => <svg data-testid="support-link-icon" />,
}));

describe('AccessSecurityPage', () => {
  it('renders the organization security content with the Carbon docs link', () => {
    render(<AccessSecurityPage />);

    expect(
      screen.getByRole('heading', { name: 'Organization Security' }),
    ).toBeInTheDocument();

    const link = screen.getByRole('link', {
      name: /Read the support documentation/i,
    });
    expect(link).toHaveAttribute('href', 'https://docs.example.test/security');
    expect(link).toHaveAttribute('data-design-system-link-size', 'sm');
    expect(screen.getByTestId('support-link-icon')).toBeInTheDocument();
  });

  it('shows two-factor authentication as a disabled Carbon toggle', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<AccessSecurityPage />);

    const toggle = screen.getByRole('switch', {
      name: 'Two-factor authentication',
    });

    expect(toggle).toBeDisabled();
    expect(toggle).toHaveAttribute('aria-checked', 'true');
    expect(toggle).toHaveAttribute('data-design-system-toggle-size', 'sm');

    await user.click(toggle);

    expect(toggle).toHaveAttribute('aria-checked', 'true');
  });
});
