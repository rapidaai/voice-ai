const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('api', {
  minimizeWindow: () => ipcRenderer.send('minimize-window'),
  maximizeWindow: () => ipcRenderer.send('maximize-window'),
  closeWindow: () => ipcRenderer.send('close-window'),
  toggleFullscreen: () => ipcRenderer.send('toggle-fullscreen'),
});

// Check if running in Electron
contextBridge.exposeInMainWorld('isElectron', {
  isElectron: true,
});

ipcRenderer.on('fullscreen-changed', (event, isFullscreen) => {
  window.isFullscreen = isFullscreen;
});
