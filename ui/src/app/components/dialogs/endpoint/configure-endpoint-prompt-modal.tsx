import { useCallback, useState } from 'react';
import {
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  type ModalProps,
  PrimaryButton,
  SecondaryButton,
} from '@/app/components/ui/primitives';
import endpointTemplates from '@/prompts/endpoints/index.json';
import { SelectableTile, Tag } from '@carbon/react';

interface EndpointTemplate {
  name: string;
  description: string;
  provider: string;
  model: string;
  parameters: {
    temperature: number;
    response_format: string;
  };
  instruction: {
    role: string;
    content: string;
  }[];
}

interface ConfigureEndpointPromptDialogProps extends ModalProps {
  onSelectTemplate?: (template: EndpointTemplate) => void;
}

export function ConfigureEndpointPromptDialog({
  modalOpen,
  setModalOpen,
  onSelectTemplate,
}: ConfigureEndpointPromptDialogProps) {
  const [selectedTemplate, setSelectedTemplate] =
    useState<EndpointTemplate | null>(null);

  const closeDialog = useCallback(() => {
    setModalOpen(false);
  }, [setModalOpen]);

  const handleContinue = useCallback(() => {
    if (selectedTemplate) {
      onSelectTemplate?.(selectedTemplate);
    }
    closeDialog();
  }, [closeDialog, onSelectTemplate, selectedTemplate]);

  return (
    <Modal
      open={modalOpen}
      onClose={closeDialog}
      size="lg"
      containerClassName="!w-[900px] !max-w-[900px]"
    >
      <ModalHeader
        label="Endpoint"
        title="Select a usecase template"
        onClose={closeDialog}
      />

      <ModalBody hasForm hasScrollingContent>
        <p className="text-xs text-muted leading-relaxed mb-4">
          Choose a pre-configured template to auto-fill your model, prompt, and
          parameters. You can customise everything after selecting.
        </p>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {(endpointTemplates as EndpointTemplate[]).map((template, index) => {
            const isSelected = selectedTemplate?.name === template.name;
            return (
              <SelectableTile
                id={`endpoint-template-${index}`}
                key={index}
                selected={isSelected}
                onClick={() => setSelectedTemplate(template)}
                className="flex min-h-52 flex-col text-left"
              >
                <h3 className="text-sm font-semibold leading-snug mb-1.5 pr-6">
                  {template.name}
                </h3>

                <p className="text-xs text-muted leading-relaxed line-clamp-2 mb-4 flex-1">
                  {template.description}
                </p>

                <div className="flex flex-wrap gap-1.5">
                  <Tag size="sm" type="cool-gray">
                    {template.provider}
                  </Tag>
                  <Tag size="sm" type="cool-gray">
                    {template.model}
                  </Tag>
                  <Tag size="sm" type="cool-gray">
                    Temp {template.parameters.temperature}
                  </Tag>
                  {template.parameters.response_format && (
                    <Tag size="sm" type="blue">
                      JSON
                    </Tag>
                  )}
                </div>
              </SelectableTile>
            );
          })}
        </div>
      </ModalBody>

      <ModalFooter>
        <SecondaryButton size="lg" onClick={closeDialog}>
          Cancel
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          disabled={!selectedTemplate}
          onClick={handleContinue}
        >
          Use template
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
