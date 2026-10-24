import {
  Button as CarbonButton,
  ButtonSkeleton as CarbonButtonSkeleton,
  IconButton as CarbonIconButton,
} from '@carbon/react';
import type { ButtonSize } from '@carbon/react/es/components/Button/Button';
import { Spinner } from '@/app/components/ui/feedback/loaders/spinner';
import { Text } from '@/app/components/ui/primitives/text';
import { cn } from '@/utils';
import type {
  ComponentProps,
  ElementType,
  MouseEventHandler,
  ReactNode,
} from 'react';

// Types

type ButtonKind =
  | 'primary'
  | 'secondary'
  | 'tertiary'
  | 'ghost'
  | 'danger'
  | 'danger--tertiary'
  | 'danger--ghost';

type IconButtonKind = 'primary' | 'secondary' | 'ghost' | 'tertiary';
type IconButtonSize = 'sm' | 'md' | 'lg';

export interface CarbonButtonProps {
  children?: ReactNode;
  className?: string;
  disabled?: boolean;
  isLoading?: boolean;
  size: ButtonSize;
  type?: 'button' | 'submit' | 'reset';
  onClick?: MouseEventHandler<HTMLButtonElement>;
  renderIcon?: ComponentProps<typeof CarbonButton>['renderIcon'];
  iconDescription?: string;
  hasIconOnly?: boolean;
  as?: ElementType;
  href?: string;
  target?: string;
  rel?: string;
  tabIndex?: number;
  tooltipPosition?: 'top' | 'bottom' | 'left' | 'right';
}

interface IconButtonWithBadgeProps extends CarbonButtonProps {
  badgeCount?: number;
}

interface ButtonSkeletonProps {
  className?: string;
  size: ButtonSize;
  small?: boolean;
  href?: string;
}

// Internal helper

function CarbonBtn({
  kind,
  isLoading,
  size,
  type = 'button',
  disabled,
  children,
  className,
  renderIcon,
  iconDescription,
  hasIconOnly,
  ...rest
}: CarbonButtonProps & { kind: ButtonKind }) {
  if (isLoading) {
    return (
      <CarbonButtonSkeleton
        size={size}
        small={hasIconOnly}
        className={cn('button', className)}
      />
    );
  }

  return (
    <CarbonButton
      {...rest}
      type={type}
      kind={kind}
      size={size}
      disabled={disabled}
      className={cn('button', className)}
      renderIcon={renderIcon}
      iconDescription={iconDescription}
      hasIconOnly={hasIconOnly}
    >
      {children}
    </CarbonButton>
  );
}

// Default / Primary

/** Carbon Primary button: the main CTA on a page. */
export function PrimaryButton({
  children,
  className,
  ...props
}: CarbonButtonProps) {
  return (
    <CarbonBtn
      kind="primary"
      className={cn('app-primary-button', className)}
      {...props}
    >
      <Text variant="body-compact-01" className="text-inherit">
        {children}
      </Text>
    </CarbonBtn>
  );
}

// Secondary

/** Carbon Secondary button: pairs with Primary as a less prominent action. */
export function SecondaryButton(props: CarbonButtonProps) {
  return <CarbonBtn kind="secondary" {...props} />;
}

// Tertiary

/** Carbon Tertiary button: bordered, transparent bg, fills on hover. */
export function TertiaryButton(props: CarbonButtonProps) {
  return <CarbonBtn kind="tertiary" {...props} />;
}

// Ghost

/** Carbon Ghost button: no border or bg, text only. Low emphasis actions. */
export function GhostButton(props: CarbonButtonProps) {
  return <CarbonBtn kind="ghost" {...props} />;
}

// Danger

/** Carbon Danger button: red filled. Destructive/irreversible actions. */
export function DangerButton(props: CarbonButtonProps) {
  return <CarbonBtn kind="danger" {...props} />;
}

// Danger Tertiary

