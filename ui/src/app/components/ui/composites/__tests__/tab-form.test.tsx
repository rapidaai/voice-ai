import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { TabForm } from '../tab-form';

jest.mock('@carbon/icons-react', () => ({
  Checkmark: () => <svg data-testid="checkmark-icon" />,
}));

jest.mock('@/app/components/ui/feedback/notification', () => ({
  Notification: ({ subtitle, title }: any) => (
    <div role="alert">
      {title}
      {subtitle}
    </div>
  ),
}));

const form = [
  {
    code: 'details',
    name: 'Details',
    description: 'Basic information',
    body: <div>Details body</div>,
    actions: [<button key="next">Next</button>],
  },
  {
    code: 'review',
    name: 'Review',
    body: <div>Review body</div>,
    actions: [<button key="save">Save</button>],
  },
];

describe('TabForm', () => {
  it('renders the active form body and actions', () => {
    render(
      <TabForm
        activeTab="details"
        onChangeActiveTab={jest.fn()}
        formHeading="Create configuration"
        form={form}
      />,
    );

    expect(screen.getByText('Create configuration')).toBeInTheDocument();
    expect(screen.getByText('Details body')).toBeInTheDocument();
    expect(screen.queryByText('Review body')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument();
  });

  it('calls the active tab change handler from the sidebar', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeActiveTab = jest.fn();

    render(
      <TabForm
        activeTab="details"
        onChangeActiveTab={onChangeActiveTab}
        form={form}
      />,
    );

    await user.click(screen.getByRole('button', { name: /review/i }));

    expect(onChangeActiveTab).toHaveBeenCalledWith('review');
  });

  it('marks completed steps with the Carbon checkmark and shows errors', () => {
    render(
      <TabForm
        activeTab="review"
        onChangeActiveTab={jest.fn()}
        errorMessage="Missing required value"
        form={form}
      />,
    );

    expect(screen.getByTestId('checkmark-icon')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Missing required value',
    );
  });
});
