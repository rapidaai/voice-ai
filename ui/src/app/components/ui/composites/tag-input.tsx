import type { KeyboardEvent } from 'react';
import { useState } from 'react';
import {
  DismissibleTag,
  Tag,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';
import { TextInput } from '@/app/components/ui/primitives/form';
import { FormLabel } from '@/app/components/ui/primitives/form-label';

interface TagInputProps {
  tags: string[];
  addTag: (tag: string) => void;
  removeTag: (tag: string) => void;
  allTags: Array<string>;
  className?: string;
  id?: string;
  labelText?: string;
  helperText?: string;
}

export function TagInput({
  tags,
  addTag,
  removeTag,
  allTags,
  id = 'tag-input',
  labelText = 'Tags (Optional)',
  helperText = 'Add tags to organize and locate items more efficiently.',
}: TagInputProps) {
  const [inputValue, setInputValue] = useState('');

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      const value = inputValue.trim();
      if (value && !tags.includes(value)) {
        addTag(value);
        setInputValue('');
      }
    }
  };

  const suggestedTags = allTags.filter(t => !tags.includes(t));

  return (
    <div>
      <div className="mb-2 flex items-center gap-1">
        <FormLabel htmlFor={id}>{labelText}</FormLabel>
        {helperText ? (
          <Toggletip align="right">
            <ToggletipButton label="Show information">
              <Information size={14} />
            </ToggletipButton>
            <ToggletipContent>{helperText}</ToggletipContent>
          </Toggletip>
        ) : null}
      </div>
      <TextInput
        id={id}
        labelText={labelText}
        hideLabel
        placeholder="Type a tag and press Enter"
        value={inputValue}
        onChange={e => setInputValue(e.target.value)}
        onKeyDown={handleKeyDown}
      />
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-3">
          {tags.map(t => (
            <DismissibleTag
              key={t}
              text={t}
              type="blue"
              size="md"
              onClose={() => removeTag(t)}
            />
          ))}
        </div>
      )}
      {suggestedTags.length > 0 && (
        <div className="mt-3">
          <p className="text-[10px] font-medium uppercase tracking-wider text-muted mb-1.5">
            Suggestions
          </p>
          <div className="flex flex-wrap gap-1">
            {suggestedTags.slice(0, 12).map(t => (
              <Tag
                key={t}
                type="outline"
                size="md"
                className="cursor-pointer"
                onClick={() => addTag(t)}
              >
                {t}
              </Tag>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
