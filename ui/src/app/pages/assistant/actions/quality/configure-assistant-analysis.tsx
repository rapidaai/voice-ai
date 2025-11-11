import { FC } from 'react';
import { useParams } from 'react-router-dom';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import React, { useState } from 'react';
import { InputGroup } from '@/app/components/input-group';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { InputHelper } from '@/app/components/input-helper';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { InputCheckbox } from '@/app/components/Form/Checkbox';

export function ConfigureAssistantAnalysisPage() {
  const { assistantId } = useParams();
  return (
    <>
      {assistantId && <ConfigureAssistantAnalysis assistantId={assistantId} />}
    </>
  );
}

const ConfigureAssistantAnalysis: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  let navigator = useGlobalNavigation();

  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const [errorMessage, setErrorMessage] = useState('');
  return (
    <>
      <ConfirmDialogComponent />
      <div className="relative flex flex-col flex-1">
        <PageHeaderBlock>
          <PageTitleBlock>Quality & Compliance</PageTitleBlock>
        </PageHeaderBlock>

        <div className="overflow-auto flex flex-col flex-1 pb-20">
          <div className=" bg-white dark:bg-gray-900">
            <SOPAdherenceComponent />
            <SentimentAnalysisComponent />
          </div>
        </div>

        <PageActionButtonBlock errorMessage={errorMessage}>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => showDialog(navigator.goBack)}
          >
            Cancel
          </ICancelButton>
          <IBlueBGButton type="submit" className="px-4 rounded-[2px]">
            Configure analysis
          </IBlueBGButton>
        </PageActionButtonBlock>
      </div>
    </>
  );
};

interface MetricItem {
  id: string;
  label: string;
  description: string;
}

interface SOPAdherenceMetricsState {
  greetingIntroduction: boolean;
  customerVerification: boolean;
  attentiveListening: boolean;
  issueResolution: boolean;
  followUp: boolean;
  closing: boolean;
  prohibitedActions: boolean;
  excessiveWaiting: boolean;
  customerBlaming: boolean;
  unprofessionalLanguage: boolean;
  serviceDiscontinuation: boolean;
  overallAdherenceScore: boolean;
  majorViolations: boolean;
  improvementSuggestions: boolean;
}

const SOPAdherenceComponent: React.FC = () => {
  const [selectedSOPAdherenceMetrics, setSelectedSOPAdherenceMetrics] =
    useState<SOPAdherenceMetricsState>({
      greetingIntroduction: false,
      customerVerification: false,
      attentiveListening: false,
      issueResolution: false,
      followUp: false,
      closing: false,
      prohibitedActions: false,
      excessiveWaiting: false,
      customerBlaming: false,
      unprofessionalLanguage: false,
      serviceDiscontinuation: false,
      overallAdherenceScore: false,
      majorViolations: false,
      improvementSuggestions: false,
    });

  const sopAdherenceMetricsCount = Object.values(
    selectedSOPAdherenceMetrics,
  ).filter(Boolean).length;

  const handleSOPAdherenceMetricChange = (
    metric: keyof SOPAdherenceMetricsState,
  ): void => {
    setSelectedSOPAdherenceMetrics({
      ...selectedSOPAdherenceMetrics,
      [metric]: !selectedSOPAdherenceMetrics[metric],
    });
  };

  const sopAdherenceMetricsData: MetricItem[] = [
    {
      id: 'greetingIntroduction',
      label: 'Greeting & Introduction',
      description: 'Proper greeting and introduction of the agent.',
    },
    {
      id: 'customerVerification',
      label: 'Customer Verification',
      description: 'Correct process followed for customer verification.',
    },
    {
      id: 'attentiveListening',
      label: 'Attentive Listening',
      description: 'Whether the agent listened actively.',
    },
    {
      id: 'issueResolution',
      label: 'Issue Resolution',
      description: "Agent's ability to resolve the issue.",
    },
    {
      id: 'followUp',
      label: 'Follow-up',
      description: 'If follow-up actions were necessary.',
    },
    {
      id: 'closing',
      label: 'Closing',
      description: 'Professional closure of the call.',
    },
    {
      id: 'prohibitedActions',
      label: 'Prohibited Actions',
      description: 'Any forbidden actions taken during the call.',
    },
    {
      id: 'excessiveWaiting',
      label: 'Excessive Waiting',
      description: 'Length of any unnecessary waiting time for the customer.',
    },
    {
      id: 'customerBlaming',
      label: 'Customer Blaming',
      description: 'Whether the agent blamed the customer for the issue.',
    },
    {
      id: 'unprofessionalLanguage',
      label: 'Unprofessional Language',
      description: 'Use of unprofessional or inappropriate language.',
    },
    {
      id: 'serviceDiscontinuation',
      label: 'Service Discontinuation',
      description: 'Unjustified ending or refusal of service.',
    },
    {
      id: 'overallAdherenceScore',
      label: 'Overall Adherence Score',
      description: 'Combined score for adherence to SOPs.',
    },
    {
      id: 'majorViolations',
      label: 'Major Violations',
      description: 'Significant breaches of protocol.',
    },
    {
      id: 'improvementSuggestions',
      label: 'Improvement Suggestions',
      description: 'Areas where the agent can improve SOP adherence.',
    },
  ];

  const sopAdherenceMetricsPairs: Array<Array<MetricItem>> = [];
  for (let i = 0; i < sopAdherenceMetricsData.length; i += 2) {
    sopAdherenceMetricsPairs.push(sopAdherenceMetricsData.slice(i, i + 2));
  }

  return (
    <InputGroup
      title={`SOP Adherence (${sopAdherenceMetricsCount}/
        ${sopAdherenceMetricsData.length})`}
    >
      <div className="p-6">
        {sopAdherenceMetricsPairs.map((pair, pairIndex) => (
          <div
            key={`sop-adherence-pair-${pairIndex}`}
            className="grid grid-cols-2 gap-4 mb-4"
          >
            {pair.map(metric => (
              <div key={metric.id} className="flex items-start">
                <div className="flex h-5 items-center mt-1">
                  <InputCheckbox
                    id={metric.id}
                    type="checkbox"
                    checked={
                      selectedSOPAdherenceMetrics[
                        metric.id as keyof SOPAdherenceMetricsState
                      ]
                    }
                    onChange={() =>
                      handleSOPAdherenceMetricChange(
                        metric.id as keyof SOPAdherenceMetricsState,
                      )
                    }
                  />
                </div>
                <FieldSet className="ml-3">
                  <label htmlFor={metric.id} className="font-medium">
                    {metric.label}
                  </label>
                  <InputHelper>{metric.description}</InputHelper>
                </FieldSet>
              </div>
            ))}
            {pair.length === 1 && <div></div>}
          </div>
        ))}
      </div>
    </InputGroup>
  );
};

