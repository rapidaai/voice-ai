import React, { useContext } from 'react';
import { CardOptionMenu } from '@/app/components/Menu';
import { useEndpointPageStore } from '@/hooks';
import { Endpoint, EndpointProviderModel } from '@rapidaai/react';

interface EndpointOptionProps {
  endpoint: Endpoint;
  endpointProviderModel?: EndpointProviderModel;
}
/**
 *
 * @param props
 * @returns
 */
export function EndpointOptions(props: EndpointOptionProps) {
  /**
   * action
   */
  const endpointActions = useEndpointPageStore();

  /**
   * options that will be display
   */
  const options = [
    {
      option: 'Integration instruction',
      onActionClick: () => {
        endpointActions.onShowInstruction();
      },
    },

    {
      option: 'Update endpoint tags',
      onActionClick: () => {
        endpointActions.onShowEditTagVisible(props.endpoint);
      },
    },

    {
      option: 'Update endpoint detail',
      onActionClick: () => {
        endpointActions.onShowUpdateDetailVisible(props.endpoint);
      },
    },
  ];
  return <CardOptionMenu options={options} classNames="rounded-[2px]" />;
}
