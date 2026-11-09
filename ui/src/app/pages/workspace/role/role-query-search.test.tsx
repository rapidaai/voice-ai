import { getRoleSearchState } from './role-query-search';

describe('getRoleSearchState', () => {
  it('maps committed query filters to the role request state', () => {
    expect(
      getRoleSearchState(
        'query:support scope:ROLE_SCOPE_PROJECT subject:ROLE_SUBJECT_USER type:ROLE_DEFINITION_TYPE_CUSTOM',
      ),
    ).toEqual({
      definitionType: 'ROLE_DEFINITION_TYPE_CUSTOM',
      query: 'support',
      scope: 'ROLE_SCOPE_PROJECT',
      subject: 'ROLE_SUBJECT_USER',
    });
  });

  it('ignores unsupported enum values', () => {
    expect(
      getRoleSearchState('scope:invalid subject:invalid type:invalid'),
    ).toEqual({
      definitionType: 'ROLE_DEFINITION_TYPE_UNSPECIFIED',
      query: '',
      scope: 'ROLE_SCOPE_UNSPECIFIED',
      subject: 'ROLE_SUBJECT_UNSPECIFIED',
    });
  });
});
