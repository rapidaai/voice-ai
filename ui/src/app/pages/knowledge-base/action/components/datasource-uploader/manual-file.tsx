import { GhostButton } from '@/app/components/ui/primitives';
import SingleRowWrapper from '@/app/components/layout/wrapper/single-row-wrapper';
import { useCreateKnowledgeDocumentPageStore } from '@/stores/knowledge/create-knowledge-document.store';
import { formatFileSize } from '@/utils/format';
import { Close, Document, DotMark, Upload } from '@carbon/icons-react';
import { FC, useCallback } from 'react';
import { useDropzone, Accept } from 'react-dropzone';

interface ManualFileProps {
  accepts?: Accept;
  multiple?: boolean;
  maxFiles?: number;
}

export const ManualFile: FC<ManualFileProps> = ({
  accepts = {
    'text/plain': ['.txt'],
    'text/markdown': ['.markdown', '.md'],
    'application/pdf': ['.pdf'],
    'text/html': ['.html', '.htm'],
    'application/vnd.ms-excel': ['.xls', '.xlsx'],
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document': [
      '.docx',
    ],
    'text/csv': ['.csv'],
    'message/rfc822': ['.eml'],
    'application/vnd.ms-outlook': ['.msg'],
    'application/vnd.ms-powerpoint': ['.ppt', '.pptx'],
    'application/xml': ['.xml'],
    'application/json': ['.json'],
    'application/epub+zip': ['.epub'],
  },
  multiple = true,
  maxFiles = 10,
}) => {
  const {
    onAddKnowledgeDocument,
    onRemoveKnowledgeDocument,
    knowledgeDocuments,
  } = useCreateKnowledgeDocumentPageStore();

  const onDrop = useCallback(acceptedFiles => {
    if (acceptedFiles.length) {
      for (let i = 0; acceptedFiles && i < acceptedFiles?.length; i++) {
        const fl = acceptedFiles[i];
        if (fl) {
          const reader = new FileReader();
          reader.onload = () => {
            onAddKnowledgeDocument({
              file: new Uint8Array(reader.result as ArrayBuffer),
              type: fl.type,
              size: fl.size,
              name: fl.name,
            });
          };
          reader.readAsArrayBuffer(fl);
        }
      }
    }
  }, []);

  const { getRootProps, getInputProps, open } = useDropzone({
    onDrop,
    maxFiles: maxFiles,
    accept: accepts,
    multiple: multiple,
    noClick: false,
  });

  return (
    <div
      className="flex items-center justify-center w-full flex-1"
      {...getRootProps()}
    >
      <label
        htmlFor="datasourceFile"
        onClick={() => {
          open();
        }}
        className="flex flex-col rounded-xl w-full py-4 group text-center"
      >
        <div className="h-full w-full text-center flex flex-col justify-center items-center px-4 space-y-4 min-h-[400px]">
          <div className="p-4 border rounded-[4px] bg-surface/10 backdrop-blur-sm">
            <Upload className="h-8 w-8 flex-no-shrink opacity-60" />
          </div>
          <p className="cursor-pointer opacity-70 text-base font-medium">
            <span className="underline text-support-info ">Drag and drop</span>{' '}
            files here <br /> or select a file from your computer
          </p>
          <div className="flex flex-col space-y-1 w-full overflow-auto max-h-[200px]">
            {knowledgeDocuments.map((knowledgeDocument, idx) => {
              return (
                <SingleRowWrapper
                  className="text-sm font-medium group w-full pl-2 bg-surface "
                  onClick={e => e.preventDefault()}
                  key={`kd-${idx}`}
                >
                  <div className="flex items-center">
                    <Document className="w-5 h-5" />
                    <span className="inline-block pl-2 font-medium">
                      {knowledgeDocument.name}
                    </span>
                    <span>
                      <DotMark className="w-5 h-5 opacity-75" />
                    </span>
                    <span className="inline-block pl-2 font-medium">
                      {formatFileSize(knowledgeDocument.size)}
                    </span>
                  </div>
                  <GhostButton
                    size="md"
                    onClick={() => {
                      onRemoveKnowledgeDocument(knowledgeDocument.name);
                    }}
                  >
                    <Close className="w-4 h-4" />
                  </GhostButton>
                </SingleRowWrapper>
              );
            })}
          </div>
        </div>
        <input
          className="hidden"
          type="file"
          multiple
          {...getInputProps()}
          name="datasourceFile"
          id="datasourceFile"
        />
      </label>
    </div>
  );
};
