import { FC } from 'react';
import { preview__ShapeIndicator as ShapeIndicatorModule } from '@carbon/react';

const ShapeIndicator =
  (ShapeIndicatorModule as unknown as { default?: FC<any> }).default ||
  (ShapeIndicatorModule as unknown as FC<any>);

export type RecordStatusIndicatorKind =
  | 'failed'
  | 'critical'
  | 'high'
  | 'medium'
  | 'low'
  | 'cautious'
  | 'undefined'
  | 'stable'
  | 'informative'
  | 'incomplete'
  | 'draft';

export interface RecordStatusIndicatorProps {
  state?: string | null;
  label?: string;
  textSize?: 12 | 14;
}

export const recordStatusToShapeIndicator = (
  state?: string | null,
): { kind: RecordStatusIndicatorKind; label: string } => {
  if (state === 'ACTIVE') {
    return { kind: 'stable', label: 'Active' };
  }

  if (state === 'INACTIVE') {
    return { kind: 'undefined', label: 'Inactive' };
  }

  return { kind: 'draft', label: 'Draft' };
};

export const RecordStatusIndicator: FC<RecordStatusIndicatorProps> = ({
  state,
  label,
  textSize = 12,
}) => {
  const resolved = recordStatusToShapeIndicator(state);

  return (
    <ShapeIndicator
      kind={resolved.kind as any}
      label={label || resolved.label}
      textSize={textSize}
    />
  );
};
