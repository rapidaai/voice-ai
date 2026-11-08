import { memo } from 'react';
import { ReactSortable } from 'react-sortablejs';
import { TextInput } from '@/app/components/ui/primitives';
import { TertiaryButton } from '@/app/components/ui/primitives';
import { Add, TrashCan, Draggable } from '@carbon/icons-react';
import { Button } from '@carbon/react';
import {
  TableForm,
  TableFormCell,
  TableFormRow,
} from '@/app/components/ui/table';

export type Options = string[];
export type IConfigSelectProps = {
  placeholder?: string;
  label?: string;
  helperText?: string;
  options: Options;
  onChange: (options: Options) => void;
};

function ConfigSelect({
  placeholder,
  label = 'Add option',
  helperText,
  options,
  onChange,
}: IConfigSelectProps) {
  const optionList = options.map((content, index) => ({
    id: index,
    name: content,
  }));

  return (
    <div>
      {options.length > 0 && (
        <TableForm>
          <ReactSortable
            tag="tbody"
            list={optionList}
            setList={list => onChange(list.map(item => item.name))}
            handle=".handle"
            ghostClass="opacity-60"
            animation={150}
          >
            {options.map((option, index) => (
              <TableFormRow key={optionList[index].id}>
                <TableFormCell className="w-8 text-center">
                  <span className="handle cursor-grab inline-flex items-center justify-center p-2 text-muted">
                    <Draggable size={16} />
                  </span>
                </TableFormCell>
                <TableFormCell>
                  <TextInput
                    id={`suggestion-${index}`}
                    labelText=""
                    hideLabel
                    value={option || ''}
                    placeholder={placeholder}
                    size="md"
                    onChange={e => {
                      onChange(
                        options.map((item, i) =>
                          i === index ? e.target.value : item,
                        ),
                      );
                    }}
                  />
                </TableFormCell>
                <TableFormCell className="w-10 text-center" divider={false}>
                  <Button
                    hasIconOnly
                    renderIcon={TrashCan}
                    iconDescription="Remove"
                    kind="ghost"
                    size="sm"
                    onClick={() =>
                      onChange(options.filter((_, i) => i !== index))
                    }
                  />
                </TableFormCell>
              </TableFormRow>
            ))}
          </ReactSortable>
        </TableForm>
      )}
      <div className="pt-2">
        <TertiaryButton
          size="md"
          renderIcon={Add}
          onClick={() => onChange([...options, ''])}
          className="!w-full !max-w-none"
        >
          {label}
        </TertiaryButton>
      </div>
      {helperText && <p className="text-xs text-muted mt-2">{helperText}</p>}
    </div>
  );
}

export default memo(ConfigSelect);
