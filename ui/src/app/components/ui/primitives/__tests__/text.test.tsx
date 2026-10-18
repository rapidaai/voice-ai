import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Text } from '../text';

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

  it('preserves existing text behavior when no variant is provided', () => {
    render(<Text as="p">Unstyled content</Text>);

    expect(screen.getByText('Unstyled content')).not.toHaveAttribute('class');
  });
});
