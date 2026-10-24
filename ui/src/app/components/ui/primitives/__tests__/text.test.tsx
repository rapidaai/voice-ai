import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Text, typographyClassNames } from '../text';

describe('Text', () => {
  it('applies the shared semantic typography scale', () => {
    render(
      <>
        <Text as="h1" variant="pageTitle">
          Project overview
        </Text>
        <Text as="p" variant="helper" className="text-muted">
          Supporting context
        </Text>
      </>,
    );

    expect(
      screen.getByRole('heading', { name: 'Project overview' }),
    ).toHaveClass('text-2xl', 'font-semibold', 'leading-8');
    expect(screen.getByText('Supporting context')).toHaveClass(
      'text-xs',
      'font-normal',
      'leading-4',
      'text-muted',
    );
  });

  it('provides the canonical Carbon utility, body, and fixed heading typesets', () => {
    expect(typographyClassNames).toMatchObject({
      'code-01':
        'font-mono text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
      'code-02':
        'font-mono text-[0.875rem] font-normal leading-[1.25rem] tracking-[0.32px]',
      'label-01':
        'font-sans text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
      'label-02':
        'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
      'helper-text-01':
        'font-sans text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
      'helper-text-02':
        'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
      'legal-01':
        'font-sans text-[0.75rem] font-normal leading-[1rem] tracking-[0.32px]',
      'legal-02':
        'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
      'body-compact-01':
        'font-sans text-[0.875rem] font-normal leading-[1.125rem] tracking-[0.16px]',
      'body-compact-02':
        'font-sans text-[1rem] font-normal leading-[1.375rem] tracking-[0]',
      'body-01':
        'font-sans text-[0.875rem] font-normal leading-[1.25rem] tracking-[0.16px]',
      'body-02':
        'font-sans text-[1rem] font-normal leading-[1.5rem] tracking-[0]',
      'heading-compact-01':
        'font-sans text-[0.875rem] font-[600] leading-[1.125rem] tracking-[0.16px]',
      'heading-compact-02':
        'font-sans text-[1rem] font-[600] leading-[1.375rem] tracking-[0]',
      'heading-01':
        'font-sans text-[0.875rem] font-[600] leading-[1.25rem] tracking-[0.16px]',
      'heading-02':
        'font-sans text-[1rem] font-[600] leading-[1.5rem] tracking-[0]',
      'heading-03':
        'font-sans text-[1.25rem] font-normal leading-[1.75rem] tracking-[0]',
      'heading-04':
        'font-sans text-[1.75rem] font-normal leading-[2.25rem] tracking-[0]',
      'heading-05':
        'font-sans text-[2rem] font-normal leading-[2.5rem] tracking-[0]',
      'heading-06':
        'font-sans text-[2.625rem] font-light leading-[3.125rem] tracking-[0]',
      'heading-07':
        'font-sans text-[3.375rem] font-light leading-[4rem] tracking-[0]',
    });

    render(
      <>
        <Text as="h1" variant="heading-06">
          Log in
        </Text>
        <Text as="p" variant="body-02">
          Don’t have an account?
        </Text>
      </>,
    );

    expect(screen.getByRole('heading', { name: 'Log in' })).toHaveClass(
      'text-[2.625rem]',
      'font-sans',
      'font-light',
      'leading-[3.125rem]',
      'tracking-[0]',
    );
    expect(screen.getByText('Don’t have an account?')).toHaveClass(
      'font-sans',
      'text-[1rem]',
      'font-normal',
      'leading-[1.5rem]',
      'tracking-[0]',
    );
  });

  it('preserves existing text behavior when no variant is provided', () => {
    render(<Text as="p">Unstyled content</Text>);

    expect(screen.getByText('Unstyled content')).not.toHaveAttribute('class');
  });
});
