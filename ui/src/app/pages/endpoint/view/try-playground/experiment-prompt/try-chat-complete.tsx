import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { InvokeResponse } from '@rapidaai/react';
import { useRapidaStore } from '@/stores/app';
import { useCredential } from '@/hooks/use-credential';
import { useEffect, useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Variable } from '@rapidaai/react';

import {
  EndpointArgumentList,
  getUnsupportedEndpointVariables,
  InputFormData,
} from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/input-var-form';

import { OutputMessage } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/output-message';
import { PlaygroundHeader } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/playground-header';
import { invokeEndpoint } from '@/clients';
import { InlineNotification } from '@carbon/react';

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
  } = useForm<Record<string, string>>({
    mode: 'onChange',
  });

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
  const [variables, setVariables] = useState<Variable[]>([]);
  const unsupportedVariables = useMemo(
    () => getUnsupportedEndpointVariables(variables),
    [variables],
  );
  const hasUnsupportedVariables = unsupportedVariables.length > 0;

  useEffect(() => {
    let endpointProviderModel = props.endpointProviderModel;
    if (endpointProviderModel.getChatcompleteprompt()) {
      let allVars = endpointProviderModel
        .getChatcompleteprompt()
        ?.getPromptvariablesList();
      if (allVars) setVariables(allVars);
    }
  }, [props.endpointProviderModel]);

  /**
   *
   * @param data
   */
  const onInvoke = async data => {
    if (hasUnsupportedVariables) {
      setCallerResponse(null);
      setError(
        'This endpoint contains variables that are not runnable in the playground.',
      );
      return;
    }

    showLoader();
    setError('');
    setCallerResponse(null);

    const formDataMap = await InputFormData(data);
    invokeEndpoint({
      endpointId: props.endpointProviderModel.getEndpointid(),
      endpointProviderModelId: props.endpointProviderModel.getId(),
      args: formDataMap,
      auth: {
        userId,
        token,
        projectId,
      },
    })
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
        hideLoader();
        setError('Unable to execute the endpoint, please try again.');
      });
  };

  return (
    <form
      onSubmit={handleSubmit(onInvoke)}
      className="flex min-h-0 flex-1 flex-col bg-surface"
    >
      <PlaygroundHeader
        isValid={isValid}
        loading={loading}
        disabled={hasUnsupportedVariables}
      />
      <div className="min-h-0 flex-1 overflow-auto bg-surface">
        <div
          className="grid min-h-full w-full lg:grid-cols-[minmax(18rem,2fr)_minmax(0,3fr)]"
          data-testid="endpoint-playground-console"
        >
          <section
            className="flex min-h-0 flex-col border-b border-border-subtle bg-layer lg:border-b-0 lg:border-r"
            aria-labelledby="playground-arguments-title"
          >
            <div className="border-b border-border-subtle bg-layer-elevated px-4 py-3">
              <h2
                id="playground-arguments-title"
                className="text-sm font-semibold text-foreground"
              >
                Arguments
              </h2>
            </div>
            <div
              aria-label="Endpoint arguments"
              className="min-h-0 flex-1 overflow-y-auto"
              tabIndex={0}
            >
              {variables.length > 0 ? (
                <EndpointArgumentList
                  variables={variables}
                  getRegistration={variable =>
                    register(variable.getName(), {
                      required: 'Please provide a valid input.',
                    })
                  }
                  getErrorMessage={variable =>
                    errors[variable.getName()]?.message?.toString()
                  }
                />
              ) : (
                <InlineNotification
                  kind="info"
                  lowContrast
                  hideCloseButton
                  title="No arguments required"
                  subtitle="Run this endpoint directly to inspect its response."
                  className="m-4! max-w-full!"
                />
              )}
            </div>
          </section>
          <OutputMessage
            callerResponse={callerResponse}
            error={error}
            loading={loading}
            isValid={isValid}
            errors={errors}
            className="min-h-[28rem]"
          />
        </div>
      </div>
    </form>
  );
}
