import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels';
import { TryChatComplete } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/try-chat-complete';
import { InputGroup } from '@/app/components/input-group';
import { Helmet } from '@/app/components/Helmet';
import { BotIcon, Code, ExternalLink, PencilRuler } from 'lucide-react';
import { ToolIcon } from '@/app/components/Icon/tool';
export function Playground(props: {
  currentEndpoint: Endpoint;
  currentEndpointProviderModel: EndpointProviderModel;
}) {
  return (
    <PanelGroup direction="horizontal" className="grow">
      <Helmet title={props.currentEndpoint.getName()} />
      <Panel className="flex flex-1 flex-col items-stretch overflow-y-auto! bg-white dark:bg-gray-900">
        <div className="py-8 px-4">
          Endpoints allow you to integrate Large Language Models (LLMs) into
          your application, providing a powerful interface for AI-driven
          functionalities.
        </div>
        <InputGroup
          title={
            <div className="flex items-center space-x-2">
              <Code className="w-4 h-4" />
              <span>Integrate into your application</span>
            </div>
          }
        >
          <div className="border-t bg-white dark:bg-gray-900">
            <p className="text-gray-600 dark:text-gray-300 py-8 px-4">
              Integrate this endpoint directly into your application code using
              our SDK.
              <a
                target="_blank"
                href="https://doc.rapida.ai/api-reference/endpoint/invoke"
                className="h-7 flex items-center font-medium hover:underline ml-auto text-blue-600"
                rel="noreferrer"
              >
                Read documentation
                <ExternalLink
                  className="shrink-0 w-4 h-4 ml-1.5"
                  strokeWidth={1.5}
                />
              </a>
            </p>
          </div>
        </InputGroup>
        <InputGroup
          title={
            <div className="flex items-center space-x-2">
              <BotIcon className="w-5 h-5" />
              <span>Post conversation llm analysis</span>
            </div>
          }
        >
          <div className="border-t bg-white dark:bg-gray-900 py-8 px-4">
            <p className="text-base font-normal leading-relaxed pb-4">
              Enhance your assistant's capabilities with LLM analysis using this
              endpoint.
            </p>
            <ol className="space-y-3 text-base">
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  1
                </div>
                <div className="absolute left-3 top-6 w-0.5 h-full bg-gray-200 dark:bg-gray-950"></div>
                Navigate to the{' '}
                <span className="text-blue-600 font-medium">Assistants</span>{' '}
                page
              </li>
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  2
                </div>
                <div className="absolute left-3 top-6 w-0.5 h-full bg-gray-200 dark:bg-gray-950"></div>
                Select your assistant
              </li>
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  3
                </div>
                <div className="absolute left-3 top-6 w-0.5 h-full bg-gray-200 dark:bg-gray-950"></div>
                Go to the <span className="font-medium">'Mange'</span> tab
              </li>
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  4
                </div>
                Add this endpoint under{' '}
                <span className="font-medium">'Analysis'</span>
              </li>
            </ol>
          </div>
        </InputGroup>
        <InputGroup
          title={
            <div className="flex items-center space-x-2">
              <PencilRuler className="w-4 h-4" />
              <span>
                Tool calls to the LLM for targeted use of its capabilities
              </span>
            </div>
          }
        >
          <div className="border-t bg-white dark:bg-gray-900 py-8 px-4">
            <p className="text-base font-normal leading-relaxed pb-4">
              Enhance your assistant's capabilities with LLM analysis using this
              endpoint.
            </p>
            <ol className="space-y-3 text-base">
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  1
                </div>
                <div className="absolute left-3 top-6 w-0.5 h-full bg-gray-200 dark:bg-gray-950"></div>
                Navigate to the{' '}
                <span className="text-blue-600 font-medium">Assistants</span>{' '}
                page
              </li>
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  2
                </div>
                <div className="absolute left-3 top-6 w-0.5 h-full bg-gray-200 dark:bg-gray-950"></div>
                Select your assistant
              </li>
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  3
                </div>
                <div className="absolute left-3 top-6 w-0.5 h-full bg-gray-200 dark:bg-gray-950"></div>
                Go to the <span className="font-medium">'Mange'</span> tab
              </li>
              <li className="relative pl-10 leading-relaxed">
                <div className="absolute left-0 top-0 w-6 h-6 bg-gray-200 dark:bg-gray-950 rounded-full flex items-center justify-center font-semibold text-sm">
                  4
                </div>
                Add this endpoint under 'Tool Call - LLM Call'
              </li>
            </ol>
          </div>
        </InputGroup>
      </Panel>
      <PanelResizeHandle className="flex w-px! bg-gray-200 dark:bg-gray-800 hover:bg-blue-700 dark:hover:bg-blue-500 items-stretch"></PanelResizeHandle>
      <Panel className="flex flex-col overflow-y-auto">
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
