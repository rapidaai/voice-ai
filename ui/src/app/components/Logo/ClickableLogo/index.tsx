import { cn } from '@/styles/media';

interface ClickableLogoProps extends React.HTMLAttributes<HTMLDivElement> {
  isBeta: boolean;
  isDev: boolean;
  alt: string;
  darkLogo: string;
  lightLogo: string;
}

export function ClickableLogo(props: ClickableLogoProps) {
  const { isBeta, isDev, darkLogo, lightLogo, ...divProps } = props;
  return (
    <div aria-label="logo" {...divProps}>
      <img
        src={darkLogo}
        className={cn('hidden dark:block h-full w-auto')}
        alt={props.alt}
      />
      <img
        src={lightLogo}
        className={cn('block dark:hidden  h-full w-auto')}
        alt={props.alt}
      />
    </div>
  );
}

ClickableLogo.defaultProps = {
  className: 'w-auto h-6 rounded-[2px]',
  isBeta: false,
  isDev: false,
  alt: 'RapidaAI Logo',
  darkLogo: '/images/logos/logo-04.png',
  lightLogo: '/images/logos/logo-02.png',
};
