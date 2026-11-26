import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateProjectToken } from './create-project-token';
import { createProjectPublishableCredential, listAccessRoles } from '@/clients';

jest.mock('@/clients', () => ({
  createProjectPublishableCredential: jest.fn(),
  listAccessRoles: jest.fn(),
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    renderIcon: Icon,
    iconDescription: _iconDescription,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement> & {
    renderIcon?: React.ComponentType;
    iconDescription?: string;
  }) => (
    <button {...props}>
      {children}
      {Icon && <Icon />}
    </button>
  ),
  InlineLoading: ({ description }: { description: string }) => (
    <p>{description}</p>
  ),
  ComboBox: function MockComboBox({
    disabled,
    items,
    itemToString,
    onChange,
    placeholder,
    selectedItem,
    titleText,
  }: any) {
    return (
      <label>
        {titleText}
        <input
          role="combobox"
          aria-controls="project-role-options"
          aria-expanded="false"
          disabled={disabled}
          placeholder={placeholder}
          defaultValue={selectedItem ? itemToString(selectedItem) : ''}
          onChange={event => {
            const nextValue = event.target.value;
            onChange({
              selectedItem:
                items.find((item: any) => itemToString(item) === nextValue) ||
                null,
            });
          }}
        />
      </label>
    );
  },
  PasswordInput: ({
    'aria-labelledby': ariaLabelledBy,
    id,
    hideLabel,
    labelText,
    value,
  }: {
    'aria-labelledby'?: string;
    id: string;
    hideLabel?: boolean;
    labelText: string;
    value: string;
  }) => (
    <>
      {!hideLabel && <label htmlFor={id}>{labelText}</label>}
      <input
        id={id}
        type="password"
        readOnly
        value={value}
        aria-labelledby={ariaLabelledBy}
      />
    </>
  ),
  Toggletip: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
  ToggletipButton: ({ label }: { label: string }) => (
    <button aria-label={label} type="button" />
  ),
  ToggletipContent: () => null,
}));

