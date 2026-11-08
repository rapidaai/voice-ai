import { useCallback, useMemo, useState } from 'react';
import {
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  type ModalProps,
  PrimaryButton,
  SecondaryButton,
} from '@/app/components/ui/primitives';
import assistantTemplates from '@/prompts/assistants/index.json';
import { ContentSwitcher, SelectableTile, Switch, Tag } from '@carbon/react';

export interface AssistantTemplate {
  name: string;
  description: string;
  category: string;
  provider: string;
  model: string;
  parameters: {
    temperature: number;
  };
  instruction: {
    role: string;
    content: string;
  }[];
}

interface ConfigureAssistantTemplateDialogProps extends ModalProps {
  onSelectTemplate?: (template: AssistantTemplate) => void;
}

export function ConfigureAssistantTemplateDialog({
  modalOpen,
  setModalOpen,
  onSelectTemplate,
}: ConfigureAssistantTemplateDialogProps) {
  const [selectedTemplate, setSelectedTemplate] =
    useState<AssistantTemplate | null>(null);
  const [activeCategory, setActiveCategory] = useState<string>('All');

  const templates = useMemo(
    () => assistantTemplates as AssistantTemplate[],
    [],
  );
  const categories = useMemo(
    () => ['All', ...Array.from(new Set(templates.map(t => t.category)))],
    [templates],
  );

  const visible = useMemo(
    () =>
      activeCategory === 'All'
        ? templates
        : templates.filter(t => t.category === activeCategory),
    [activeCategory, templates],
  );

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
      containerClassName="!h-[90vh] !w-[90vw] !max-h-[90vh] !max-w-[90vw]"
    >
      <ModalHeader
        label="Assistant"
        title="Select a usecase template"
        onClose={closeDialog}
      />

      <ModalBody hasScrollingContent>
        <p className="text-xs text-muted leading-relaxed mb-4">
          Choose a pre-configured assistant template to auto-fill your model,
          prompt, and parameters. You can customise everything after selecting.
        </p>

        <div className="flex items-center gap-2 flex-wrap mb-4">
          <ContentSwitcher
            onChange={({ name }) => {
              setActiveCategory(name as string);
              setSelectedTemplate(null);
            }}
            selectedIndex={categories.indexOf(activeCategory)}
            size="sm"
          >
            {categories.map(cat => (
              <Switch key={cat} name={cat} text={cat} />
            ))}
          </ContentSwitcher>
          {selectedTemplate && (
            <span className="ml-auto text-xs text-muted ">
              Selected:{' '}
              <span className="font-medium text-foreground ">
                {selectedTemplate.name}
              </span>
            </span>
          )}
        </div>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {visible.map((template, index) => {
            const isSelected = selectedTemplate?.name === template.name;
            return (
              <SelectableTile
                id={`assistant-template-${index}`}
                key={index}
                selected={isSelected}
                onClick={() => setSelectedTemplate(template)}
                className="flex min-h-56 flex-col text-left"
              >
                <Tag size="sm" type="blue" className="!self-start !mb-2">
                  {template.category}
                </Tag>

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
