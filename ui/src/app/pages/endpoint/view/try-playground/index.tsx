import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels';
import { TryChatComplete } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/try-chat-complete';
import { Helmet } from '@/app/components/helmet';
import { Launch } from '@carbon/icons-react';
import { useDocumentationUrl } from '@/theme/documentation-url';

const overviewSections = [
  {
    title: 'Post conversation analysis',
    description:
      'Use this endpoint after an assistant call to run structured LLM analysis.',
    steps: [
      'Open Assistants',
      'Select an assistant',
      'Go to Manage',
      'Add this endpoint under Analysis',
    ],
  },
  {
    title: 'Tool call from LLM',
    description:
      'Expose this endpoint as an LLM tool for targeted runtime calls.',
    steps: [
      'Open Assistants',
      'Select an assistant',
      'Go to Manage',
      'Add this endpoint under Tool Call - LLM Call',
    ],
  },
];

function EndpointOverviewSection(props: {
  title: string;
  description: string;
  steps: string[];
}) {
  return (
    <section className="border-b border-gray-200 bg-white px-5 py-5 dark:border-gray-800 dark:bg-gray-900">
      <div className="mb-4">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {props.title}
        </h3>
        <p className="mt-1 text-sm leading-6 text-gray-600 dark:text-gray-400">
          {props.description}
        </p>
      </div>
      <ol className="divide-y divide-gray-100 border-y border-gray-100 dark:divide-gray-800 dark:border-gray-800">
        {props.steps.map((step, index) => (
          <li
            key={step}
            className="grid grid-cols-[2rem_minmax(0,1fr)] items-center py-2.5 text-sm"
          >
            <span className="font-mono text-xs tabular-nums text-gray-500 dark:text-gray-400">
              {String(index + 1).padStart(2, '0')}
            </span>
            <span className="text-gray-800 dark:text-gray-200">{step}</span>
          </li>
        ))}
      </ol>
    </section>
  );
}

export function Playground(props: {
  currentEndpoint: Endpoint;
  currentEndpointProviderModel: EndpointProviderModel;
}) {
  const apiReferenceUrl = useDocumentationUrl('/api-reference/endpoint/invoke');

  return (
    <PanelGroup direction="horizontal" className="grow">
      <Helmet title={props.currentEndpoint.getName()} />
      <Panel
        defaultSize={40}
        minSize={30}
        className="flex flex-1 flex-col items-stretch overflow-y-auto! bg-gray-50 dark:bg-gray-950"
      >
        <section className="border-b border-gray-200 bg-white px-5 py-5 dark:border-gray-800 dark:bg-gray-900">
          <p className="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
            Endpoint overview
          </p>
          <h2 className="mt-2 truncate text-base font-semibold text-gray-900 dark:text-gray-100">
            {props.currentEndpoint.getName()}
          </h2>
          <p className="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-400">
            Invoke this endpoint directly or attach it to assistant workflows.
          </p>
          <a
            target="_blank"
            href={apiReferenceUrl}
            className="mt-4 inline-flex items-center gap-2 text-sm font-medium text-blue-600 hover:underline dark:text-blue-400"
            rel="noreferrer"
          >
            API reference
            <Launch size={16} />
          </a>
        </section>

        {overviewSections.map(section => (
          <EndpointOverviewSection key={section.title} {...section} />
        ))}
      </Panel>
      <PanelResizeHandle className="flex w-px! items-stretch bg-gray-200 hover:bg-blue-600 dark:bg-gray-800"></PanelResizeHandle>
      <Panel
        defaultSize={60}
        minSize={42}
        className="flex flex-col overflow-hidden"
      >
        <div className="flex flex-1 flex-col items-stretch overflow-hidden">
          <TryChatComplete
            currentEndpoint={props.currentEndpoint}
            endpointProviderModel={props.currentEndpointProviderModel}
          />
        </div>
      </Panel>
    </PanelGroup>
  );
}
