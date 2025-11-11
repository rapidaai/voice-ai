import { FieldSet } from '@/app/components/Form/Fieldset';
import React, { FC } from 'react';
import { CloseIcon } from '@/app/components/Icon/Close';
import { Input } from '@/app/components/Form/Input';

/**
 *
 */
interface TagInputProps {
  tags: string[];
  addTag: (string) => void;
  removeTag: (string) => void;
  allTags: Array<string>;
  className?: string;
}

/**
 *
 * @param param0
 * @returns
 */
export const TagInput: FC<TagInputProps> = ({
  tags,
  addTag,
  removeTag,
  allTags,
  className,
}) => {
  //   all the tags

  //
  return (
    <div>
      <div className="mb-4 gap-2 flex">
        {tags.map((t, idx) => {
          return (
            <div
              key={idx}
              className="rounded-[2px] px-2 flex w-fit items-center justify-center shrink-0 border-[0.5px]! dark:border-gray-700 py-1 bg-gray-200 hover:border-blue-600 dark:hover:border-blue-600 dark:bg-gray-900"
            >
              <span className="ml-1.5 mr-1.5 text-sm">{t}</span>
              <CloseIcon
                className="h-3.5 w-3.5 cursor-pointer opacity-60 hover:opacity-90"
                stroke="currentColor"
                onClick={() => {
                  removeTag(t);
                }}
              />
            </div>
          );
        })}
      </div>
      <FieldSet>
        <div className="text-sm">Tags</div>
        <Input
          type="text"
          className={className}
          placeholder="Add tags"
          onKeyDown={e => {
            if (e.key === 'Enter' && e.currentTarget.value.trim() !== '') {
              addTag(e.currentTarget.value.trim());
              e.currentTarget.value = '';
            }
          }}
        />

        <div className="text-[0.8rem] text-gray-500">
          Add tags to make projects easier to find. To add tags, separate them
          with commas and press Enter.
        </div>
      </FieldSet>
    </div>
  );
};
