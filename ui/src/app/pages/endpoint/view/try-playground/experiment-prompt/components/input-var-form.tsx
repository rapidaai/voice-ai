import { Variable, BytesToAny, StringToAny } from '@rapidaai/react';
import React from 'react';
import p from 'google-protobuf/google/protobuf/any_pb';
import { cn } from '@/utils';
import { InputVarType } from '@/models/common';
import { InlineNotification, Tag, TextArea, TextInput } from '@carbon/react';
import type { UseFormRegisterReturn } from 'react-hook-form';

export const ENDPOINT_PLAYGROUND_EDITABLE_TYPES = [
  InputVarType.stringInput,
  InputVarType.textInput,
  InputVarType.paragraph,
  InputVarType.number,
  InputVarType.json,
  InputVarType.url,
] as string[];

const UNSUPPORTED_TYPE_COPY: Record<string, string> = {
  [InputVarType.select]:
    'Select variables do not expose options in the endpoint model yet.',
  [InputVarType.files]:
    'File variables require upload support before this endpoint can run here.',
  [InputVarType.contexts]:
    'Context variables are resolved by runtime retrieval and cannot be edited here.',
};

const VARIABLE_PLACEHOLDERS: Record<string, string> = {
  [InputVarType.stringInput]: 'Enter value',
  [InputVarType.textInput]: 'Enter value',
  [InputVarType.paragraph]: 'Enter paragraph',
  [InputVarType.number]: 'Enter number',
  [InputVarType.url]: 'https://example.com',
  [InputVarType.json]: '{ "key": "value" }',
};

export function isEndpointVariableEditable(type?: string): boolean {
  return Boolean(type && ENDPOINT_PLAYGROUND_EDITABLE_TYPES.includes(type));
}

export function getUnsupportedEndpointVariableReason(type?: string): string {
  if (!type) return 'This variable does not define a supported type.';
  return (
    UNSUPPORTED_TYPE_COPY[type] ??
    'This variable type is not supported by the endpoint playground.'
  );
}

export function getUnsupportedEndpointVariables(
  variables: Variable[],
): Variable[] {
  return variables.filter(
    variable => !isEndpointVariableEditable(variable.getType()),
  );
}

function getFieldId(variable: Variable): string {
  const rawId = variable.getId() || variable.getName();
  return `endpoint-playground-${rawId.replace(/[^a-zA-Z0-9_-]/g, '-')}`;
}

function getRows(type: string): number {
  if (type === InputVarType.json) return 8;
  if (type === InputVarType.paragraph) return 5;
  return 3;
}

function renderVariableInput({
  variable,
  registration,
  error,
}: {
  variable: Variable;
  registration?: UseFormRegisterReturn;
  error?: React.ReactNode;
}) {
  const type = variable.getType();
  const id = getFieldId(variable);
  const labelText = `{{${variable.getName()}}}`;
  const invalid = Boolean(error);
  const invalidText = error || 'Please provide a valid input.';
  const commonProps = {
    id,
    labelText,
    hideLabel: true,
    defaultValue: variable.getDefaultvalue(),
    invalid,
    invalidText,
    placeholder: VARIABLE_PLACEHOLDERS[type] ?? 'Enter value',
    ...registration,
  };

  if (
    type === InputVarType.paragraph ||
    type === InputVarType.json ||
    type === InputVarType.textInput
  ) {
    return (
      <TextArea
        {...commonProps}
        rows={getRows(type)}
        className={cn(type === InputVarType.json && '[&_textarea]:font-mono')}
      />
    );
  }

  return (
    <TextInput
      {...commonProps}
      type={
        type === InputVarType.number
          ? 'number'
          : type === InputVarType.url
            ? 'url'
            : 'text'
      }
    />
  );
}

export function EndpointArgumentField(props: {
  variable: Variable;
  registration?: UseFormRegisterReturn;
  error?: React.ReactNode;
}) {
  const { variable, registration, error } = props;
  const type = variable.getType();
  const editable = isEndpointVariableEditable(type);

  return (
    <div
      className={cn(
        'border-b border-border-subtle bg-layer px-4 py-4',
        'last:border-b-0',
      )}
    >
      <label
        htmlFor={getFieldId(variable)}
        className="mb-3 flex shrink-0 items-start justify-between gap-4"
      >
        <span className="min-w-0">
          <span
            className="block truncate font-mono text-sm font-semibold text-foreground"
            title={variable.getName()}
          >
            {`{{${variable.getName()}}}`}
          </span>
          {variable.getDefaultvalue() && (
            <span className="mt-1 block text-xs text-muted">
              Default value loaded
            </span>
          )}
        </span>
        <Tag
          type={editable ? 'blue' : 'gray'}
          size="sm"
          title={type || 'unknown'}
        >
          {type || 'unknown'}
        </Tag>
      </label>

      {editable ? (
        renderVariableInput({ variable, registration, error })
      ) : (
        <InlineNotification
          kind="warning"
          lowContrast
          hideCloseButton
          title="Not runnable in playground"
          subtitle={getUnsupportedEndpointVariableReason(type)}
          className="!m-0 !max-w-full"
        />
      )}
    </div>
  );
}

export function EndpointArgumentList(props: {
  variables: Variable[];
  getRegistration: (variable: Variable) => UseFormRegisterReturn;
  getErrorMessage?: (variable: Variable) => React.ReactNode;
}) {
  const { variables, getRegistration, getErrorMessage } = props;

  if (variables.length === 0) {
    return (
      <div className="flex min-h-[14rem] items-center justify-center bg-layer px-6 text-center">
        <div>
          <p className="text-sm font-semibold text-foreground">
            No arguments defined
          </p>
          <p className="mt-1 max-w-sm text-sm text-muted">
            This endpoint does not expose prompt variables.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="divide-y divide-border-subtle">
      {variables.map(variable => {
        const editable = isEndpointVariableEditable(variable.getType());
        return (
          <EndpointArgumentField
            key={variable.getId() || variable.getName()}
            variable={variable}
            registration={editable ? getRegistration(variable) : undefined}
            error={getErrorMessage?.(variable)}
          />
        );
      })}
    </div>
  );
}

//
export const InputFormData = async (data): Promise<Map<string, p.Any>> => {
  const formDataMap = new Map<string, p.Any>();
  const handleFileAsync = (file: File): Promise<Uint8Array> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        resolve(new Uint8Array(reader.result as ArrayBuffer));
      };
      reader.onerror = reject;
      reader.readAsArrayBuffer(file);
    });
  };

  for (const [key, value] of Object.entries(data)) {
    const file =
      value instanceof File
        ? value
        : typeof FileList !== 'undefined' &&
            value instanceof FileList &&
            value.length > 0
          ? value.item(0)
          : null;

    if (file) {
      try {
        const fileContent = await handleFileAsync(file);
        formDataMap.set(key, BytesToAny(fileContent));
      } catch (error) {
        // return error;
      }
    } else {
      formDataMap.set(key, StringToAny(value as string));
    }
  }
  return formDataMap;
};
