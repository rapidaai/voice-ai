import {
  ConnectionConfig,
  EndpointDefinition,
  Invoke,
  InvokeRequest,
  StringToAny,
} from '@rapidaai/react';
import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { InvokeResponse } from '@rapidaai/react';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import React, { useCallback, useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Variable } from '@rapidaai/react';

import {
  InputFormData,
  InputVarForm,
} from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/input-var-form';

import {
  JsonTextarea,
  NumberTextarea,
  ParagraphTextarea,
  TextTextarea,
  UrlTextarea,
} from '@/app/components/Form/Textarea';

import { InputVarType } from '@/models/common';
import { OutputMessage } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/output-message';
import { PlaygroundHeader } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/playground-header';
import { connectionConfig } from '@/configs';

export function TryChatComplete(props: {
  currentEndpoint: Endpoint;
  endpointProviderModel: EndpointProviderModel;
}) {
  /**
   *
   */
  const [error, setError] = useState('');
  const {
    register,
    handleSubmit,
    formState: { errors, isValid },
  } = useForm();

  /**
   *
   */
  const [callerResponse, setCallerResponse] = useState<InvokeResponse | null>(
    null,
  );
  /**
   *
   */
  const [userId, token, projectId] = useCredential();

  /**
   *
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   *
   */
  const [variables, setVaribales] = useState<Variable[]>([]);

  useEffect(() => {
    let endpointProviderModel = props.endpointProviderModel;
    if (endpointProviderModel.getChatcompleteprompt()) {
      let allVars = endpointProviderModel
        .getChatcompleteprompt()
        ?.getPromptvariablesList();
      if (allVars) setVaribales(allVars);
    }
  }, [props.endpointProviderModel]);

  /**
   *
   * @param data
   */
  const onInvoke = async data => {
    showLoader();
    setError('');
    setCallerResponse(null);

    const formDataMap = await InputFormData(data);
    const request = new InvokeRequest();
    const endpoint = new EndpointDefinition();
    endpoint.setEndpointid(props.endpointProviderModel.getEndpointid());
    endpoint.setVersion(props.endpointProviderModel.getId());
    request.setEndpoint(endpoint);
    request.getMetadataMap().set('source', StringToAny('web-app'));
    request.getMetadataMap().set('experiemental', StringToAny('true'));
    formDataMap.forEach((value, key) => {
      request.getArgsMap().set(key, value);
    });
    Invoke(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: userId,
        projectId: projectId,
      }),
    )
      .then(at => {
        hideLoader();
        if (at?.getSuccess()) {
          setCallerResponse(at);
          return;
        }
        let er = at?.getError();
        if (er) {
          setError(er.getHumanmessage());
          return;
        }
        setError('Unable to execute the endpoint, please try again.');
      })
      .catch(error => {
        showLoader();
        setError('Unable to execute the endpoint, please try again.');
      });
  };

  return (
    <form onSubmit={handleSubmit(onInvoke)} className="flex flex-col flex-1">
      <PlaygroundHeader isValid={isValid} loading={loading} />
      <div className="flex-1 overflow-y-auto transition-all duration-300 max-h-screen">
        <div className="flex flex-1 h-full w-full flex-col overflow-auto bg-divider-500/20 pt-0">
          <div className="flex flex-col">
            {variables.map((x, idx) => {
              return (
                <InputVarForm key={idx} var={x}>
                  {x.getType() === InputVarType.textInput && (
                    <TextTextarea
                      className="bg-light-background"
                      id={x.getName()}
                      {...register(x.getName(), {
                        required: 'Please provide a valid input.',
                      })}
                    />
                  )}
                  {x.getType() === InputVarType.paragraph && (
                    <ParagraphTextarea
                      id={x.getName()}
                      {...register(x.getName(), {
                        required: 'Please provide a valid input.',
                      })}
                    />
                  )}

                  {x.getType() === InputVarType.number && (
                    <NumberTextarea
                      id={x.getName()}
                      {...register(x.getName(), {
                        required: 'Please provide a valid input.',
                      })}
                    />
                  )}

                  {x.getType() === InputVarType.json && (
                    <JsonTextarea
                      id={x.getName()}
                      {...register(x.getName(), {
                        required: 'Please provide a valid input.',
                      })}
                    />
                  )}

                  {x.getType() === InputVarType.url && (
                    <UrlTextarea
                      id={x.getName()}
                      {...register(x.getName(), {
                        required: 'Please provide a valid input.',
                      })}
                    />
                  )}
                </InputVarForm>
              );
            })}
          </div>
          <OutputMessage
            callerResponse={callerResponse}
            error={error}
            loading={loading}
            isValid={isValid}
            errors={errors}
          />
        </div>
      </div>
    </form>
  );
}