interface MetricItem {
  id: string;
  label: string;
  description: string;
}

interface SentimentAnalysisMetricsState {
  initialSentiment: boolean;
  sentimentTrend: boolean;
  finalSentiment: boolean;
  overallSentimentScore: boolean;
  explanation: boolean;
}

const SentimentAnalysisComponent: React.FC = () => {
  const [
    selectedSentimentAnalysisMetrics,
    setSelectedSentimentAnalysisMetrics,
  ] = useState<SentimentAnalysisMetricsState>({
    initialSentiment: false,
    sentimentTrend: false,
    finalSentiment: false,
    overallSentimentScore: false,
    explanation: false,
  });

  const sentimentAnalysisMetricsCount = Object.values(
    selectedSentimentAnalysisMetrics,
  ).filter(Boolean).length;

  const handleSentimentAnalysisMetricChange = (
    metric: keyof SentimentAnalysisMetricsState,
  ): void => {
    setSelectedSentimentAnalysisMetrics({
      ...selectedSentimentAnalysisMetrics,
      [metric]: !selectedSentimentAnalysisMetrics[metric],
    });
  };

  const sentimentAnalysisMetricsData: MetricItem[] = [
    {
      id: 'initialSentiment',
      label: 'Initial Sentiment',
      description: "Customer's sentiment at the beginning of the call.",
    },
    {
      id: 'sentimentTrend',
      label: 'Sentiment Trend',
      description: "Changes in the customer's sentiment throughout the call.",
    },
    {
      id: 'finalSentiment',
      label: 'Final Sentiment',
      description: "Customer's sentiment at the end of the call.",
    },
    {
      id: 'overallSentimentScore',
      label: 'Overall Sentiment Score',
      description:
        'Numeric score based on the overall sentiment trend (e.g., 1-5).',
    },
    {
      id: 'explanation',
      label: 'Explanation',
      description: 'Narrative explanation of sentiment trends.',
    },
  ];

  const sentimentAnalysisMetricsPairs: Array<Array<MetricItem>> = [];
  for (let i = 0; i < sentimentAnalysisMetricsData.length; i += 2) {
    sentimentAnalysisMetricsPairs.push(
      sentimentAnalysisMetricsData.slice(i, i + 2),
    );
  }

  return (
    <InputGroup
      title={` Sentiment Analysis (${sentimentAnalysisMetricsCount}/
        ${sentimentAnalysisMetricsData.length})`}
    >
      <div className="p-6">
        {sentimentAnalysisMetricsPairs.map((pair, pairIndex) => (
          <div
            key={`sentiment-analysis-pair-${pairIndex}`}
            className="grid grid-cols-2 gap-4 mb-4"
          >
            {pair.map(metric => (
              <div key={metric.id} className="flex items-start">
                <div className="flex h-5 items-center mt-1">
                  <InputCheckbox
                    id={metric.id}
                    type="checkbox"
                    checked={
                      selectedSentimentAnalysisMetrics[
                        metric.id as keyof SentimentAnalysisMetricsState
                      ]
                    }
                    onChange={() =>
                      handleSentimentAnalysisMetricChange(
                        metric.id as keyof SentimentAnalysisMetricsState,
                      )
                    }
                  />
                </div>
                <FieldSet className="ml-3">
                  <label htmlFor={metric.id} className="font-medium">
                    {metric.label}
                  </label>
                  <InputHelper>{metric.description}</InputHelper>
                </FieldSet>
              </div>
            ))}
            {pair.length === 1 && <div></div>}
          </div>
        ))}
      </div>
    </InputGroup>
  );
};
