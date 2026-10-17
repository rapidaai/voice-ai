import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import test from 'node:test';

const require = createRequire(import.meta.url);
const {
  checkJourneyDecisions,
  checkSubjourneyDecisions,
} = require('./check-e2e-route-coverage.cjs');

test('accepts a journey with all route check decisions', () => {
  const result = checkJourneyDecisions([
    {
      id: 'dashboard.home',
      checks: ['smoke', 'a11y'],
      screenshotDeferredReason: 'The layout is covered by another journey.',
    },
  ]);

  assert.deepEqual(result, {
    journeysWithoutSmoke: [],
    journeysWithoutA11yDecision: [],
    journeysWithoutScreenshotDecision: [],
    journeysWithUnknownChecks: [],
    journeysWithConflictingDecisions: [],
  });
});

test('reports missing, unknown, and conflicting route check decisions', () => {
  const result = checkJourneyDecisions([
    {
      id: 'dashboard.home',
      checks: ['visual'],
    },
    {
      id: 'account.settings',
      checks: ['smoke', 'a11y', 'screenshot'],
      a11yDeferredReason: 'Stale decision.',
      screenshotDeferredReason: 'Stale decision.',
    },
  ]);

  assert.deepEqual(result, {
    journeysWithoutSmoke: ['dashboard.home'],
    journeysWithoutA11yDecision: ['dashboard.home'],
    journeysWithoutScreenshotDecision: ['dashboard.home'],
    journeysWithUnknownChecks: ['dashboard.home:visual'],
    journeysWithConflictingDecisions: [
      'account.settings:a11y',
      'account.settings:screenshot',
    ],
  });
});

test('accepts covered, deferred, and disabled subjourney decisions', () => {
  const result = checkSubjourneyDecisions(
    [
      {
        id: 'dashboard.loading',
        parentJourney: 'dashboard.home',
        status: 'covered',
        evidence: 'journeys/dashboard/state-screenshots.spec.js',
      },
      {
        id: 'preview.chat',
        parentJourney: 'dashboard.home',
        status: 'deferred',
        reason: 'Realtime fixtures are required.',
      },
      {
        id: 'knowledge.list',
        status: 'disabled',
        reason: 'Knowledge is disabled.',
      },
    ],
    new Set(['dashboard.home']),
  );

  assert.deepEqual(result, {
    duplicateSubjourneys: [],
    subjourneysWithUnknownStatus: [],
    subjourneysWithUnknownParent: [],
    coveredSubjourneysWithoutEvidence: [],
    deferredSubjourneysWithoutReason: [],
  });
});

test('reports incomplete subjourney decisions', () => {
  const result = checkSubjourneyDecisions(
    [
      { id: 'duplicate', status: 'covered' },
      {
        id: 'duplicate',
        parentJourney: 'missing',
        status: 'waiting',
      },
      { id: 'disabled', status: 'disabled' },
    ],
    new Set(['dashboard.home']),
  );

  assert.deepEqual(result, {
    duplicateSubjourneys: ['duplicate'],
    subjourneysWithUnknownStatus: ['duplicate'],
    subjourneysWithUnknownParent: ['duplicate -> missing'],
    coveredSubjourneysWithoutEvidence: ['duplicate'],
    deferredSubjourneysWithoutReason: ['duplicate', 'disabled'],
  });
});
