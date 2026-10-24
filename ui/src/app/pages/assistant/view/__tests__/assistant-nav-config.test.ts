import { SourceControl } from '@carbon/icons-react';
import { assistantNavSections } from '@/app/pages/assistant/view/assistant-nav-config';

describe('assistantNavSections', () => {
  it('uses source control icon for assistant versions navigation', () => {
    const versionsItem = assistantNavSections
      .flatMap(section => section.items)
      .find(item => item.key === 'versions');

    expect(versionsItem?.icon).toBe(SourceControl);
  });

  it('exposes chat and phone as native playground links', () => {
    const playground = assistantNavSections.find(
      section => section.label === 'Playground',
    );

    expect(playground?.items).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'debugging',
          label: 'Chat',
          path: 'playground/chat',
        }),
        expect.objectContaining({
          key: 'phone-preview',
          label: 'Phone call',
          path: 'playground/phone',
        }),
      ]),
    );
    expect(playground?.items.every(item => !item.action)).toBe(true);
  });
});