/** Carbon Danger Tertiary: red bordered, transparent bg, fills red on hover. */
export function DangerTertiaryButton(props: CarbonButtonProps) {
  return <CarbonBtn kind="danger--tertiary" {...props} />;
}

// Danger Ghost

/** Carbon Danger Ghost: red text only, no border or bg. */
export function DangerGhostButton(props: CarbonButtonProps) {
  return <CarbonBtn kind="danger--ghost" {...props} />;
}

// Icon Button

/** Carbon Icon-only button: square button with just an icon. */
export function IconOnlyButton({
  isLoading,
  size,
  disabled,
  className,
  renderIcon,
  iconDescription = 'Action',
  kind,
  onClick,
  tooltipPosition,
  ...rest
}: Omit<CarbonButtonProps, 'size'> & {
  kind: IconButtonKind;
  size: IconButtonSize;
}) {
  return (
    <CarbonIconButton
      label={iconDescription}
      size={size}
      disabled={Boolean(disabled || isLoading)}
      className={cn(className)}
      kind={kind}
      onClick={onClick}
      align={tooltipPosition}
      {...rest}
    >
      {isLoading ? (
        <Spinner className="w-4 h-4" size="xs" />
      ) : renderIcon ? (
        (() => {
          const Icon = renderIcon as ElementType;
          return <Icon className="h-4 w-4" strokeWidth={1.5} />;
        })()
      ) : null}
    </CarbonIconButton>
  );
}

// Icon Button With Badge

/** Carbon Icon button with a notification badge count overlay. */
export function IconButtonWithBadge({
  badgeCount,
  isLoading,
  size,
  disabled,
  className,
  renderIcon,
  iconDescription = 'Action',
  kind,
  onClick,
  ...rest
}: Omit<IconButtonWithBadgeProps, 'size'> & {
  kind: IconButtonKind;
  size: IconButtonSize;
}) {
  return (
    <span className="relative inline-flex">
      <CarbonIconButton
        label={iconDescription}
        size={size}
        disabled={Boolean(disabled || isLoading)}
        className={cn(className)}
        kind={kind}
        onClick={onClick}
        {...rest}
      >
        {isLoading ? (
          <Spinner className="w-4 h-4" size="xs" />
        ) : renderIcon ? (
          (() => {
            const Icon = renderIcon as ElementType;
            return <Icon className="h-4 w-4" strokeWidth={1.5} />;
          })()
        ) : null}
      </CarbonIconButton>
      {badgeCount != null && badgeCount > 0 && (
        <span
          className={cn(
            'absolute -top-1 -right-1 flex items-center justify-center',
            'min-w-[18px] h-[18px] px-1 rounded-full',
            'bg-red-600 text-white text-[10px] font-semibold leading-none',
            'pointer-events-none',
          )}
        >
          {badgeCount > 99 ? '99+' : badgeCount}
        </span>
      )}
    </span>
  );
}

// Labeled Icon Button

/** Carbon button with a leading icon and a text label. */
export function LabeledIconButton({
  isLoading,
  size,
  type = 'button',
  disabled,
  children,
  className,
  renderIcon,
  iconDescription,
  kind,
  ...rest
}: CarbonButtonProps & { kind: ButtonKind }) {
  return (
    <CarbonButton
      {...rest}
      type={type}
      kind={kind}
      size={size}
      disabled={Boolean(disabled || isLoading)}
      className={cn('button', className)}
      renderIcon={renderIcon}
      iconDescription={iconDescription}
    >
      {isLoading ? <Spinner className="w-4 h-4" size="xs" /> : children}
    </CarbonButton>
  );
}

// Skeleton

/** Carbon ButtonSkeleton: placeholder shimmer while content loads. */
export function ButtonSkeleton({
  className,
  size,
  small,
  href,
}: ButtonSkeletonProps) {
  return (
    <CarbonButtonSkeleton
      className={cn('button', className)}
      size={size}
      small={small}
      href={href}
    />
  );
}
