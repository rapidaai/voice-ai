const electron = require('electron');
const { app, BrowserWindow, ipcMain } = require('electron');
const path = require('path');
const url = require('url');

let mainWindow;
process.env.IS_ELECTRON = 'true';
function createWindow() {
  const { width, height } = electron.screen.getPrimaryDisplay().workAreaSize;
  mainWindow = new BrowserWindow({
    width: width,
    height: height,
    title: 'Rapida Talk',
    frame: false,
    resizable: true,
    autoHideMenuBar: true,
    icon: path.join(__dirname, 'favicon_io', 'rapida.ico'), // Use path.join for cross-platform compatibility
    webPreferences: {
      preload: path.join(__dirname, './preload.js'),
      nodeIntegration: false, // Consider changing this to false for better security
      contextIsolation: true, // Consider changing this to true for better security
    },
  });

  let indexPath;

  console.dir(`NODE PROCESS : ${process.env.NODE_ENV}`);
  if (process.env.NODE_ENV !== 'development') {
    indexPath = url.format({
      protocol: 'file:',
      pathname: path.join(__dirname, '../build', 'index.html'),
      slashes: true,
    });
  } else {
    mainWindow.webContents.openDevTools();
    indexPath = 'http://localhost:3000';
  }

  // mainWindow.loadURL('http://localhost:3000');
  mainWindow.loadURL(indexPath);
  // Set a global variable to check if running in Electron
  mainWindow.webContents.on('did-finish-load', () => {
    mainWindow.webContents.executeJavaScript(`
      window.isElectron = true;
    `);
  });

  // Handle window closed event
  mainWindow.on('closed', () => {
    app.quit();
    // mainWindow = null;
  });

  // Handle IPC messages
  ipcMain.on('minimize-window', () => {
    mainWindow.minimize();
  });

  ipcMain.on('maximize-window', () => {
    if (mainWindow.isMaximized()) {
      mainWindow.unmaximize();
    } else {
      mainWindow.maximize();
    }
  });
  ipcMain.on('toggle-fullscreen', () => {
    const isFullscreen = mainWindow.isFullScreen();
    mainWindow.setFullScreen(!isFullscreen);
    mainWindow.webContents.send('fullscreen-changed', !isFullscreen);
  });

  // Send initial fullscreen state
  mainWindow.webContents.on('did-finish-load', () => {
    mainWindow.webContents.send(
      'fullscreen-changed',
      mainWindow.isFullScreen(),
    );
  });

  ipcMain.on('close-window', () => {
    mainWindow.close();
  });

  // Intercept file navigation to handle all routes with React Router

  mainWindow.webContents.on('will-navigate', (event, url) => {
    // Prevent Electron from loading the new route and instead route it through React Router
    event.preventDefault();
    mainWindow.loadURL(
      url.format({
        protocol: 'file:',
        pathname: path.join(__dirname, '../build', 'index.html'),
        slashes: true,
      }),
    );
  });

  mainWindow.webContents.on('new-window', (event, url) => {
    // Prevent Electron from opening a new window when clicking on a link
    event.preventDefault();
    mainWindow.loadURL(
      url.format({
        protocol: 'file:',
        pathname: path.join(__dirname, '../build', 'index.html'),
        slashes: true,
      }),
    );
  });

  //
  if (process.env.NODE_ENV === 'production') {
    mainWindow.webContents.session.webRequest.onHeadersReceived(
      (details, callback) => {
        callback({
          responseHeaders: {
            ...details.responseHeaders,
            'Content-Security-Policy': [
              "default-src 'self'; font-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com; style-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com 'unsafe-inline'; script-src 'self' https://www.googletagmanager.com https://tag.clearbitscripts.com 'unsafe-inline'; worker-src 'self' blob:; connect-src 'self' https://sentry.io https://o4506771747831808.ingest.sentry.io https://api.rapida.ai https://rapida.ai; img-src *",
            ],
          },
        });
      },
    );
  }
}

// App ready event to create window
app.on('ready', createWindow);

// Quit app when all windows are closed, except on macOS
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// Re-create window when the app is activated (macOS)
app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  }
});