jest.mock('@/app/components/ui/primitives', () => ({
  ComboBox: function MockComboBox({
    disabled,
    items,
    itemToString,
    onChange,
    placeholder,
    selectedItem,
    titleText,
  }: any) {
    return (
      <label>
        {titleText}
        <input
          role="combobox"
          aria-label={titleText}
          disabled={disabled}
          placeholder={placeholder}
          defaultValue={selectedItem ? itemToString(selectedItem) : ''}
          onChange={event => {
            const nextValue = event.target.value;
            onChange({
              selectedItem:
                items.find((item: any) => itemToString(item) === nextValue) ||
                null,
            });
          }}
        />
      </label>
    );
  },
  FieldSet: ({ children }: { children: React.ReactNode }) => (
    <fieldset>{children}</fieldset>
  ),
  Modal: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div role="dialog">{children}</div> : null,
  ModalHeader: ({
    onClose,
    title,
  }: {
    onClose?: () => void;
    title: string;
  }) => (
    <>
      <h2>{title}</h2>
      {onClose && (
        <button aria-label="Close modal" type="button" onClick={onClose} />
      )}
    </>
  ),
  ModalBody: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  ModalFooter: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  FormLabel: ({
    children,
    htmlFor,
    ...props
  }: React.LabelHTMLAttributes<HTMLLabelElement>) => (
    <label htmlFor={htmlFor} {...props}>
      {children}
    </label>
  ),
  Input: ({
    decorator,
    hideLabel,
    id,
    labelText,
    ...props
  }: React.InputHTMLAttributes<HTMLInputElement> & {
    decorator?: React.ReactNode;
    hideLabel?: boolean;
    labelText?: React.ReactNode;
  }) => (
    <>
      {!hideLabel && <label htmlFor={id}>{labelText}</label>}
      {decorator}
      <input id={id} {...props} />
    </>
  ),
  PrimaryButton: ({
    children,
    iconDescription: _iconDescription,
    renderIcon: Icon,
    ...props
  }: any) => (
    <button {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  TertiaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  Text: ({ as: Tag = 'span', children, variant: _variant, ...props }: any) => (
    <Tag {...props}>{children}</Tag>
  ),
}));

jest.mock('@/app/components/ui/composites', () => ({
  HelpToggletip: ({ helpText, label }: any) =>
    helpText ? (
      <span>
        <button aria-label={`${label} information`} type="button" />
        {helpText}
      </span>
    ) : null,
}));

const role = (id: string, name: string, key: string) => ({
  getId: () => id,
  getKey: () => key,
  getName: () => name,
});

const roleListResponse = (roles: any[], total: number) => ({
  getSuccess: () => true,
  getRolesList: () => roles,
  getPaginated: () => ({ getTotalitem: () => total }),
});

describe('CreateProjectToken', () => {
  const originalClipboard = navigator.clipboard;

  beforeEach(() => {
    jest.resetAllMocks();
    Object.assign(navigator, { clipboard: { writeText: jest.fn() } });
  });

  afterAll(() => {
    Object.assign(navigator, { clipboard: originalClipboard });
  });

  it('loads every role page, searches roles, and submits the selected role', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const writeText = jest.spyOn(navigator.clipboard, 'writeText');
    const runtime = role('114', 'Runtime Invoker', 'runtime_invoker');
    const observer = role('116', 'Observer Client', 'observer_client');
    (listAccessRoles as jest.Mock)
      .mockResolvedValueOnce(roleListResponse([runtime], 2))
      .mockResolvedValueOnce(roleListResponse([observer], 2));
    (createProjectPublishableCredential as jest.Mock).mockImplementation(
      ({ callback }) =>
        callback(null, {
          getSuccess: () => true,
          getData: () => ({ getKey: () => 'rpx_project_key' }),
        }),
    );
    const onCreated = jest.fn();
    const onClose = jest.fn();

    render(
      <CreateProjectToken
        open
        projectId="project-1"
        auth={{
          organizationId: 'organization-1',
          token: 'token',
          userId: 'user-1',
        }}
        onCreated={onCreated}
        onClose={onClose}
      />,
    );

    expect(screen.getByText('Loading project roles')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Create a project-scoped API key for an application or service. Access is governed by the assigned project role.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Close modal' }),
    ).toBeInTheDocument();
    const select = await screen.findByRole('combobox', {
      name: 'Project Role',
    });
    expect(select).toHaveAttribute('placeholder', 'Search or select a role');
    expect(listAccessRoles).toHaveBeenCalledTimes(2);
    expect(
      screen.getByRole('button', { name: 'Create API key' }),
    ).toBeDisabled();

    await user.type(select, 'Observer Client (observer_client)');
    expect(
      screen.queryByRole('region', { name: 'Role permission preview' }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Create API key' }),
    ).toBeDisabled();
    expect(
      screen.getByRole('button', { name: 'Application name information' }),
    ).toBeInTheDocument();
    await user.type(screen.getByLabelText('Application Name'), 'Widget');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));

    expect(createProjectPublishableCredential).toHaveBeenCalledWith(
      expect.objectContaining({
        projectId: 'project-1',
        name: 'Widget',
        roleDefinitionId: '116',
      }),
    );
    expect(onCreated).toHaveBeenCalledTimes(1);
    expect(onClose).not.toHaveBeenCalled();
    expect(
      screen.getByRole('heading', { name: 'Create API key' }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('API Key')).toHaveValue('rpx_project_key');
    expect(
      screen.getByRole('button', { name: 'API key security information' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'This API key grants the access defined by the assigned project role.',
      ),
    ).toBeInTheDocument();
    expect(screen.getByText('API key created')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Copy' }));
    expect(writeText).toHaveBeenCalledWith('rpx_project_key');
    expect(
      await screen.findByRole('button', { name: 'Copied' }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Close' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('distinguishes roles with the same name by their immutable key', async () => {
    const user = userEvent.setup();
    const readOnly = role('116', 'Support Agent', 'support_agent_read');
    const manager = role('117', 'Support Agent', 'support_agent_manage');
    (listAccessRoles as jest.Mock).mockResolvedValue(
      roleListResponse([readOnly, manager], 2),
    );
    (createProjectPublishableCredential as jest.Mock).mockImplementation(
      ({ callback }) =>
        callback(null, {
          getSuccess: () => true,
          getData: () => ({ getKey: () => 'rpx_project_key' }),
        }),
    );

    render(
      <CreateProjectToken
        open
        projectId="project-1"
        auth={{
          organizationId: 'organization-1',
          token: 'token',
          userId: 'user-1',
        }}
        onCreated={jest.fn()}
        onClose={jest.fn()}
      />,
    );

    await user.type(
      await screen.findByRole('combobox'),
      'Support Agent (support_agent_manage)',
    );
    await user.type(screen.getByLabelText('Application Name'), 'Support app');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));

    expect(createProjectPublishableCredential).toHaveBeenCalledWith(
      expect.objectContaining({ roleDefinitionId: '117' }),
    );
  });

  it('retries a role catalog failure', async () => {
    const user = userEvent.setup();
    const runtime = role('114', 'Runtime Invoker', 'runtime_invoker');
    (listAccessRoles as jest.Mock)
      .mockRejectedValueOnce(new Error('Role catalog unavailable.'))
      .mockResolvedValue(roleListResponse([runtime], 1));

    render(
      <CreateProjectToken
        open
        projectId="project-1"
        auth={{
          organizationId: 'organization-1',
          token: 'token',
          userId: 'user-1',
        }}
        onCreated={jest.fn()}
        onClose={jest.fn()}
      />,
    );

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Role catalog unavailable.',
    );
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(await screen.findByRole('combobox')).toHaveAttribute(
      'placeholder',
      'Search or select a role',
    );
  });

  it('shows the empty state when no project role is available', async () => {
    (listAccessRoles as jest.Mock).mockResolvedValue(roleListResponse([], 0));

    render(
      <CreateProjectToken
        open
        projectId="project-1"
        auth={{
          organizationId: 'organization-1',
          token: 'token',
          userId: 'user-1',
        }}
        onCreated={jest.fn()}
        onClose={jest.fn()}
      />,
    );

    expect(
      await screen.findByText(
        'No active project roles are available for API access.',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Create API key' }),
    ).toBeDisabled();
  });

  it.each([
    ['transport', new Error('Credential service unavailable.'), null],
    [
      'response',
      null,
      {
        getSuccess: () => false,
        getError: () => ({
          getHumanmessage: () => 'The selected role is no longer active.',
        }),
      },
    ],
  ])(
    'keeps the form open after a %s creation failure',
    async (_, error, response) => {
      const user = userEvent.setup();
      const runtime = role('114', 'Runtime Invoker', 'runtime_invoker');
      (listAccessRoles as jest.Mock).mockResolvedValue(
        roleListResponse([runtime], 1),
      );
      (createProjectPublishableCredential as jest.Mock).mockImplementation(
        ({ callback }) => callback(error, response),
      );
      const onCreated = jest.fn();
      const onClose = jest.fn();

      render(
        <CreateProjectToken
          open
          projectId="project-1"
          auth={{
            organizationId: 'organization-1',
            token: 'token',
            userId: 'user-1',
          }}
          onCreated={onCreated}
          onClose={onClose}
        />,
      );

      await user.type(
        await screen.findByRole('combobox'),
        'Runtime Invoker (runtime_invoker)',
      );
      await user.type(screen.getByLabelText('Application Name'), 'Widget');
      await user.click(screen.getByRole('button', { name: 'Create API key' }));

      expect(await screen.findByRole('alert')).toHaveTextContent(
        error?.message || 'The selected role is no longer active.',
      );
      expect(onCreated).not.toHaveBeenCalled();
      expect(onClose).not.toHaveBeenCalled();
    },
  );

  it('does not offer another generation when a successful response omits the key', async () => {
    const user = userEvent.setup();
    const runtime = role('114', 'Runtime Invoker', 'runtime_invoker');
    (listAccessRoles as jest.Mock).mockResolvedValue(
      roleListResponse([runtime], 1),
    );
    (createProjectPublishableCredential as jest.Mock).mockImplementation(
      ({ callback }) =>
        callback(null, {
          getSuccess: () => true,
          getData: () => undefined,
        }),
    );
    const onCreated = jest.fn();

    render(
      <CreateProjectToken
        open
        projectId="project-1"
        auth={{
          organizationId: 'organization-1',
          token: 'token',
          userId: 'user-1',
        }}
        onCreated={onCreated}
        onClose={jest.fn()}
      />,
    );

    await user.type(
      await screen.findByRole('combobox'),
      'Runtime Invoker (runtime_invoker)',
    );
    await user.type(screen.getByLabelText('Application Name'), 'Widget');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));

    expect(onCreated).toHaveBeenCalledTimes(1);
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'API key created, but its value was not returned.',
    );
    expect(screen.queryByRole('button', { name: 'Create API key' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Copy' })).toBeDisabled();
  });

  it.each([
    ['asynchronous', () => Promise.reject(new Error('clipboard unavailable'))],
    [
      'synchronous',
      () => {
        throw new Error('clipboard unavailable');
      },
    ],
  ])('reports an %s clipboard failure', async (_, copyKey) => {
    const user = userEvent.setup({ writeToClipboard: false });
    jest.spyOn(navigator.clipboard, 'writeText').mockImplementation(copyKey);
    const runtime = role('114', 'Runtime Invoker', 'runtime_invoker');
    (listAccessRoles as jest.Mock).mockResolvedValue(
      roleListResponse([runtime], 1),
    );
    (createProjectPublishableCredential as jest.Mock).mockImplementation(
      ({ callback }) =>
        callback(null, {
          getSuccess: () => true,
          getData: () => ({ getKey: () => 'rpx_project_key' }),
        }),
    );

    render(
      <CreateProjectToken
        open
        projectId="project-1"
        auth={{
          organizationId: 'organization-1',
          token: 'token',
          userId: 'user-1',
        }}
        onCreated={jest.fn()}
        onClose={jest.fn()}
      />,
    );

    await user.type(
      await screen.findByRole('combobox'),
      'Runtime Invoker (runtime_invoker)',
    );
    await user.type(screen.getByLabelText('Application Name'), 'Widget');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));
    await user.click(screen.getByRole('button', { name: 'Copy' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Unable to copy the API key.',
    );
    expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument();
  });

  it('ignores clipboard completion from an earlier modal session', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    let finishFirstCopy: () => void = () => {};
    jest
      .spyOn(navigator.clipboard, 'writeText')
      .mockImplementationOnce(
        () =>
          new Promise<void>(resolve => {
            finishFirstCopy = resolve;
          }),
      )
      .mockResolvedValue(undefined);
    const runtime = role('114', 'Runtime Invoker', 'runtime_invoker');
    (listAccessRoles as jest.Mock).mockResolvedValue(
      roleListResponse([runtime], 1),
    );
    let creationCount = 0;
    (createProjectPublishableCredential as jest.Mock).mockImplementation(
      ({ callback }) => {
        creationCount += 1;
        callback(null, {
          getSuccess: () => true,
          getData: () => ({
            getKey: () =>
              creationCount === 1 ? 'rpx_first_key' : 'rpx_second_key',
          }),
        });
      },
    );
    const props = {
      projectId: 'project-1',
      auth: {
        organizationId: 'organization-1',
        token: 'token',
        userId: 'user-1',
      },
      onCreated: jest.fn(),
      onClose: jest.fn(),
    };
    const { rerender } = render(<CreateProjectToken {...props} open />);

    await user.type(
      await screen.findByRole('combobox'),
      'Runtime Invoker (runtime_invoker)',
    );
    await user.type(screen.getByLabelText('Application Name'), 'First widget');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));
    await user.click(screen.getByRole('button', { name: 'Copy' }));
    await user.click(screen.getByRole('button', { name: 'Close' }));
    rerender(<CreateProjectToken {...props} open={false} />);
    rerender(<CreateProjectToken {...props} open />);

    await user.type(
      await screen.findByRole('combobox'),
      'Runtime Invoker (runtime_invoker)',
    );
    await user.type(screen.getByLabelText('Application Name'), 'Second widget');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));
    expect(screen.getByLabelText('API Key')).toHaveValue('rpx_second_key');
    finishFirstCopy();

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument();
      expect(screen.queryByRole('alert')).toBeNull();
    });
    await user.click(screen.getByRole('button', { name: 'Copy' }));
    expect(
      await screen.findByRole('button', { name: 'Copied' }),
    ).toBeInTheDocument();
  });

  it('ignores a completed submission from an earlier modal session', async () => {
    const user = userEvent.setup();
    const runtime = role('114', 'Runtime Invoker', 'runtime_invoker');
    const observer = role('116', 'Observer Client', 'observer_client');
    let finishCreation: (
      error: null,
      response: { getSuccess: () => boolean },
    ) => void = () => {};
    (listAccessRoles as jest.Mock).mockResolvedValue(
      roleListResponse([runtime, observer], 2),
    );
    (createProjectPublishableCredential as jest.Mock).mockImplementation(
      ({ callback }) => {
        finishCreation = callback;
      },
    );
    const onCreated = jest.fn();
    const onClose = jest.fn();
    const props = {
      projectId: 'project-1',
      auth: {
        organizationId: 'organization-1',
        token: 'token',
        userId: 'user-1',
      },
      onCreated,
      onClose,
    };
    const { rerender } = render(<CreateProjectToken {...props} open />);

    const select = await screen.findByRole('combobox');
    await user.type(screen.getByLabelText('Application Name'), 'Widget');
    await user.type(select, 'Runtime Invoker (runtime_invoker)');
    await user.click(screen.getByRole('button', { name: 'Create API key' }));
    await user.click(screen.getByRole('button', { name: 'Create API key' }));
    expect(createProjectPublishableCredential).toHaveBeenCalledTimes(1);
    expect(
      screen.getByRole('button', { name: 'Create API key' }),
    ).toBeDisabled();
    expect(select).toBeDisabled();
    expect(select).toHaveValue('Runtime Invoker (runtime_invoker)');

    await user.click(screen.getByRole('button', { name: 'Close' }));
    rerender(<CreateProjectToken {...props} open={false} />);
    rerender(<CreateProjectToken {...props} open />);
    await screen.findByRole('combobox');
    finishCreation(null, { getSuccess: () => true });

    await waitFor(() => {
      expect(onCreated).not.toHaveBeenCalled();
      expect(onClose).toHaveBeenCalledTimes(1);
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });
  });
});
