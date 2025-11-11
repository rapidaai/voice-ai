import { FC } from 'react';
import { Endpoint } from '@rapidaai/react';
import { IBlueButton } from '@/app/components/Form/Button';
import { useNavigate } from 'react-router-dom';
import { useResourceRole } from '@/hooks/use-credential';
import { isOwnerResource } from '@/utils';
import { Plus } from 'lucide-react';

/**
 *
 * @param param0
 * @returns
 */
export const EndpointAction: FC<{ currentEndpoint: Endpoint }> = ({
  currentEndpoint,
}) => {
  /**
   * element
   */
  const role = useResourceRole(currentEndpoint);

  /**
   * element
   */
  if (isOwnerResource(role)) {
    return <OwnerAction currentEndpoint={currentEndpoint} />;
  }
  return <div />;
};

/**
 *
 * actions for the owner of assistant
 * @param param0
 * @returns
 */
const OwnerAction: FC<{ currentEndpoint: Endpoint }> = ({
  currentEndpoint,
}) => {
  /**
   * dom navigation
   */
  const navigate = useNavigate();
  return (
    <>
      {currentEndpoint != null && (
        <IBlueButton
          onClick={() => {
            navigate(
              `/deployment/endpoint/${currentEndpoint?.getId()}/create-endpoint-version`,
            );
          }}
        >
          Create new version
          <Plus strokeWidth={1.5} className="ml-1.5 h-4 w-4" />
        </IBlueButton>
      )}
    </>
  );
};
