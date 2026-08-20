import { Helmet as HM } from 'react-helmet-async';
import { useTheme } from '@/theme/theme-provider';

/**
 *
 */
interface HelmetProps {
  title?: string;
  meta?: { name: string; content: string }[];
}

/**
 *
 * @param props
 * @returns
 */
export function Helmet(props: HelmetProps) {
  const { resolvedMode, theme } = useTheme();
  const title = props.title
    ? `${props.title} - ${theme.brand.name}`
    : theme.brand.name;
  const favicon =
    theme.brand.logos?.compact[resolvedMode] ?? theme.brand.favicon;

  return (
    <HM>
      <title>{title}</title>
      <meta name="application-name" content={theme.brand.name} />
      {favicon && <link rel="icon" href={favicon} />}
      {props.meta &&
        props.meta.map((mt, idx) => {
          return (
            <meta key={`meta_${idx}`} name={mt.name} content={mt.content} />
          );
        })}
    </HM>
  );
}
