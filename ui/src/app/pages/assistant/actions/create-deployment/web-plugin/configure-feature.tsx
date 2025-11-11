import { InputCheckbox } from '@/app/components/Form/Checkbox';
import { Label } from '@/app/components/Form/Label';
import { InputGroup } from '@/app/components/input-group';
import { cn } from '@/styles/media';
import { useEffect, useState } from 'react';

export interface FeatureConfig {
  qAListing: boolean;
  productCatalog: boolean;
  blogPost: boolean;
}

interface ConfigureFeatureProps {
  onConfigChange: (config: FeatureConfig) => void;
  config: FeatureConfig;
}

export const ConfigureFeature: React.FC<ConfigureFeatureProps> = ({
  onConfigChange,
  config,
}) => {
  const handleCheckboxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, checked } = e.target;
    // Map HTML form names to state property names
    const featureMap: Record<string, keyof typeof config> = {
      q_a_listing: 'qAListing',
      product_catalog: 'productCatalog',
      blog_post: 'blogPost',
    };

    const stateKey = featureMap[name];
    if (stateKey) {
      onConfigChange({
        ...config,
        [stateKey]: checked,
      });
    }
  };

  return (
    <InputGroup title="Agent Features" initiallyExpanded={false}>
      <div className={cn('p-6 flex gap-8')}>
        <div className="grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2 flex-1 grow">
          <div className="block flex-1 md:col-span-2">
            <Label className="mb-3">Sections</Label>
            <div className="-mt-1.5 mb-3 text-sm text-gray-500">
              Each section offers different features and contents in the web
              widget.
            </div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <InputCheckbox
                  name="q_a_listing"
                  id="q_a_listing"
                  checked={config.qAListing}
                  onChange={handleCheckboxChange}
                />
                <Label className="text-[0.9rem]" for="q_a_listing">
                  Help center / Q&A Listing
                </Label>
              </div>
              <div className="flex items-center gap-2">
                <InputCheckbox
                  name="product_catalog"
                  id="product_catalog"
                  checked={config.productCatalog}
                  onChange={handleCheckboxChange}
                />
                <Label className="text-[0.9rem]" for="product_catalog">
                  Product Catalog
                </Label>
              </div>
              <div className="flex items-center gap-2">
                <InputCheckbox
                  name="blog_post"
                  id="blog_post"
                  checked={config.blogPost}
                  onChange={handleCheckboxChange}
                />
                <Label className="text-[0.9rem]" for="blog_post">
                  Blog Post / Articles
                </Label>
              </div>
            </div>
          </div>
        </div>
      </div>
    </InputGroup>
  );
};
