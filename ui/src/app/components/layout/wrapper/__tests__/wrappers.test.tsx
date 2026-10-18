import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  ErrorWrapper,
  InfoWrapper,
  PlainWrapper,
  SuccessWrapper,
  WarnWrapper,
} from '../alert-wrapper';
import { BluredWrapper } from '../blured-wrapper';
import SingleRowWrapper from '../single-row-wrapper';

describe('layout wrappers', () => {
  it('renders alert wrapper variants with semantic Carbon status borders', () => {
    render(
      <>
        <ErrorWrapper data-testid="error">Error</ErrorWrapper>
        <SuccessWrapper data-testid="success">Success</SuccessWrapper>
        <InfoWrapper data-testid="info">Info</InfoWrapper>
        <WarnWrapper data-testid="warning">Warning</WarnWrapper>
        <PlainWrapper data-testid="plain">Plain</PlainWrapper>
      </>,
    );

    expect(screen.getByTestId('error')).toHaveClass(
      'bg-layer',
      '[border-left-color:var(--cds-support-error)]',
    );
    expect(screen.getByTestId('success')).toHaveClass(
      '[border-left-color:var(--cds-support-success)]',
    );
    expect(screen.getByTestId('info')).toHaveClass(
      '[border-left-color:var(--cds-support-info)]',
    );
    expect(screen.getByTestId('warning')).toHaveClass(
      '[border-left-color:var(--cds-support-warning)]',
    );
    expect(screen.getByTestId('plain')).toHaveClass(
      'border-border-subtle',
      'bg-layer',
    );
  });

  it('forwards custom attributes from alert wrappers', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onClick = jest.fn();

    render(
      <InfoWrapper
        aria-label="Information"
        className="custom-info"
        data-testid="info"
        onClick={onClick}
      >
        Details
      </InfoWrapper>,
    );

    await user.click(screen.getByLabelText('Information'));

    expect(onClick).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('info')).toHaveClass('custom-info');
    expect(screen.getByTestId('info')).toHaveTextContent('Details');
  });

  it('renders the header-style wrapper with theme tokens', () => {
    render(
      <BluredWrapper
        aria-label="Knowledge header"
        className="custom-header"
        data-testid="blurred"
      >
        Header content
      </BluredWrapper>,
    );

    expect(screen.getByLabelText('Knowledge header')).toHaveClass(
      'custom-header',
      'bg-shell',
      'border-border-subtle',
      'text-foreground',
    );
  });

  it('renders the single row wrapper and forwards interaction props', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onClick = jest.fn();

    render(
      <SingleRowWrapper
        aria-label="Manual file row"
        className="custom-row"
        data-testid="row"
        onClick={onClick}
      >
        Row content
      </SingleRowWrapper>,
    );

    await user.click(screen.getByLabelText('Manual file row'));

    expect(onClick).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('row')).toHaveClass(
      'custom-row',
      'bg-layer',
      'border-border-subtle',
    );
  });
});
