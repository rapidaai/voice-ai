import React from 'react';
import { Globe, Bug, Code, Coffee, Phone } from 'lucide-react';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { WhatsappIcon } from '@/app/components/Icon/whatsapp';

const sourceConfig = {
  'web-plugin': {
    bgColor: 'bg-indigo-100 dark:bg-indigo-900/30',
    textColor: 'text-indigo-700 dark:text-indigo-300',
    borderColor: 'border-indigo-200 dark:border-indigo-900/20',
    icon: <Globe className="w-4 h-4" />,
    label: 'Web Plugin',
  },
  debugger: {
    bgColor: 'bg-yellow-50 dark:bg-yellow-900/10',
    textColor: 'text-yellow-700 dark:text-yellow-700',
    borderColor: 'border-yellow-300 dark:border-yellow-900/20',
    icon: <Bug className="w-4 h-4" />,
    label: 'Debugger',
  },
  'rapida-app': {
    bgColor: 'bg-sky-100 dark:bg-sky-900/30',
    textColor: 'text-sky-700 dark:text-sky-300',
    borderColor: 'border-sky-200 dark:border-sky-900/20',
    icon: <RapidaIcon className="w-4 h-4" />,
    label: 'Rapida App',
  },
  'node-sdk': {
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-700 dark:text-green-300',
    borderColor: 'border-green-200 dark:border-green-900/20',
    icon: <Code className="w-4 h-4" />,
    label: 'Node SDK',
  },
  'go-sdk': {
    bgColor: 'bg-cyan-100 dark:bg-cyan-900/30',
    textColor: 'text-cyan-700 dark:text-cyan-300',
    borderColor: 'border-cyan-200 dark:border-cyan-900/20',
    icon: <Code className="w-4 h-4" />,
    label: 'Go SDK',
  },
  'typescript-sdk': {
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    textColor: 'text-blue-700 dark:text-blue-300',
    borderColor: 'border-blue-200 dark:border-blue-900/20',
    icon: <Code className="w-4 h-4" />,
    label: 'TypeScript SDK',
  },
  'java-sdk': {
    bgColor: 'bg-amber-100 dark:bg-amber-900/30',
    textColor: 'text-amber-700 dark:text-amber-300',
    borderColor: 'border-amber-200 dark:border-amber-900/20',
    icon: <Coffee className="w-4 h-4" />,
    label: 'Java SDK',
  },
  'php-sdk': {
    bgColor: 'bg-purple-100 dark:bg-purple-900/30',
    textColor: 'text-purple-700 dark:text-purple-300',
    borderColor: 'border-purple-200 dark:border-purple-900/20',
    icon: <Code className="w-4 h-4" />,
    label: 'PHP SDK',
  },
  'rust-sdk': {
    bgColor: 'bg-orange-100 dark:bg-orange-900/30',
    textColor: 'text-orange-700 dark:text-orange-300',
    borderColor: 'border-orange-200 dark:border-orange-900/20',
    icon: <Code className="w-4 h-4" />,
    label: 'Rust SDK',
  },
  'python-sdk': {
    bgColor: 'bg-yellow-100 dark:bg-yellow-900/30',
    textColor: 'text-yellow-700 dark:text-yellow-300',
    borderColor: 'border-yellow-200 dark:border-yellow-900/20',
    icon: <Code className="w-4 h-4" />,
    label: 'Python SDK',
  },
  'react-sdk': {
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    textColor: 'text-blue-700 dark:text-blue-300',
    borderColor: 'border-blue-200 dark:border-blue-900/20',
    icon: <Code className="w-4 h-4" />,
    label: 'React SDK',
  },
  'twilio-call': {
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    textColor: 'text-green-400 dark:text-green-600',
    borderColor: 'border-green-200 dark:border-green-900/20',
    icon: <Phone className="w-4 h-4" />,
    label: 'Phone',
  },
  'twilio-whatsapp': {
    bgColor: 'bg-emerald-100 dark:bg-emerald-900/30',
    textColor: 'text-emerald-700 dark:text-emerald-300',
    borderColor: 'border-emerald-200 dark:border-emerald-900/20',
    icon: <WhatsappIcon className="w-4 h-4" />,
    label: 'WhatsApp',
  },
};

interface SourceSelectorProps {
  selectedSource?: string;
  selectSource: (source: string) => void;
}

const SourceSelector: React.FC<SourceSelectorProps> = ({
  selectedSource,
  selectSource,
}) => {
  return (
    <div className="flex flex-col space-y-2">
      {Object.entries(sourceConfig).map(([value, config]) => (
        <button
          key={value}
          onClick={() => selectSource(value)}
          className={`w-full flex items-center gap-3 p-3 border transition-all ${
            selectedSource === value
              ? `${config.bgColor} ${config.borderColor} ${config.textColor} border-l-4`
              : 'bg-light-background dark:bg-gray-950'
          }`}
        >
          {config.icon}
          <span className="text-sm font-medium">{config.label}</span>
          {selectedSource === value && (
            <div
              className={`ml-auto w-2 h-2 ${config.textColor.replace('text', 'bg')} rounded-[2px]`}
            />
          )}
        </button>
      ))}
    </div>
  );
};

export default SourceSelector;
