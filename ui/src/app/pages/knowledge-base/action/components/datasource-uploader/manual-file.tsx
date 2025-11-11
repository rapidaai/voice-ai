import { SimpleButton } from '@/app/components/Form/Button';
import { CloseIcon } from '@/app/components/Icon/Close';
import { FileExtensionIcon } from '@/app/components/Icon/file-extension';
import { FileUploadIcon } from '@/app/components/Icon/file-upload';
import { SingleDotIcon } from '@/app/components/Icon/single-dot';
import SingleRowWrapper from '@/app/components/Wrapper/single-row-wrapper';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import { formatFileSize } from '@/utils/format';
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

  const { getRootProps, getInputProps, isDragActive, open } = useDropzone({
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
          <div className="p-4 border rounded-[4px] bg-white/10 backdrop-blur-sm">
            <FileUploadIcon className="h-8 w-8 flex-no-shrink opacity-60" />
          </div>
          <p className="cursor-pointer opacity-70 text-base font-medium">
            <span className="underline text-blue-600 dark:text-blue-400">
              Drag and drop
            </span>{' '}
            files here <br /> or select a file from your computer
          </p>
          <div className="flex flex-col space-y-1 w-full overflow-auto max-h-[200px]">
            {knowledgeDocuments.map((knowledgeDocument, idx) => {
              return (
                <SingleRowWrapper
                  className="text-sm font-medium group w-full pl-2 bg-white dark:bg-gray-950"
                  onClick={e => e.preventDefault()}
                  key={`kd-${idx}`}
                >
                  <div className="flex items-center">
                    <FileExtensionIcon filename={knowledgeDocument.name} />
                    <span className="inline-block pl-2 font-medium">
                      {knowledgeDocument.name}
                    </span>
                    <span>
                      <SingleDotIcon />
                    </span>
                    <span className="inline-block pl-2 font-medium">
                      {formatFileSize(knowledgeDocument.size)}
                    </span>
                  </div>
                  <SimpleButton
                    onClick={() => {
                      onRemoveKnowledgeDocument(knowledgeDocument.name);
                    }}
                  >
                    <CloseIcon className="w-4 h-4" />
                  </SimpleButton>
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
