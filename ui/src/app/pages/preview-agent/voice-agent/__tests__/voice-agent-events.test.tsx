import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Table, TableBody } from '@carbon/react';
import { ConversationEventRow } from '../voice-agent';

jest.mock('@uiw/react-markdown-preview', () => ({
  __esModule: true,
  default: () => null,
}));

describe('ConversationEventRow', () => {
  it('expands and collapses a Carbon event row from the keyboard', async () => {
    const user = userEvent.setup();
    render(
      <Table>
        <TableBody>
          <ConversationEventRow
            entry={
              {
                type: 'pipelineEvent',
                ts: new Date('2026-01-01T12:00:00Z'),
                payload: {
                  name: 'agent',
                  dataMap: [['request', 'ready']],
                  id: 'event-1',
                },
              } as any
            }
          />
        </TableBody>
      </Table>,
    );

    const eventRow = screen.getByRole('row');
    const detail = () =>
      screen.queryByText(
        (content, element) =>
          element?.tagName === 'PRE' && content.includes('"request": "ready"'),
      );

    expect(eventRow).toHaveAttribute('tabindex', '0');
    expect(eventRow).toHaveAttribute('aria-expanded', 'false');
    expect(detail()).not.toBeInTheDocument();

    eventRow.focus();
    await user.keyboard('[Enter]');
    expect(eventRow).toHaveAttribute('aria-expanded', 'true');
    expect(detail()).toBeInTheDocument();

    await user.keyboard('[Enter]');
    expect(eventRow).toHaveAttribute('aria-expanded', 'false');
    expect(detail()).not.toBeInTheDocument();

    await user.keyboard('[Space]');
    expect(eventRow).toHaveAttribute('aria-expanded', 'true');
    expect(detail()).toBeInTheDocument();
  });
});
