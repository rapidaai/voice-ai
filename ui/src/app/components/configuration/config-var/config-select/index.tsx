import type { FC } from 'react';
import React from 'react';
import { ReactSortable } from 'react-sortablejs';
import { Input } from '@/app/components/Form/Input';
import { cn } from '@/styles/media';
import { IBlueBorderPlusButton } from '@/app/components/Form/Button';
import { DeleteButton } from '@/app/components/Form/Button/delete-button';
import { GripVertical } from 'lucide-react';

export type Options = string[];
export type IConfigSelectProps = {
  placeholder?: string;
  label?: string;
  options: Options;
  onChange: (options: Options) => void;
};

const ConfigSelect: FC<IConfigSelectProps> = ({
  placeholder,
  label = 'Add options',
  options,
  onChange,
}) => {
  const optionList = options.map((content, index) => {
    return {
      id: index,
      name: content,
    };
  });

  return (
    <div>
      {options.length > 0 && (
        <div className="mb-2.5">
          <ReactSortable
            className="space-y-1"
            list={optionList}
            setList={list => onChange(list.map(item => item.name))}
            handle=".handle"
            ghostClass="opacity-10"
            animation={150}
          >
            {options.map((o, index) => (
              <div
                className={cn(
                  'relative flex rounded-[2px]',
                  'border border-gray-300 dark:border-gray-800 rounded-[2px]',
                  'focus-within:border-blue-600!',
                )}
                key={index}
              >
                <div
                  className={cn(
                    'handle flex items-center justify-center px-2 cursor-grab',
                    'rounded-[2px]',
                  )}
                >
                  <GripVertical className="w-4 h-4" />
                </div>
                <Input
                  key={index}
                  type="input"
                  value={o || ''}
                  className="border-none group form-input"
                  placeholder={placeholder}
                  onChange={e => {
                    const value = e.target.value;
                    onChange(
                      options.map((item, i) => {
                        if (index === i) return value;

                        return item;
                      }),
                    );
                  }}
                />

                <div className="absolute top-1 right-1">
                  <DeleteButton
                    onClick={() => {
                      onChange(options.filter((_, i) => index !== i));
                    }}
                  ></DeleteButton>
                </div>
              </div>
            ))}
          </ReactSortable>
        </div>
      )}

      <IBlueBorderPlusButton
        onClick={() => {
          onChange([...options, '']);
        }}
      >
        {label}
      </IBlueBorderPlusButton>
    </div>
  );
};

export default React.memo(ConfigSelect);
