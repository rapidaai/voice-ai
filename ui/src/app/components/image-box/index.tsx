import { SimpleButton } from '@/app/components/Form/Button';
import React, { useEffect, useState } from 'react';

export interface ImageBoxProps
  extends React.ImgHTMLAttributes<HTMLImageElement> {
  children?: React.ReactNode;
  type: 'blob' | 'url' | 'intArray';
  dataSrc: string | Uint8Array;
}

export function ImageBox({ type, dataSrc, alt, ...imageAttr }: ImageBoxProps) {
  const expandImage = (type: string, image: string) => {
    const downloadLink = document.createElement('a');

    // Check if the type is 'blob' or undefined (default to original image)
    if (type === 'blob') {
      // If type is 'blob', decode the base64 image data
      const decodedData = window.atob(image);

      // Convert the decoded data to a Uint8Array
      const uint8Array = new Uint8Array(decodedData.length);
      for (let i = 0; i < decodedData.length; i++) {
        uint8Array[i] = decodedData.charCodeAt(i);
      }

      // Create a Blob from the Uint8Array
      const blob = new Blob([uint8Array], {
        type: 'application/octet-stream',
      });

      // Set the download link's href to the Blob's data URL
      downloadLink.href = window.URL.createObjectURL(blob);
    } else {
      // If type is not 'blob' or undefined, use the original image URL
      downloadLink.href = image;
    }

    // Open the link in a new tab
    downloadLink.target = '_blank';

    // Append the link to the document body
    document.body.appendChild(downloadLink);

    // Simulate a click on the link to trigger the download
    downloadLink.click();

    // Remove the link from the document body
    document.body.removeChild(downloadLink);
  };

  const [imageContent, setImageContent] = useState('');
  useEffect(() => {
    if (type === 'blob') {
      setImageContent(
        `data:application/octet-stream;base64,${dataSrc as string}`,
      );
    }

    if (type === 'intArray') {
      console.dir(dataSrc as Uint8Array);
      const blob = new Blob([dataSrc as Uint8Array], { type: 'image/png' });
      setImageContent(URL.createObjectURL(blob));
    }

    // setImageContent(dataSrc as string);
  }, []);
  return (
    <div className="relative">
      <div className="absolute right-0 top-0 space-x-2 p-2 flex">
        <SimpleButton
          onClick={() => {
            expandImage(type, dataSrc as string);
          }}
          className="backdrop-blur-xl bg-white/30 dark:text-gray-100 text-white"
        >
          <span className="sr-only">Expand</span>

          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth="1.5"
            stroke="currentColor"
            className="w-4 h-4"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3"
            />
          </svg>
        </SimpleButton>
      </div>
      <img {...imageAttr} src={imageContent} alt={alt} />
    </div>
  );
}

ImageBox.defaultProps = {
  alt: '',
};
