import { useState, type FC } from 'react';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { Textarea } from '@/app/components/Form/Textarea';
import { Input } from '@/app/components/Form/Input';
import { Label } from '@/app/components/Form/Label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { JsonEditor } from '@/app/components/json-editor';
import { AssistantTool } from '@rapidaai/react';

interface ToolSettingOptionProps {
  isShow: boolean;
  onClose: () => void;
  assistantTool: AssistantTool;
  onSave: (at: AssistantTool) => void;
  readonly: boolean;
}

export const ToolSettingOption: React.FC<ToolSettingOptionProps> = ({
  isShow,
  onClose,
  assistantTool,
  onSave,
  readonly,
}) => {
  //   const [name, setName] = useState(
  //     assistantTool.getOptions()?.getFieldsMap().get('name')?.getStringValue() ||
  //       '',
  //   );
  //   const [description, setDescription] = useState(
  //     assistantTool
  //       .getOptions()
  //       ?.getFieldsMap()
  //       .get('description')
  //       ?.getStringValue() || '',
  //   );
  //   const [fields, setFields] = useState(
  //     JSON.stringify(
  //       assistantTool
  //         .getOptions()
  //         ?.getFieldsMap()
  //         .get('parameters')
  //         ?.toJavaScript() || {},
  //       null,
  //       2,
  //     ),
  //   );

  //   const handleSave = () => {
  //     assistantTool
  //       .getOptions()
  //       ?.getFieldsMap()
  //       .get('description')
  //       ?.setStringValue(description);
  //     assistantTool
  //       .getOptions()
  //       ?.getFieldsMap()
  //       .get('name')
  //       ?.setStringValue(name);

  //     assistantTool
  //       .getOptions()
  //       ?.getFieldsMap()
  //       .get('parameters')
  //       ?.setStructValue(Struct.fromJavaScript(JSON.parse(fields)));

  //     onSave(assistantTool);
  //     onClose();
  //   };

  return <></>;
};
