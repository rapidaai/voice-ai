import { IBlueBorderButton, ICancelButton } from '@/app/components/Form/Button';
import { Input } from '@/app/components/Form/Input';
import { cn } from '@/styles/media';
import { Plus, Trash2 } from 'lucide-react';
import { FC, useEffect, useState } from 'react';

interface Header {
  key: string;
  value: string;
}

export const APiHeader: FC<{
  inputClass?: string;
  headers: Header[];
  setHeaders: (headers: Header[]) => void;
}> = ({ headers, setHeaders, inputClass }) => {
  const updateHeader = (
    index: number,
    field: 'key' | 'value',
    value: string,
  ) => {
    const updatedHeaders = [...headers];
    updatedHeaders[index][field] = value;
    setHeaders(updatedHeaders);
  };

  return (
    <>
      <div className="text-sm grid w-full ">
        {headers.map((header, index) => (
          <div
            key={index}
            className="grid grid-cols-2 border-b border-gray-400 dark:border-gray-600"
          >
            <div className="flex col-span-1 items-center border-r">
              <Input
                value={header.key}
                onChange={e => updateHeader(index, 'key', e.target.value)}
                placeholder="Key"
                className={cn(
                  'bg-light-background w-full border-none',
                  inputClass,
                )}
              />
            </div>
            <div className="col-span-1 flex">
              <Input
                value={header.value}
                onChange={e => updateHeader(index, 'value', e.target.value)}
                placeholder="Value"
                className={cn(
                  'bg-light-background w-full border-none',
                  inputClass,
                )}
              />
              <ICancelButton
                className={cn(
                  'border-transparent hover:!border-red-600 outline-hidden cursor-pointer hover:!text-red-600',
                  inputClass,
                )}
                onClick={() => {
                  const updatedHeaders = headers.filter((_, i) => i !== index);
                  setHeaders(updatedHeaders);
                }}
                type="button"
              >
                <Trash2 className="w-4 h-4" strokeWidth={1.5} />
              </ICancelButton>
            </div>
          </div>
        ))}
      </div>
      <IBlueBorderButton
        onClick={() => {
          const updatedHeaders = [...headers, { key: '', value: '' }];
          setHeaders(updatedHeaders);
        }}
        className="justify-between space-x-8"
      >
        <span>Add header</span> <Plus className="h-4 w-4 ml-1.5" />
      </IBlueBorderButton>
    </>
  );
};

export const APiStringHeader: FC<{
  inputClass?: string;
  headerValue?: string;
  setHeaderValue: (s: string) => void;
}> = ({ headerValue = '{}', setHeaderValue, inputClass }) => {
  const [headers, setHeaders] = useState<Header[]>([{ key: '', value: '' }]);

  // Sync headers when headerValue prop changes
  useEffect(() => {
    try {
      const parsedHeaders = JSON.parse(headerValue);
      const headerArray = Object.entries(parsedHeaders).map(([key, value]) => ({
        key,
        value: value as string,
      }));
      setHeaders(headerArray);
    } catch (error) {
      console.error('Error parsing header JSON:', error);
    }
  }, [headerValue]);

  // Sync internal headers array with external JSON string
  const handleSetHeaders = (updatedHeaders: Header[]) => {
    setHeaders(updatedHeaders);
    const headersObject = updatedHeaders.reduce(
      (acc, header) => {
        if (header.key) acc[header.key] = header.value; // Only include non-empty keys
        return acc;
      },
      {} as Record<string, string>,
    );
    setHeaderValue(JSON.stringify(headersObject));
  };

  return (
    <APiHeader
      inputClass={inputClass}
      headers={headers}
      setHeaders={handleSetHeaders}
    />
  );
};
