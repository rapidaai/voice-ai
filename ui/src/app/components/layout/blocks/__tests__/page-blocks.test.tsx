import React from 'react';
import { render, screen } from '@testing-library/react';
import { PageActionButtonBlock } from '../page-action-button-block';
import { CompactPageHeader } from '../../compact-page-header';
import { PageTitleBlock } from '../page-title-block';
import { PageTitleWithCount } from '../page-title-with-count';
import { PaginationButtonBlock } from '../pagination-button-block';
import { SectionDivider } from '../section-divider';

describe('layout page blocks', () => {
  it('renders header, title, count, pagination, and divider content', () => {
    render(
      <CompactPageHeader>
        <PageTitleWithCount count={3} data-testid="title-count" total={8}>
          Assistants
        </PageTitleWithCount>
        <PaginationButtonBlock data-testid="pagination-actions">
          <button type="button">Previous</button>
          <button type="button">Next</button>
        </PaginationButtonBlock>
      </CompactPageHeader>,
    );

    expect(screen.getByText('Assistants')).toBeInTheDocument();
    expect(screen.getByText('3/8')).toBeInTheDocument();
    expect(screen.getByLabelText('3 of 8')).toHaveClass('text-muted');
    expect(screen.getByTestId('title-count')).toHaveClass('flex');
    expect(screen.getByTestId('pagination-actions')).toHaveClass(
      'divide-border-subtle',
    );
    expect(
      screen.getByRole('button', { name: 'Previous' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument();

    render(<SectionDivider className="mt-4" label="Provider" />);

    expect(screen.getByText('Provider')).toBeInTheDocument();
    expect(screen.getByRole('separator')).toHaveClass('mt-4');
  });

  it('renders action content and optional validation message', () => {
    render(
      <PageActionButtonBlock
        className="sticky"
        data-testid="page-actions"
        errorMessage="Fix the highlighted settings."
      >
        <button type="button">Save</button>
      </PageActionButtonBlock>,
    );

    expect(screen.getByTestId('page-actions')).toHaveClass('sticky');
    expect(screen.getAllByRole('alert')[0]).toHaveTextContent(
      'Fix the highlighted settings.',
    );
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
  });

  it('supports caller div attributes for title and header blocks', () => {
    render(
      <CompactPageHeader aria-label="Page toolbar" data-testid="page-header">
        <PageTitleBlock className="custom-title" id="page-title">
          Deployments
        </PageTitleBlock>
      </CompactPageHeader>,
    );

    expect(screen.getByTestId('page-header')).toHaveAccessibleName(
      'Page toolbar',
    );
    expect(screen.getByTestId('page-header')).toHaveClass(
      'border-border-subtle',
      'bg-shell',
    );
    expect(screen.getByTestId('page-header').tagName).toBe('HEADER');
    expect(screen.getByText('Deployments')).toHaveClass('custom-title');
    expect(screen.getByText('Deployments')).toHaveAttribute('id', 'page-title');
    expect(screen.getByText('Deployments')).toHaveClass('text-foreground');
  });
});
