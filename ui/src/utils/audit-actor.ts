export interface AuditActorLike {
  getType(): string;
  getId(): string;
  getDisplayname(): string;
}

interface AuditActorOwnerLike {
  getCreatedactor?: () => AuditActorLike | undefined;
}

export function createdAuditActor(owner?: unknown): AuditActorLike | undefined {
  const getter = (owner as AuditActorOwnerLike | undefined)?.getCreatedactor;
  return typeof getter === 'function' ? getter.call(owner) : undefined;
}

export function auditActorLabel(actor?: AuditActorLike): string {
  if (!actor) {
    return '—';
  }
  const displayName = actor.getDisplayname().trim();
  if (displayName) {
    return displayName;
  }
  const type = actor.getType().trim();
  const id = actor.getId().trim();
  return type && id ? `${type}:${id}` : '—';
}
