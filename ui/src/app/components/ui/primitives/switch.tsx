import { memo, useId } from 'react';
import { Toggle } from '@carbon/react';
import { cn } from '@/utils';

export interface SwitchProps {
  enable: boolean;
  setEnable: (e: boolean) => void;
  id?: string;
  name?: string;
  label?: string;
}

function SwitchComponent({ enable, setEnable, id, name, label }: SwitchProps) {
  const generatedId = useId();
  const toggleId = id ?? `${name ?? 'switch'}-${generatedId}`;

  return (
    <Toggle
      id={toggleId}
      name={name}
      labelText={label ?? name ?? 'Switch'}
      hideLabel
      size="sm"
      toggled={enable}
      onToggle={setEnable}
    />
  );
}

export const Switch = memo(SwitchComponent);

export interface SwitchWithLabelProps {
  enable: boolean;
  setEnable: (e: boolean) => void;
  id?: string;
  label?: string;
  className?: string;
}

function SwitchWithLabelComponent({
  enable,
  setEnable,
  id,
  label,
  className,
}: SwitchWithLabelProps) {
  return (
    <div
      className={cn(
        'w-full',
        'form-input',
        'min-h-10',
        'text-foreground ',
        'border-b border-border-subtle ',
        'bg-surface ',
        'px-4 py-2 flex justify-between items-center w-full gap-4',
        className,
      )}
    >
      <span>{label}</span>
      <Switch enable={enable} setEnable={setEnable} id={id} label={label} />
    </div>
  );
}

export const SwitchWithLabel = memo(SwitchWithLabelComponent);
