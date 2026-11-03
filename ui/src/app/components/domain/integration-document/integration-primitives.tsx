import { CodeSnippet, Tag } from '@carbon/react';
import type { ReactNode } from 'react';

interface IntegrationCodeBlockProps {
  code: string;
  language: string;
}

export function IntegrationCodeBlock({
  code,
  language,
}: IntegrationCodeBlockProps) {
  return (
    <div className="w-full" data-language={language}>
      <CodeSnippet
        aria-label={`${language} code snippet`}
        copyButtonDescription={`Copy ${language} snippet`}
        copyText={code}
        feedback="Copied"
        maxCollapsedNumberOfRows={20}
        maxExpandedNumberOfRows={40}
        minCollapsedNumberOfRows={3}
        type="multi"
        wrapText
      >
        {code}
      </CodeSnippet>
    </div>
  );
}

interface IntegrationStepProps {
  number: number;
  title: string;
  description: string;
  children?: ReactNode;
}

export function IntegrationStep({
  number,
  title,
  description,
  children,
}: IntegrationStepProps) {
  return (
    <section className="space-y-3 border-b border-gray-200 px-4 py-5 last:border-b-0 dark:border-gray-800">
      <div className="flex items-center gap-2.5">
        <Tag size="sm" type="blue">
          Step {number}
        </Tag>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {title}
        </h3>
      </div>
      <p className="text-sm text-gray-500 dark:text-gray-400">{description}</p>
      {children}
    </section>
  );
}
