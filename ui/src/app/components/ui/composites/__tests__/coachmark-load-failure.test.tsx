import React, { Component, createRef, type ReactNode } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { Coachmark } from '../coachmark';

jest.mock('../coachmark-content', () => {
  throw new Error('Unable to load coachmark content');
});

class OuterErrorBoundary extends Component<
  { children: ReactNode },
  { hasError: boolean }
> {
  state = { hasError: false };

  static getDerivedStateFromError() {
    return { hasError: true };
  }

  render() {
    return this.state.hasError ? <div>Outer error</div> : this.props.children;
  }
}

describe('Coachmark loading failure', () => {
  it('keeps the target usable when optional coachmark content cannot load', async () => {
    const consoleError = jest
      .spyOn(console, 'error')
      .mockImplementation(() => undefined);

    render(
      <OuterErrorBoundary>
        <Coachmark
          actionLabel="Next"
          align="bottom"
          description="Description"
          launcherButtonRef={createRef<HTMLButtonElement>()}
          onAction={jest.fn()}
          onClose={jest.fn()}
          open
          title="Title"
        >
          <button type="button">Target</button>
        </Coachmark>
      </OuterErrorBoundary>,
    );

    await waitFor(() => expect(consoleError).toHaveBeenCalled());
    expect(screen.getByRole('button', { name: 'Target' })).toBeVisible();
    expect(screen.queryByText('Outer error')).not.toBeInTheDocument();
    expect(screen.queryByRole('region')).not.toBeInTheDocument();
    consoleError.mockRestore();
  });
});
