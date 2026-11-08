import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { useState } from 'react';
import { QuerySearch } from '../query-search';

const fields = [
  {
    queryKey: 'name',
    text: 'Name',
    type: 'string' as const,
  },
];

function QuerySearchHarness({ initialValue = '' }: { initialValue?: string }) {
  const [value, setValue] = useState(initialValue);

  return (
    <QuerySearch
      fields={fields}
      value={value}
      onApply={jest.fn()}
      onChange={setValue}
      placeholder="Filter conversations"
    />
  );
}

describe('QuerySearch', () => {
  it('renders a completed query token as an editable filter chip', () => {
    render(<QuerySearchHarness initialValue={'name:"Alice" '} />);

    expect(
      screen.getByRole('button', { name: 'Remove Name filter' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Alice' })).toBeInTheDocument();
  });

  it('removes a filter chip and restores the empty search input', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<QuerySearchHarness initialValue={'name:"Alice" '} />);
    await user.click(
      screen.getByRole('button', { name: 'Remove Name filter' }),
    );

    expect(screen.getByPlaceholderText('Filter conversations')).toHaveValue(
      '',
    );
  });
});
