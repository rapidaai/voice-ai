/**
 * Copyright IBM Corp. 2024, 2025
 * Licensed under the Apache-2.0 license.
 */

import { User } from '@carbon/icons-react';
import { Tooltip, usePrefix, type PopoverAlignment } from '@carbon/react';
import {
  forwardRef,
  type ElementType,
  type HTMLAttributes,
  type ReactNode,
} from 'react';
import { cn } from '@/utils';

export type UserAvatarBackgroundColor =
  | 'order-1-cyan'
  | 'order-2-gray'
  | 'order-3-green'
  | 'order-4-magenta'
  | 'order-5-purple'
  | 'order-6-teal'
  | 'order-7-cyan'
  | 'order-8-gray'
  | 'order-9-green'
  | 'order-10-magenta'
  | 'order-11-purple'
  | 'order-12-teal';

export type UserAvatarSize = 'xl' | 'lg' | 'md' | 'sm';

type UserAvatarImageProps =
  | { image: string; imageDescription: string }
  | { image?: never; imageDescription?: never };

export type UserAvatarProps = Omit<
  HTMLAttributes<HTMLDivElement>,
  'children'
> & {
  backgroundColor?: UserAvatarBackgroundColor;
  name?: string;
  renderIcon?: ElementType;
  size?: UserAvatarSize;
  tooltipAlignment?: PopoverAlignment;
  tooltipText?: ReactNode;
} & UserAvatarImageProps;

const blockClass = 'cds--user-avatar';

const iconSizes: Record<UserAvatarSize, 16 | 20 | 24 | 32> = {
  sm: 16,
  md: 20,
  lg: 24,
  xl: 32,
};

export const UserAvatar = forwardRef<HTMLDivElement, UserAvatarProps>(
  (
    {
      backgroundColor = 'order-1-cyan',
      className,
      image,
      imageDescription,
      name,
      renderIcon: RenderIcon,
      size = 'md',
      tooltipAlignment = 'bottom',
      tooltipText,
      ...rest
    },
    ref,
  ) => {
    const carbonPrefix = usePrefix();
    const parts = name?.split(' ') ?? [];
    const firstName = parts[0] ?? '';
    const firstCharacter = firstName.charAt(0).toUpperCase();
    const secondCharacter = firstName.charAt(1).toUpperCase();
    const lastCharacter = parts.at(-1)?.charAt(0).toUpperCase();
    const initials =
      parts.length === 1
        ? firstCharacter + secondCharacter
        : firstCharacter + (lastCharacter ?? '');
    const iconProps = { size: iconSizes[size] };

    let content: ReactNode = <User {...iconProps} />;
    if (image) {
      content = (
        <img
          alt={imageDescription}
          className={`${blockClass}__photo ${blockClass}__photo--${size}`}
          src={image}
        />
      );
    } else if (RenderIcon) {
      content = <RenderIcon {...iconProps} />;
    } else if (name) {
      content = initials;
    }

    const avatar = (
      <div
        {...rest}
        className={cn(
          blockClass,
          className,
          `${blockClass}--${backgroundColor}`,
          `${blockClass}--${size}`,
        )}
        ref={ref}
      >
        {content}
      </div>
    );

    if (!tooltipText) return avatar;

    return (
      <Tooltip
        align={tooltipAlignment}
        className={`${blockClass}__tooltip ${carbonPrefix}--icon-tooltip`}
        label={tooltipText}
      >
        <button className={`${blockClass}__tooltip-trigger`} type="button">
          {avatar}
        </button>
      </Tooltip>
    );
  },
);

UserAvatar.displayName = 'UserAvatar';
