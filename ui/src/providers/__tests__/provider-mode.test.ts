import developmentProviders from '@/providers/provider.development.json';
import productionProviders from '@/providers/provider.production.json';
import { allProvider } from '@/providers';

describe('provider mode selection', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    delete process.env.MODE;
    delete process.env.VITE_MODE;
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  it('uses development providers when the app mode differs from NODE_ENV', () => {
    process.env.MODE = 'development';
    process.env.NODE_ENV = 'production';

    expect(allProvider()).toEqual(developmentProviders);
  });

  it('falls back to NODE_ENV when no app mode is configured', () => {
    process.env.NODE_ENV = 'production';

    expect(allProvider()).toEqual(productionProviders);
  });
});
