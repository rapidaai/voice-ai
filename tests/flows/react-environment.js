global.window = globalThis;
global.self = globalThis;
global.screen = { width: 1920, height: 1080, colorDepth: 24 };
global.innerWidth = 1920;
global.innerHeight = 1080;
global.location = { href: "http://localhost/" };
global.document = { referrer: "" };
global.navigator = {
  cookieEnabled: false,
  doNotTrack: null,
  hardwareConcurrency: 1,
  language: "en-US",
  platform: "node",
  userAgent: "node",
};
