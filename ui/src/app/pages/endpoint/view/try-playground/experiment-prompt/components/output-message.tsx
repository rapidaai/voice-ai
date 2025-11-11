import { Metric } from '@rapidaai/react';
import { InvokeResponse } from '@rapidaai/react';
import { Spinner } from '@/app/components/Loader/Spinner';
import { MarkdownViewer } from '@/app/components/MarkdownViewer';
import { Tab } from '@/app/components/Tab';
import { ExecuteMessage } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/execute-message';
import { cn } from '@/styles/media';
import { FC, useEffect, useState } from 'react';
import { CodeHighlighting } from '@/app/components/code-highlighting';

/**
 * OutputMessage Component
 *
 * This component displays the output of an executed program, handles the loading state,
 * and displays any errors encountered during execution. It supports rendering text,
 * images, and audio based on the content type received in the `CallerResponse`.
 *
 * Props:
 * - callerResponse: The response object returned from the execution API.
 * - error: A string representing any errors encountered during the execution.
 * - loading: A boolean indicating if the execution is still in progress.
 * - isValid: A boolean that indicates whether the input form is valid.
 * - errors: A list of errors related to the input form validation.
 */
export const OutputMessage: FC<{
  callerResponse: InvokeResponse | null;
  error: string;
  loading: boolean;
  isValid: boolean;
  errors: any;
}> = ({ callerResponse, error, isValid, errors, loading }) => {
  // State to manage the outputs extracted from the callerResponse
  const [outputs, setOutputs] = useState<
    | {
        content: Uint8Array | string;
        contenttype: string;
        contentformat: string;
      }[]
    | null
  >(null);

  // State to manage the metrics received from the callerResponse
  const [endpointMetrics, setEndpointMetrics] = useState<Array<Metric>>([]);

  // useEffect to handle the processing of callerResponse
  useEffect(() => {
    if (callerResponse) {
      const metrics = callerResponse.getMetricsList();
      const responses = callerResponse.getDataList();

      if (responses && responses.length > 0) {
        setOutputs(
          responses.map(response => ({
            contentformat: response.getContentformat(),
            contenttype: response.getContenttype(),
            content:
              response.getContenttype() === 'text' ||
              response.getContenttype() === 'image'
                ? response.getContent_asB64()
                : response.getContent_asU8(),
          })),
        );
      }

      if (metrics) setEndpointMetrics(metrics);
    } else {
      setOutputs(null);
      setEndpointMetrics([]);
    }
  }, [callerResponse]);

  return (
    <div className="flex-col flex flex-1">
      <ExecuteMessage
        className="dark:border-gray-800  border-gray-200"
        apiError={error}
        loading={loading}
        metrics={endpointMetrics}
        formError={isValid ? undefined : errors}
      />
      <Tab
        active="ouput"
        className={cn(
          'text-sm',
          'bg-gray-50 border-b dark:bg-gray-900 dark:border-gray-800',
        )}
        tabs={[
          {
            label: 'ouput',
            element: (
              <div className="flex-1">
                <div className="min-h-[250px] max-h-[450px] flex flex-col justify-start items-center relative">
                  {outputs ? (
                    outputs.map((out, i) => {
                      if (out.contenttype === 'text') {
                        return (
                          <MarkdownViewer
                            text={atob(out.content as string)}
                            key={i}
                          />
                        );
                      }

                      if (out.contenttype === 'image') {
                        return (
                          <div
                            key={i}
                            className="bg-white dark:bg-gray-900 p-4 rounded-[2px] flex justify-start items-start w-full"
                          >
                            <img
                              alt="response-image"
                              className="rounded-[2px] h-[250px] border shadow-sm"
                              src={`data:image/png;base64,${atob(out.content as string)}`}
                            />
                          </div>
                        );
                      }

                      if (out.contenttype === 'audio') {
                        return (
                          <AudioPlayer audioData={out.content as Uint8Array} />
                        );
                      }

                      return null;
                    })
                  ) : (
                    <div className="opacity-60 w-full p-4">
                      Output will be printed here after the completion of
                      execution
                    </div>
                  )}
                </div>
              </div>
            ),
          },
          {
            label: 'attributes',
            element: (
              <div className="flex-1">
                {callerResponse ? (
                  <CodeHighlighting
                    className="max-w-full h-full"
                    code={JSON.stringify(
                      callerResponse.getDataList().map(rc => rc.toObject()),
                      null,
                      2,
                    )}
                  />
                ) : (
                  <div className="opacity-60 w-full p-4">
                    Attributes will be available here after the completion of
                    execution
                  </div>
                )}
              </div>
            ),
          },
          {
            label: 'metadatas',
            element: (
              <div className="flex-1">
                {callerResponse ? (
                  <CodeHighlighting
                    className="max-w-full h-full"
                    code={JSON.stringify(callerResponse.getMeta(), null, 2)}
                  />
                ) : (
                  <div className="opacity-60 w-full p-4">
                    Metadata will be available here after the completion of
                    execution
                  </div>
                )}
              </div>
            ),
          },
          {
            label: 'metrics',
            element: (
              <div className="flex-1">
                {callerResponse ? (
                  <CodeHighlighting
                    className="max-w-full h-full"
                    code={JSON.stringify(
                      callerResponse.getMetricsList().map(rc => rc.toObject()),
                      null,
                      2,
                    )}
                  />
                ) : (
                  <div className="opacity-60 w-full p-4">
                    Metrics will be available here after the completion of
                    execution
                  </div>
                )}
              </div>
            ),
          },
        ]}
      />
    </div>
  );
};

/**
 * AudioPlayer Component
 *
 * This component is responsible for playing audio data. It takes in an audio data buffer (Uint8Array),
 * creates a Blob URL for the audio, and renders an HTML audio player with controls.
 * If the audio data is not available, it shows a loading spinner.
 *
 * Props:
 * - audioData: A Uint8Array representing the audio data to be played.
 */
const AudioPlayer: FC<{ audioData: ArrayBuffer | Uint8Array }> = ({
  audioData,
}) => {
  const [audioSrc, setAudioSrc] = useState<string | null>(null);

  useEffect(() => {
    if (audioData) {
      const blob = new Blob([audioData], { type: 'audio/wav' }); // Create a Blob from the audio data
      const url = URL.createObjectURL(blob); // Generate a URL for the Blob
      setAudioSrc(url); // Set the URL as the source for the audio player

      // Cleanup function to revoke the Blob URL when the component unmounts or audioData changes
      return () => {
        URL.revokeObjectURL(url);
      };
    }
  }, [audioData]);

  return (
    <div className="h-full flex justify-center items-center flex-1">
      {audioSrc ? (
        <audio controls>
          <source src={audioSrc} type="audio/wav" />
          Your browser does not support the audio element.
        </audio>
      ) : (
        <Spinner /> // Display a loading spinner if the audio source is not yet available
      )}
    </div>
  );
};
