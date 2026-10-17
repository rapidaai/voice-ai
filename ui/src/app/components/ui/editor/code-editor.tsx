import type { ReactNode } from 'react';
import { useRef } from 'react';
import { useBoolean } from 'ahooks';
import { cn } from '@/utils';
import { useToggleExpend } from '@/hooks/use-toggle-expend';
import { JsonEditor } from '@/app/components/ui/editor/json-editor';
import { Copy, Checkmark, Maximize, Minimize } from '@carbon/icons-react';
import { Button } from '@carbon/react';

type CodeEditorProps = {
  placeholder: string;
  value: string;
  onChange: (value: string) => void;
  className?: string;
  labelText?: ReactNode;
  helperText?: ReactNode;
};

export function CodeEditor({
  placeholder,
  value,
  onChange,
  className,
  labelText,
  helperText,
}: CodeEditorProps) {
  const ref = useRef<HTMLDivElement>(null);
  const { isExpand, setIsExpand } = useToggleExpend(ref);
  const [isChecked, { setTrue: setChecked, setFalse: setUnCheck }] =
    useBoolean(false);

  const handlePromptChange = (newValue: string) => {
    if (value === newValue) return;
    onChange(newValue);
  };

  const copyItem = (item: string) => {
    setChecked();
    navigator.clipboard.writeText(item);
    setTimeout(() => setUnCheck(), 4000);
  };

  return (
    <div
      ref={ref}
      className={cn(
        'cds--form-item w-full',
        isExpand &&
          'fixed inset-0 z-50 bg-[var(--cds-background)] flex flex-col',
      )}
    >
      {labelText && !isExpand && (
        <label className="cds--label">{labelText}</label>
      )}
      <div
        className={cn(
          'relative group w-full',
          isExpand && 'flex-1 min-h-0',
        )}
      >
        <div className="flex items-center absolute right-0 top-0 z-20 opacity-0 group-hover:opacity-100 transition-opacity">
          <Button
            hasIconOnly
            renderIcon={isChecked ? Checkmark : Copy}
            iconDescription="Copy"
            kind="ghost"
            size="sm"
            onClick={() => copyItem(value)}
            tabIndex={-1}
          />
          <Button
            hasIconOnly
            renderIcon={isExpand ? Minimize : Maximize}
            iconDescription={isExpand ? 'Minimize' : 'Maximize'}
            kind="ghost"
            size="sm"
            onClick={() => setIsExpand(!isExpand)}
            tabIndex={-1}
          />
        </div>

        <JsonEditor
          className={cn('w-full h-full', className)}
          height={isExpand ? '100%' : '200px'}
          placeholder={placeholder}
          value={value}
          onChange={handlePromptChange}
        />
      </div>
      {helperText && <div className="cds--form__helper-text">{helperText}</div>}
    </div>
  );
}
