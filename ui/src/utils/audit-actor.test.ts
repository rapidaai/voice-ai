import {
  auditActorLabel,
  AuditActorLike,
  createdAuditActor,
} from './audit-actor';

function actor(type: string, id: string, displayName = ''): AuditActorLike {
  return {
    getType: () => type,
    getId: () => id,
    getDisplayname: () => displayName,
  };
}

describe('auditActorLabel', () => {
  it('uses the user display name when present', () => {
    expect(auditActorLabel(actor('user', '42', 'Assistant Owner'))).toBe(
      'Assistant Owner',
    );
  });

  it('uses durable type and id for non-user actors', () => {
    expect(auditActorLabel(actor('service', '41'))).toBe('service:41');
  });

  it('uses a neutral fallback for absent or malformed actors', () => {
    expect(auditActorLabel()).toBe('—');
    expect(auditActorLabel(actor('', ''))).toBe('—');
  });

  it('reads actor getters without requiring a generated SDK type upgrade', () => {
    const createdActor = actor('project', '84');
    expect(createdAuditActor({ getCreatedactor: () => createdActor })).toBe(
      createdActor,
    );
    expect(createdAuditActor({})).toBeUndefined();
  });
});
