import { FC } from 'react';
import { EndpointIcon } from '@/app/components/Icon/Endpoint';
import {
  Card,
  CardDescription,
  CardTag,
  CardTitle,
  ClickableCard,
} from '@/app/components/base/cards';
import { SearchableDeployment } from '@rapidaai/react';
import { cn } from '@/styles/media';
import { BorderButton } from '@/app/components/Form/Button';
import { useNavigate } from 'react-router-dom';

/**
 *
 */
interface EndpointCardProps {
  deployment: {
    name: string;
    description: string;
    id: string;
    tags: string[];
    status: string;
  };
}

/**
 *
 * @param param0
 * @returns
 */
export const EndpointCard: FC<EndpointCardProps> = ({ deployment }) => {
  return (
    <ClickableCard
      to={`/deployment/endpoint/${deployment.id}`}
      className={cn('relative min-h-full p-4 md:p-5 rounded-xl')}
    >
      <div className="border border-gray-300/10 bg-gray-600/10 rounded-[2px] flex items-center justify-center shrink-0 h-10 w-10 p-1 mr-3">
        <EndpointIcon className="w-6 h-6 text-green-600" strokeWidth={1.5} />
      </div>
      <CardTitle
        className="text-[1.1rem] font-medium mt-4 opacity-80 hover:underline hover:text-blue-600"
        title={deployment.name}
      />
      <CardDescription
        className="mt-1 opacity-70 text-[.95rem] leading-6"
        description={deployment.description}
      />
      <div className="flex justify-end space-x-1.5 mt-6">
        <CardTag tags={deployment.tags} />
      </div>
    </ClickableCard>
  );
};

/**
 *
 * @param param0
 * @returns
 */
export const HubEndpointCard: FC<{ deployment: SearchableDeployment }> = ({
  deployment,
}) => {
  const navigator = useNavigate();
  return (
    <Card className={cn('relative min-h-full p-4 md:p-5 rounded-xl group')}>
      <div className="flex justify-between items-center">
        <div className="border border-gray-300/10 bg-gray-600/10 rounded-[2px] flex items-center justify-center shrink-0 h-10 w-10 p-1 mr-3">
          <EndpointIcon className="w-6 h-6 text-green-600" strokeWidth={1.5} />
        </div>
        <BorderButton
          className="h-8 text-sm space-x-1 relative border-[0.5px] w-fit invisible group-hover:visible hover:border-green-600! hover:text-green-600!"
          onClick={() => {
            navigator(
              `/deployment/endpoint/configure-endpoint/${deployment.getId()}`,
            );
          }}
        >
          <span className="block">Activate endpoint</span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth="1.5"
            stroke="currentColor"
            className="w-4 h-4"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M9.813 15.904 9 18.75l-.813-2.846a4.5 4.5 0 0 0-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 0 0 3.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 0 0 3.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 0 0-3.09 3.09ZM18.259 8.715 18 9.75l-.259-1.035a3.375 3.375 0 0 0-2.455-2.456L14.25 6l1.036-.259a3.375 3.375 0 0 0 2.455-2.456L18 2.25l.259 1.035a3.375 3.375 0 0 0 2.456 2.456L21.75 6l-1.035.259a3.375 3.375 0 0 0-2.456 2.456ZM16.894 20.567 16.5 21.75l-.394-1.183a2.25 2.25 0 0 0-1.423-1.423L13.5 18.75l1.183-.394a2.25 2.25 0 0 0 1.423-1.423l.394-1.183.394 1.183a2.25 2.25 0 0 0 1.423 1.423l1.183.394-1.183.394a2.25 2.25 0 0 0-1.423 1.423Z"
            />
          </svg>
        </BorderButton>
      </div>
      <CardTitle
        className="text-[1rem] font-semibold mt-4 opacity-80"
        title={deployment.getName()}
      />
      <CardDescription
        className="mt-1 opacity-70 text-[.95rem]! leading-6"
        description={deployment.getDescription()}
      />
      <div className="flex justify-end space-x-1.5 mt-6">
        <CardTag tags={deployment.getTagList()} />
      </div>
    </Card>
  );
};
