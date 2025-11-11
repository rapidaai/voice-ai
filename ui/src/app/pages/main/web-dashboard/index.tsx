import React from 'react';
import { MessageSquare, TestTube } from 'lucide-react';
import { ClickableCard } from '@/app/components/base/cards';
import { useCurrentCredential } from '@/hooks/use-credential';
import { KnowledgeIcon } from '@/app/components/Icon/knowledge';
import { EndpointIcon } from '@/app/components/Icon/Endpoint';
import { DeploymentIcon } from '@/app/components/Icon/Deployment';
import { AssistantIcon } from '@/app/components/Icon/Assistant';
import { ToolIcon } from '@/app/components/Icon/tool';
import { ModelIcon } from '@/app/components/Icon/Model';

export const HomePage = () => {
  const coreFeatures = [
    {
      icon: KnowledgeIcon,
      title: 'Knowledge Hub',
      description:
        'Unified repository for documents, training data, and AI knowledge management — the foundation of contextual intelligence.',
      color: 'bg-blue-500',
      route: '/knowledge',
    },
    {
      icon: MessageSquare,
      title: 'Conversational AI',
      description:
        'Context-aware, LLM-powered chat experiences that understand user intent and deliver accurate responses.',
      color: 'bg-yellow-500',
      route: '/deployment/assistant',
    },
    {
      icon: AssistantIcon,
      title: 'AI Assistants',
      description:
        'Deploy domain-specific AI agents with custom skills, workflows, and multi-step reasoning.',
      color: 'bg-green-500',
      route: '/deployment/assistant',
    },
    {
      icon: DeploymentIcon,
      title: 'Seamless Deployment',
      description:
        'One-click agent deployment with built-in auto-scaling, version control, and monitoring.',
      color: 'bg-orange-500',
      route: '/deployment',
    },
    {
      icon: EndpointIcon,
      title: 'Governance & Endpoints',
      description:
        'Secure API endpoints with fine-grained governance, audit trails, and enterprise-grade access control.',
      color: 'bg-purple-500',
      route: '/deployment',
    },
    {
      icon: ModelIcon,
      title: 'Model Integration',
      description:
        'Bring your own model — support for OpenAI, Anthropic, and custom LLMs with fine-tuning capabilities.',
      color: 'bg-red-500',
      route: '/integration/models',
    },
    {
      icon: TestTube,
      title: 'Real-time Testing & Monitoring',
      description:
        'Instantly test AI agents and flows in a live sandbox to iterate faster and ship confidently.',
      color: 'bg-indigo-500',
      route: '/observability',
    },
    {
      icon: ToolIcon,
      title: 'External Integrations',
      description:
        'Connect effortlessly to CRMs, internal APIs, databases, and third-party tools to extend agent capabilities.',
      color: 'bg-teal-500',
      route: '/integration/tools',
    },
  ];
  const { user } = useCurrentCredential();
  return (
    <div className="flex-1 overflow-auto flex">
      <main className="max-w-7xl mx-auto my-auto px-6 py-8">
        <h1 className="text-4xl font-semibold mb-2">
          Good afternoon, {user?.name?.split(/\s+/)[0] || user?.name}{' '}
        </h1>
        <p className="text-lg mb-8">
          Build, deploy, and manage intelligent AI agents with enterprise-grade
          tools
        </p>

        {/* Core Platform Features */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {coreFeatures.map((feature, index) => (
            <ClickableCard
              to={feature.route}
              key={index}
              className="transition-all duration-200 cursor-pointer group p-8 hover:shadow-lg border-b-2 hover:border-blue-600 h-full"
            >
              <div className="flex flex-col space-y-4">
                <div className="flex items-center justify-between">
                  <div
                    className={`w-12 h-12 ${feature.color} flex items-center justify-center group-hover:scale-110 transition-transform`}
                  >
                    <feature.icon
                      className="h-6 w-6 text-white"
                      strokeWidth={1.5}
                    />
                  </div>
                </div>
                <div>
                  <h3 className="text-lg font-semibold mb-2">
                    {feature.title}
                  </h3>
                  <p className="text-[0.95rem]">{feature.description}</p>
                </div>
              </div>
            </ClickableCard>
          ))}
        </div>
      </main>
    </div>
  );
};
