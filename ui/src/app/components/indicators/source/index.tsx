import React from 'react';
import { Tag, Tooltip } from '@carbon/react';
import {
  Phone,
  Code,
  Globe,
  Debug,
  Application,
  LogoReact,
  LogoPython,
} from '@carbon/icons-react';

interface SourceIndicatorProps {
  source: string;
  withLabel?: boolean;
}

const sourceConfig: Record<
  string,
  { tagType: string; icon: React.ReactNode; label: string }
> = {
  'phone-call': { tagType: 'green', icon: <Phone size={16} />, label: 'Phone' },
  sdk: { tagType: 'warm-gray', icon: <Code size={16} />, label: 'SDK' },
  'web-plugin': {
    tagType: 'purple',
    icon: <Globe size={16} />,
    label: 'Web Plugin',
  },
  debugger: {
    tagType: 'warm-gray',
    icon: <Debug size={16} />,
    label: 'Debugger',
  },
  'rapida-app': {
    tagType: 'blue',
    icon: <Application size={16} />,
    label: 'App',
  },
  'node-sdk': { tagType: 'green', icon: <Code size={16} />, label: 'Node SDK' },
  'go-sdk': { tagType: 'cyan', icon: <Code size={16} />, label: 'Go SDK' },
  'typescript-sdk': {
    tagType: 'blue',
    icon: <Code size={16} />,
    label: 'TypeScript SDK',
  },
  'java-sdk': {
    tagType: 'warm-gray',
    icon: <Code size={16} />,
    label: 'Java SDK',
  },
  'php-sdk': { tagType: 'purple', icon: <Code size={16} />, label: 'PHP SDK' },
  'rust-sdk': {
    tagType: 'warm-gray',
    icon: <Code size={16} />,
    label: 'Rust SDK',
  },
  'python-sdk': {
    tagType: 'warm-gray',
    icon: <LogoPython size={16} />,
    label: 'Python SDK',
  },
  'react-sdk': {
    tagType: 'blue',
    icon: <LogoReact size={16} />,
    label: 'React SDK',
  },
  'twilio-call': {
    tagType: 'green',
    icon: <Phone size={16} />,
    label: 'Phone',
  },
  'exotel-call': {
    tagType: 'green',
    icon: <Phone size={16} />,
    label: 'Phone',
  },
};

const defaultConfig = {
  tagType: 'gray',
  icon: <Code size={16} />,
  label: 'Unknown',
};

export const SourceIndicator: React.FC<SourceIndicatorProps> = ({
  source,
  withLabel = true,
}) => {
  const config = sourceConfig[source] || defaultConfig;
  if (!withLabel) {
    return (
      <Tooltip label={config.label} align="bottom">
        <Tag
          size="md"
          type={config.tagType as any}
          className="flex items-center"
        >
          <span className="flex items-center justify-center leading-none [&>svg]:block">
            {config.icon}
          </span>
        </Tag>
      </Tooltip>
    );
  }

  return (
    <Tag
      size="md"
      type={config.tagType as any}
      className="inline-flex items-center"
    >
      <span className="flex items-center gap-1.5 leading-none [&>svg]:block">
        {config.icon}
        {config.label}
      </span>
    </Tag>
  );
};

export default SourceIndicator;
