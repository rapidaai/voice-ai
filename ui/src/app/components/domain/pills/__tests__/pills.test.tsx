import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { DocumentSourcePill } from '../document-source-pill';
import { ProviderPill } from '../provider-model-pill';
import { ToolProviderPill } from '../tool-provider-pill';

const toolProviders = [
  {
    getId: () => 'tool-1',
    getImage: () => '/tool.svg',
    getName: () => 'Webhook tool',
  },
];

jest.mock('@carbon/icons-react', () => ({
  Globe: () => <svg data-testid="globe-icon" />,
}));

jest.mock('@carbon/react', () => ({
  Tag: ({ children, onClick, size, type }: any) => (
    <span data-size={size} data-type={type} onClick={onClick}>
      {children}
    </span>
  ),
}));

jest.mock('@/providers', () => ({
  allProvider: () => [
    {
      code: 'openai',
      image: '/openai.svg',
      name: 'OpenAI',
    },
  ],
}));

jest.mock('@/context/provider-context', () => ({
  useProviderContext: () => ({
    toolProviders,
  }),
}));

describe('domain pills', () => {
  it('renders provider details from the provider registry', () => {
    render(<ProviderPill provider="OPENAI" />);

    expect(screen.getByText('OpenAI').closest('[data-type]')).toHaveAttribute(
      'data-type',
      'gray',
    );
    expect(screen.getByAltText('OpenAI')).toHaveAttribute('src', '/openai.svg');
  });

  it('falls back to the provider id when the provider is unknown', () => {
    render(
      <>
        <ProviderPill provider="custom-provider" />
        <ProviderPill />
      </>,
    );

    expect(
      screen.getByText('custom-provider').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'gray');
    expect(
      screen.getByText('Unknown provider').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'gray');
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });

  it('renders tool providers from props or context lookup', () => {
    const directTool = {
      getId: () => 'direct-tool',
      getImage: () => '/direct.svg',
      getName: () => 'Direct tool',
    };

    render(
      <>
        <ToolProviderPill toolProviderId="tool-1" />
        <ToolProviderPill toolProvider={directTool} />
        <ToolProviderPill toolProviderId="missing-tool" />
        <ToolProviderPill />
      </>,
    );

    expect(
      screen.getByText('Webhook tool').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'blue');
    expect(screen.getByAltText('Webhook tool')).toHaveAttribute(
      'src',
      '/tool.svg',
    );
    expect(
      screen.getByText('Direct tool').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'blue');
    expect(
      screen.getByText('missing-tool').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'blue');
    expect(
      screen.getByText('Unknown tool').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'blue');
  });

  it('renders document sources and delegates tool sources', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onClick = jest.fn();

    render(
      <>
        <DocumentSourcePill type="manual" onClick={onClick} />
        <DocumentSourcePill source="tool-1" type="tool" />
        <DocumentSourcePill source="upload">Uploaded file</DocumentSourcePill>
        <DocumentSourcePill />
      </>,
    );

    expect(screen.getByText('manual').closest('[data-type]')).toHaveAttribute(
      'data-type',
      'gray',
    );
    expect(screen.getAllByTestId('globe-icon')).toHaveLength(3);
    expect(
      screen.getByText('Webhook tool').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'blue');
    expect(
      screen.getByText('Uploaded file').closest('[data-type]'),
    ).toHaveAttribute('data-type', 'gray');
    expect(screen.getByText('Document').closest('[data-type]')).toHaveAttribute(
      'data-type',
      'gray',
    );

    await user.click(screen.getByText('manual'));
    expect(onClick).toHaveBeenCalled();
  });
});
