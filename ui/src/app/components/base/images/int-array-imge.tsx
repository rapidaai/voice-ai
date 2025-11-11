import React, { useState, useEffect, HTMLAttributes } from 'react';

interface IntArrayImageProps extends HTMLAttributes<HTMLImageElement> {
  imageData: Uint8Array;
}

export const IntArrayImage: React.FC<IntArrayImageProps> = ({
  imageData,
  className,
}) => {
  const [imageSrc, setImageSrc] = useState<string | null>(null);

  useEffect(() => {
    if (imageData) {
      // Create a Blob from the Uint8Array
      const blob = new Blob([imageData], { type: 'image/png' }); // Adjust type if necessary

      // Create a URL for the Blob
      const imageUrl = URL.createObjectURL(blob);

      setImageSrc(imageUrl);

      // Clean up the URL when the component unmounts
      return () => {
        URL.revokeObjectURL(imageUrl);
      };
    }
  }, [imageData]);

  if (!imageSrc) {
    return <div>Loading image...</div>;
  }

  return <img src={imageSrc} className={className} />;
};
