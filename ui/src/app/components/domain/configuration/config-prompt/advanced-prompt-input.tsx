import { memo, useRef } from 'react';
import { useBoolean } from 'ahooks';
import MessageTypeSelector from './message-type-selector';
import type { PromptRole } from '@/models/prompt';
import { GhostButton } from '@/app/components/ui/primitives';
import { cn } from '@/utils';
import { useToggleExpend } from '@/hooks/use-toggle-expend';
import PromptEditor from '@/app/components/domain/prompt-editor/prompt-editor';
import {
  TrashCan,
  Copy,
  Checkmark,
  Maximize,
  Minimize,
} from '@carbon/icons-react';
type PromptEditorProps = {
  type?: PromptRole;
  isChatMode: boolean;
  value: string;
  onTypeChange: (value: PromptRole) => void;
  onChange: (value: string) => void;
  canDelete: boolean;
  onDelete: () => void;
  className?: string;
  instanceId?: string;
  enableReservedVariableSuggestions?: boolean;
};

function AdvancedPromptInput({
  type,
  value,
  onChange,
  onTypeChange,
  canDelete,
  onDelete,
  className,
  enableReservedVariableSuggestions = false,
}: PromptEditorProps) {
  const ref = useRef<HTMLDivElement>(null);
  const { wrapClassName, isExpand, setIsExpand } = useToggleExpend(ref);

  const [isFocus, { setTrue: setFocus, setFalse: setBlur }] = useBoolean(false);

  const [isChecked, { setTrue: setChecked, setFalse: setUnCheck }] =
    useBoolean(false);

  const handlePromptChange = (newValue: string) => {
    if (value === newValue) return;
    onChange(newValue);
  };

  const copyItem = (item: string) => {
    setChecked();
    navigator.clipboard.writeText(item);
    setTimeout(() => {
      setUnCheck();
    }, 4000);
  };

  return (
    <div className={cn(wrapClassName)}>
      <div
        ref={ref}
        className={cn(
          'border-b border-border-subtle ',
          'transition-colors duration-100',
          'relative',
          'bg-field',
          isFocus && 'border-primary',
          'after:content-[""] after:absolute after:inset-0 after:pointer-events-none after:z-[1]',
          'after:outline-solid after:outline-2 after:outline-transparent after:outline-offset-[-2px]',
          'focus-within:after:outline-primary',
          isFocus && 'after:outline-primary!',
          isExpand ? 'h-full z-50' : '',
          className,
        )}
      >
        <div
          className={cn(
            'flex h-10 items-stretch bg-field',
            'border-b border-border-strong',
          )}
        >
          <div className="min-w-0 flex-1">
            <MessageTypeSelector value={type} onChange={onTypeChange} />
          </div>
          <div className="flex shrink-0 items-stretch">
            {canDelete && (
              <GhostButton
                hasIconOnly
                iconDescription="Delete prompt message"
                size="md"
                onClick={onDelete}
                tabIndex={-1}
                className="border border-l-border-subtle border-transparent transition-colors hover:border-support-error"
              >
                <TrashCan size={16} className="text-support-error" />
              </GhostButton>
            )}
            <GhostButton
              hasIconOnly
              iconDescription={isChecked ? 'Prompt copied' : 'Copy prompt'}
              size="md"
              tabIndex={-1}
              onClick={() => {
                copyItem(value);
              }}
              className="border border-l-border-subtle border-transparent transition-colors hover:border-support-info"
            >
              {isChecked ? (
                <Checkmark size={16} className="text-support-success" />
              ) : (
                <Copy size={16} />
              )}
            </GhostButton>
            <GhostButton
              hasIconOnly
              iconDescription={isExpand ? 'Collapse prompt' : 'Expand prompt'}
              size="md"
              tabIndex={-1}
              onClick={() => {
                setIsExpand(!isExpand);
              }}
              className="border border-l-border-subtle border-transparent transition-colors hover:border-support-info"
            >
              {isExpand ? <Minimize size={16} /> : <Maximize size={16} />}
            </GhostButton>
          </div>
        </div>

        <PromptEditor
          className={cn('min-h-[200px]', isExpand && 'h-screen')}
          placeholder={`Write your prompt here. enter {{variable}} to insert a variable.`}
          value={value}
          onFocus={setFocus}
          onChange={handlePromptChange}
          onBlur={setBlur}
          enableReservedVariableSuggestions={enableReservedVariableSuggestions}
        />
      </div>
    </div>
  );
}

export default memo(AdvancedPromptInput);
