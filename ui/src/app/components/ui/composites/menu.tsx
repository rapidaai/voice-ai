import type { ElementType } from 'react';
import { cn } from '@/utils';
import { ChevronDown, OverflowMenuHorizontal } from '@carbon/icons-react';
import {
  OverflowMenu,
  OverflowMenuItem,
} from '@/app/components/ui/primitives/overflow-menu';

export interface OptionMenuOption {
  label: string;
  kind?: 'default' | 'danger';
  onActionClick: () => void;
  renderIcon?: ElementType;
}

interface OptionMenuProps {
  options: OptionMenuOption[];
  classNames?: string;
  activeClassName?: string;
}

function renderMenuItems(options: OptionMenuProps['options']) {
  return options.map((opt, idx) => {
    const isDanger = opt.kind === 'danger';

    return (
      <OverflowMenuItem
        key={`opt-menu-${idx}`}
        itemText={opt.label}
        isDelete={isDanger}
        hasDivider={isDanger}
        onClick={opt.onActionClick}
        renderIcon={opt.renderIcon}
      />
    );
  });
}

export function OptionMenu({ options, classNames }: OptionMenuProps) {
  return (
    <OverflowMenu
      size="sm"
      iconDescription="Menu"
      renderIcon={OverflowMenuHorizontal}
      direction="bottom"
      flipped
      className={cn(classNames)}
    >
      {renderMenuItems(options)}
    </OverflowMenu>
  );
}

export function CardOptionMenu({ options, classNames }: OptionMenuProps) {
  return (
    <OverflowMenu
      size="sm"
      iconDescription="Menu"
      renderIcon={ChevronDown}
      direction="bottom"
      flipped
      className={cn(classNames)}
    >
      {renderMenuItems(options)}
    </OverflowMenu>
  );
}
