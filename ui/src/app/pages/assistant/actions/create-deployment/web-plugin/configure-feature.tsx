import { InputCheckbox } from '@/app/components/ui/primitives';
import { FormLabel } from '@/app/components/ui/primitives';
import { InputHelper } from '@/app/components/ui/primitives';
import { Text } from '@/app/components/ui/primitives/text';

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
    const featureMap: Record<string, keyof typeof config> = {
      q_a_listing: 'qAListing',
      product_catalog: 'productCatalog',
      blog_post: 'blogPost',
    };

    const stateKey = featureMap[name];
    if (stateKey) {
      onConfigChange({ ...config, [stateKey]: checked });
    }
  };

  return (
    <div className="border-b border-gray-200 dark:border-gray-800">
      <div className="flex flex-col gap-6 max-w-4xl px-6 py-8">
        <fieldset className="flex flex-col space-y-2 min-w-0 border-0 p-0 m-0">
          <FormLabel>Sections</FormLabel>
          <InputHelper>
            Each section offers different features and content in the web
            widget.
          </InputHelper>
          <div className="flex flex-col gap-3 pt-2">
            <label
              htmlFor="q_a_listing"
              className="flex items-center gap-3 cursor-pointer"
            >
              <InputCheckbox
                name="q_a_listing"
                id="q_a_listing"
                checked={config.qAListing}
                onChange={handleCheckboxChange}
              />
              <Text as="span" variant="body-compact-01">
                Help center / Q&A Listing
              </Text>
            </label>
            <label
              htmlFor="product_catalog"
              className="flex items-center gap-3 cursor-pointer"
            >
              <InputCheckbox
                name="product_catalog"
                id="product_catalog"
                checked={config.productCatalog}
                onChange={handleCheckboxChange}
              />
              <Text as="span" variant="body-compact-01">
                Product Catalog
              </Text>
            </label>
            <label
              htmlFor="blog_post"
              className="flex items-center gap-3 cursor-pointer"
            >
              <InputCheckbox
                name="blog_post"
                id="blog_post"
                checked={config.blogPost}
                onChange={handleCheckboxChange}
              />
              <Text as="span" variant="body-compact-01">
                Blog Post / Articles
              </Text>
            </label>
          </div>
        </fieldset>
      </div>
    </div>
  );
};
