export function getProviderAsset<T = any>(
  provider: string,
  filename: string,
): T | null {
  try {
    return require(`./${provider}/${filename}`) as T;
  } catch {
    return null;
  }
}
