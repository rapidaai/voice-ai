import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  RecordStatusIndicator,
  recordStatusToShapeIndicator,
} from '.';

jest.mock('@carbon/react', () => ({
  preview__ShapeIndicator: ({ kind, label, textSize }: any) => (
    <span data-kind={kind} data-text-size={textSize}>
      {label}
    </span>
  ),
}));

describe('RecordStatusIndicator', () => {
  it('maps active records to stable', () => {
    expect(recordStatusToShapeIndicator('ACTIVE')).toEqual({
      kind: 'stable',
      label: 'Active',
    });

    render(<RecordStatusIndicator state="ACTIVE" textSize={14} />);

    expect(screen.getByText('Active')).toHaveAttribute('data-kind', 'stable');
    expect(screen.getByText('Active')).toHaveAttribute('data-text-size', '14');
  });

  it('maps inactive records to undefined', () => {
    expect(recordStatusToShapeIndicator('INACTIVE')).toEqual({
      kind: 'undefined',
      label: 'Inactive',
    });

    render(<RecordStatusIndicator state="INACTIVE" />);

    expect(screen.getByText('Inactive')).toHaveAttribute(
      'data-kind',
      'undefined',
    );
  });

  it('maps any other record status to draft', () => {
    expect(recordStatusToShapeIndicator('PENDING')).toEqual({
      kind: 'draft',
      label: 'Draft',
    });
    expect(recordStatusToShapeIndicator('active')).toEqual({
      kind: 'draft',
      label: 'Draft',
    });

    render(<RecordStatusIndicator state="PENDING" />);

    expect(screen.getByText('Draft')).toHaveAttribute('data-kind', 'draft');
  });
});
