const providerAssets = import.meta.glob('./*/*.json', {
  eager: true,
  import: 'default',
});

export function getProviderAsset<T = any>(
  provider: string,
  filename: string,
): T | null {
  return (providerAssets[`./${provider}/${filename}`] as T | undefined) ?? null;
}
