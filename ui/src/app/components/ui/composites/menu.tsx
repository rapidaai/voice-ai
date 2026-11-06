import { isValidElement, type ReactNode } from 'react';
import { cn } from '@/utils';
import { ChevronDown, OverflowMenuHorizontal } from '@carbon/icons-react';
import {
  OverflowMenu,
  OverflowMenuItem,
} from '@/app/components/ui/primitives/overflow-menu';

interface OptionMenuProps {
  options: { option: ReactNode; onActionClick: () => void }[];
  classNames?: string;
  activeClassName?: string;
}

const isDangerOption = (option: ReactNode) =>
  isValidElement<{ type?: string }>(option) &&
  option.type === OptionMenuItem &&
  option.props.type === 'danger';

function renderMenuItems(options: OptionMenuProps['options']) {
  return options.map((opt, idx) => {
    const isDanger = isDangerOption(opt.option);

    return (
      <OverflowMenuItem
        key={`opt-menu-${idx}`}
        itemText={opt.option}
        isDelete={isDanger}
        hasDivider={isDanger}
        onClick={opt.onActionClick}
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

export function OptionMenuItem(props: {
  type: 'danger' | 'info';
  children?: ReactNode;
}) {
  return props.type === 'danger' ? (
    <span className="text-support-error ">{props.children}</span>
  ) : (
    <></>
  );
}
