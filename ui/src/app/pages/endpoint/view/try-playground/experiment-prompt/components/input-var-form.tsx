import { Variable } from '@rapidaai/react';
import { Pill } from '@/app/components/Pill';
import React, { HTMLAttributes } from 'react';
import p from 'google-protobuf/google/protobuf/any_pb';
import { BytesToAny, StringToAny } from '@/utils/rapida_value';
import { cn } from '@/styles/media';

export const InputVarForm = React.forwardRef<
  HTMLTextAreaElement,
  {
    var: Variable;
  } & HTMLAttributes<HTMLDivElement>
>((props, ref) => {
  return (
    <div
      className={cn(
        'bg-white dark:bg-gray-900 focus-within:border-blue-600! border border-transparent!',
        props.className,
      )}
    >
      <label
        htmlFor={props.var.getName()}
        className="flex shrink-0 items-center justify-between break-all p-3 pr-5 font-mono text-sm font-medium tracking-wide"
      >
        <span>
          {'{{'}
          {props.var.getName()}
          {'}}'}
        </span>
        <Pill className="py-0.5 px-2">{props.var.getType()}</Pill>
      </label>
      {props.children}
    </div>
  );
});

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
    if (value instanceof File) {
      try {
        const fileContent = await handleFileAsync(value);
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
