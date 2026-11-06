const fs = require('fs');
const path = require('path');
const ts = require('typescript');
const {
  routeRoots,
  routeJourneys,
  sourceRouteInventory,
} = require('../e2e/route-manifest');
const { journeySubjourneys } = require('../e2e/journey-catalog');

const appRouteFile = path.join(__dirname, '..', 'src', 'app', 'index.tsx');
const routeDir = path.join(__dirname, '..', 'src', 'app', 'routes');
const e2eDir = path.join(__dirname, '..', 'e2e');
const devConfigFile = path.join(
  __dirname,
  '..',
  'src',
  'configs',
  'config.development.json',
);

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function readAppRouteRoots() {
  const sources = [
    fs.readFileSync(appRouteFile, 'utf8'),
    ...routeSourceFiles().map(fileName =>
      fs.readFileSync(path.join(routeDir, fileName), 'utf8'),
    ),
  ];
  const roots = new Set();
  const objectPathPattern = /\bpath:\s*["'`]([^"'`{}]+)["'`]/g;

  for (const source of sources) {
    objectPathPattern.lastIndex = 0;
    let match;
    while ((match = objectPathPattern.exec(source))) {
      const value = match[1];
      if (value === '/' || value === '*') continue;
      if (value.startsWith('/')) roots.add(value);
    }
  }

  return [...roots].sort();
}

function readStringProperty(object, propertyName) {
  const property = object.properties.find(
    candidate =>
      ts.isPropertyAssignment(candidate) &&
      candidate.name.getText().replace(/["']/g, '') === propertyName,
  );

  if (
    !property ||
    (!ts.isStringLiteral(property.initializer) &&
      !ts.isNoSubstitutionTemplateLiteral(property.initializer))
  ) {
    return undefined;
  }

  return property.initializer.text;
}

function readChildrenProperty(object) {
  const property = object.properties.find(
    candidate =>
      ts.isPropertyAssignment(candidate) &&
      candidate.name.getText().replace(/["']/g, '') === 'children',
  );
  return property?.initializer;
}

function readRouteObjectPaths(source) {
  const sourceFile = ts.createSourceFile(
    'routes.tsx',
    source,
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TSX,
  );
  const paths = [];

  const walkChildren = (expression, parentPath) => {
    if (ts.isParenthesizedExpression(expression)) {
      walkChildren(expression.expression, parentPath);
      return;
    }
    if (ts.isConditionalExpression(expression)) {
      walkChildren(expression.whenTrue, parentPath);
      walkChildren(expression.whenFalse, parentPath);
      return;
    }
    if (!ts.isArrayLiteralExpression(expression)) return;

    for (const element of expression.elements) {
      if (ts.isSpreadElement(element)) {
        walkChildren(element.expression, parentPath);
      } else if (ts.isObjectLiteralExpression(element)) {
        walkRoute(element, parentPath, false);
      }
    }
  };

  const walkRoute = (object, parentPath, isRoot) => {
    const routePath = readStringProperty(object, 'path');
    let currentPath = parentPath;

    if (routePath && routePath !== '*') {
      const segment = routePath.replace(/^\/+|\/+$/g, '');
      currentPath = isRoot
        ? ''
        : [parentPath, segment].filter(Boolean).join('/');
      paths.push(isRoot ? '/' : currentPath);
    }

    const children = readChildrenProperty(object);
    if (children) walkChildren(children, currentPath);
  };

  for (const statement of sourceFile.statements) {
    if (!ts.isVariableStatement(statement)) continue;
    for (const declaration of statement.declarationList.declarations) {
      if (
        declaration.type?.getText(sourceFile) === 'RouteObject' &&
        declaration.initializer &&
        ts.isObjectLiteralExpression(declaration.initializer)
      ) {
        walkRoute(declaration.initializer, '', true);
      }
    }
  }

  return paths;
}

function routeSourceFiles() {
  return fs
    .readdirSync(routeDir)
    .filter(fileName => fileName.endsWith('.tsx'))
    .filter(fileName => fileName !== 'index.tsx')
    .sort();
}

function increment(map, key) {
  map.set(key, (map.get(key) || 0) + 1);
}

function routeKey(fileName, routePath) {
  return `${fileName}:${routePath}`;
}

function readRouteSourceEntries() {
  const entries = new Map();

  for (const fileName of routeSourceFiles()) {
    const source = fs.readFileSync(path.join(routeDir, fileName), 'utf8');
    for (const routePath of readRouteObjectPaths(source)) {
      increment(entries, routeKey(fileName, routePath));
    }
  }

  return entries;
}

function getConfigValue(config, featurePath) {
  return featurePath
    .split('.')
    .reduce((current, key) => (current ? current[key] : undefined), config);
}

function inventoryEntries() {
  const entries = [];

  for (const [fileName, routes] of Object.entries(sourceRouteInventory)) {
    for (const route of routes) {
      entries.push({ fileName, ...route });
    }
  }

  return entries;
}

const allowedJourneyChecks = new Set(['smoke', 'a11y', 'screenshot']);

function checkJourneyDecisions(journeys) {
  const checksFor = journey =>
    Array.isArray(journey.checks) ? journey.checks : [];

  return {
    journeysWithoutSmoke: journeys
      .filter(journey => !checksFor(journey).includes('smoke'))
      .map(journey => journey.id),
    journeysWithoutA11yDecision: journeys
      .filter(
        journey =>
          !checksFor(journey).includes('a11y') && !journey.a11yDeferredReason,
      )
      .map(journey => journey.id),
    journeysWithoutScreenshotDecision: journeys
      .filter(
        journey =>
          !checksFor(journey).includes('screenshot') &&
          !journey.screenshotDeferredReason,
      )
      .map(journey => journey.id),
    journeysWithUnknownChecks: journeys.flatMap(journey =>
      checksFor(journey)
        .filter(check => !allowedJourneyChecks.has(check))
        .map(check => `${journey.id}:${check}`),
    ),
    journeysWithConflictingDecisions: journeys.flatMap(journey => {
      const conflicts = [];
      if (checksFor(journey).includes('a11y') && journey.a11yDeferredReason) {
        conflicts.push(`${journey.id}:a11y`);
      }
      if (
        checksFor(journey).includes('screenshot') &&
        journey.screenshotDeferredReason
      ) {
        conflicts.push(`${journey.id}:screenshot`);
      }
      return conflicts;
    }),
  };
}

function checkSubjourneyDecisions(subjourneys, journeyIds) {
  const allowedStatuses = new Set(['covered', 'deferred', 'disabled']);
  const ids = subjourneys.map(subjourney => subjourney.id);

  return {
    duplicateSubjourneys: ids.filter((id, index) => ids.indexOf(id) !== index),
    subjourneysWithUnknownStatus: subjourneys
      .filter(subjourney => !allowedStatuses.has(subjourney.status))
      .map(subjourney => subjourney.id),
    subjourneysWithUnknownParent: subjourneys
      .filter(
        subjourney =>
          subjourney.parentJourney && !journeyIds.has(subjourney.parentJourney),
      )
      .map(subjourney => `${subjourney.id} -> ${subjourney.parentJourney}`),
    coveredSubjourneysWithoutEvidence: subjourneys
      .filter(
        subjourney => subjourney.status === 'covered' && !subjourney.evidence,
      )
      .map(subjourney => subjourney.id),
    deferredSubjourneysWithoutReason: subjourneys
      .filter(
        subjourney => subjourney.status !== 'covered' && !subjourney.reason,
      )
      .map(subjourney => subjourney.id),
  };
}

function checkRouteCoverage() {
  const appRoots = readAppRouteRoots();
  const config = readJSON(devConfigFile);
  const manifestRoots = routeRoots.map(route => route.root).sort();
  const journeyIds = new Set(routeJourneys.map(journey => journey.id));
  const journeyById = new Map(
    routeJourneys.map(journey => [journey.id, journey]),
  );

  const missingRoots = appRoots.filter(root => !manifestRoots.includes(root));
  const unknownRoots = manifestRoots.filter(root => !appRoots.includes(root));
  const duplicateRoots = manifestRoots.filter(
    (root, index) => manifestRoots.indexOf(root) !== index,
  );
  const duplicateJourneys = routeJourneys
    .map(journey => journey.id)
    .filter((id, index, ids) => ids.indexOf(id) !== index);
  const invalidDisabledRoots = routeRoots
    .filter(route => route.disabledFeature)
    .filter(route => getConfigValue(config, route.disabledFeature) !== false)
    .map(route => route.root);
  const rootsWithoutCoverageDecision = routeRoots
    .filter(
      route =>
        !route.journey && !route.disabledFeature && !route.deferredReason,
    )
    .map(route => route.root);
  const unmappedEnabledRoots = routeRoots
    .filter(route => !route.disabledFeature && !route.deferredReason)
    .filter(route => !route.journey || !journeyIds.has(route.journey))
    .map(route => route.root);
  const misroutedRootJourneys = routeRoots
    .filter(route => route.journey && journeyIds.has(route.journey))
    .filter(route => {
      const journey = journeyById.get(route.journey);
      return (
        journey.path !== route.root &&
        !journey.path.startsWith(`${route.root}/`)
      );
    })
    .map(route => `${route.root} -> ${route.journey}`);
  const journeysWithoutAssertions = routeJourneys
    .filter(
      journey =>
        !journey.path ||
        !journey.expectText ||
        !Array.isArray(journey.checks) ||
        journey.checks.length === 0,
    )
    .map(journey => journey.id);
  const journeyDecisionIssues = checkJourneyDecisions(routeJourneys);
  const subjourneyDecisionIssues = checkSubjourneyDecisions(
    journeySubjourneys,
    journeyIds,
  );
  const coveredSubjourneysWithMissingEvidence = journeySubjourneys
    .filter(subjourney => subjourney.status === 'covered')
    .filter(
      subjourney =>
        !fs.existsSync(path.join(e2eDir, subjourney.evidence || '')),
    )
    .map(subjourney => `${subjourney.id} -> ${subjourney.evidence}`);
  const discoveredRouteSources = readRouteSourceEntries();
  const inventory = inventoryEntries();
  const inventorySourceKeys = new Map(
    inventory.map(route => [
      routeKey(route.fileName, route.path),
      route.count || 1,
    ]),
  );
  const missingSourceRoutes = [...discoveredRouteSources.keys()]
    .filter(key => !inventorySourceKeys.has(key))
    .sort();
  const unknownSourceRoutes = [...inventorySourceKeys.keys()]
    .filter(key => !discoveredRouteSources.has(key))
    .sort();
  const sourceRouteCountMismatches = [...discoveredRouteSources.entries()]
    .filter(([key, count]) => inventorySourceKeys.get(key) !== count)
    .map(
      ([key, count]) =>
        `${key} expected ${inventorySourceKeys.get(key) || 0} found ${count}`,
    );
  const sourceRoutesWithoutDecision = inventory
    .filter(
      route =>
        !route.journey && !route.disabledFeature && !route.deferredReason,
    )
    .map(route => routeKey(route.fileName, route.path));
  const sourceRoutesWithUnknownJourney = inventory
    .filter(route => route.journey && !journeyIds.has(route.journey))
    .map(
      route => `${routeKey(route.fileName, route.path)} -> ${route.journey}`,
    );
  const sourceRoutesWithInvalidDisabledFeature = inventory
    .filter(route => route.disabledFeature)
    .filter(route => getConfigValue(config, route.disabledFeature) !== false)
    .map(route => routeKey(route.fileName, route.path));

  return {
    appRoots,
    manifestRoots,
    missingRoots,
    unknownRoots,
    duplicateRoots,
    duplicateJourneys,
    invalidDisabledRoots,
    rootsWithoutCoverageDecision,
    unmappedEnabledRoots,
    misroutedRootJourneys,
    journeysWithoutAssertions,
    ...journeyDecisionIssues,
    ...subjourneyDecisionIssues,
    coveredSubjourneysWithMissingEvidence,
    missingSourceRoutes,
    unknownSourceRoutes,
    sourceRouteCountMismatches,
    sourceRoutesWithoutDecision,
    sourceRoutesWithUnknownJourney,
    sourceRoutesWithInvalidDisabledFeature,
  };
}

if (require.main === module) {
  const result = checkRouteCoverage();
  const hasFailure =
    result.missingRoots.length > 0 ||
    result.unknownRoots.length > 0 ||
    result.duplicateRoots.length > 0 ||
    result.duplicateJourneys.length > 0 ||
    result.invalidDisabledRoots.length > 0 ||
    result.rootsWithoutCoverageDecision.length > 0 ||
    result.unmappedEnabledRoots.length > 0 ||
    result.misroutedRootJourneys.length > 0 ||
    result.journeysWithoutAssertions.length > 0 ||
    result.journeysWithoutSmoke.length > 0 ||
    result.journeysWithoutA11yDecision.length > 0 ||
    result.journeysWithoutScreenshotDecision.length > 0 ||
    result.journeysWithUnknownChecks.length > 0 ||
    result.journeysWithConflictingDecisions.length > 0 ||
    result.duplicateSubjourneys.length > 0 ||
    result.subjourneysWithUnknownStatus.length > 0 ||
    result.subjourneysWithUnknownParent.length > 0 ||
    result.coveredSubjourneysWithoutEvidence.length > 0 ||
    result.coveredSubjourneysWithMissingEvidence.length > 0 ||
    result.deferredSubjourneysWithoutReason.length > 0 ||
    result.missingSourceRoutes.length > 0 ||
    result.unknownSourceRoutes.length > 0 ||
    result.sourceRouteCountMismatches.length > 0 ||
    result.sourceRoutesWithoutDecision.length > 0 ||
    result.sourceRoutesWithUnknownJourney.length > 0 ||
    result.sourceRoutesWithInvalidDisabledFeature.length > 0;

  console.log(JSON.stringify(result, null, 2));
  process.exit(hasFailure ? 1 : 0);
}

module.exports = {
  checkJourneyDecisions,
  checkSubjourneyDecisions,
  checkRouteCoverage,
  readAppRouteRoots,
  readRouteObjectPaths,
};
