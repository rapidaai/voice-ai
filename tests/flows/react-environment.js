global.window = globalThis;
global.self = globalThis;
global.screen = { width: 1920, height: 1080, colorDepth: 24 };
global.innerWidth = 1920;
global.innerHeight = 1080;
global.location = { href: "http://localhost/" };
global.document = { referrer: "" };
Object.defineProperties(global.navigator, {
  cookieEnabled: { configurable: true, value: false },
  doNotTrack: { configurable: true, value: null },
  hardwareConcurrency: { configurable: true, value: 1 },
  language: { configurable: true, value: "en-US" },
  platform: { configurable: true, value: "node" },
  userAgent: { configurable: true, value: "node" },
});
