import { FieldSet } from '@/app/components/form/fieldset';
import { Label } from '@/app/components/form/label';
import { PrivateVisibilityIcon } from '@/app/components/Icon/private-visibility';
import { PublicVisibilityIcon } from '@/app/components/Icon/public-visibility';
import { FC } from 'react';

const visibilityOptions = [
  {
    value: 'private',
    label: 'Private',
    description: 'Only you or members of your organization can see and change.',
    icon: <PublicVisibilityIcon className="mt-2" />,
  },
  {
    value: 'public',
    label: 'Public',
    description:
      'Anyone on the internet can see. Only you or members of your organization can change.',
    icon: <PrivateVisibilityIcon className="mt-2" />,
  },
  // Add other options here
];

export const VisibilityInput: FC<{
  visibility: string;
  onChangeVisibility: (v: 'private' | 'public') => void;
  readonly?: boolean;
}> = ({ visibility, readonly, onChangeVisibility }) => (
  <FieldSet>
    <Label for="visibility">Visibility</Label>
    <FieldSet className="space-y-0">
      {visibilityOptions.map(option => (
        <div key={option.value} className="flex items-start mb-2">
          <input
            id={`v_${option.value}`}
            type="radio"
            name="visibility"
            value={option.value}
            disabled={visibility !== option.value && readonly}
            onChange={() => {
              option.value === 'private' && onChangeVisibility('private');
              option.value === 'public' && onChangeVisibility('public');
            }}
            className="form-input mr-2 mt-2 h-3.5 w-3.5 shrink-0"
            checked={visibility === option.value}
          />
          <label
            htmlFor={`v_${option.value}`}
            className="ms-2 space-x-2 flex items-start cursor-pointer"
          >
            {option.icon}
            <div className="flex flex-col">
              <div className="font-semibold">{option.label}</div>
              <span className="text-sm">{option.description}</span>
            </div>
          </label>
        </div>
      ))}
    </FieldSet>
  </FieldSet>
);
