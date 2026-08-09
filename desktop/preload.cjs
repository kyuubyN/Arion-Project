// SPDX-License-Identifier: GPL-3.0-only

'use strict';

const { contextBridge, ipcRenderer } = require('electron');

const validPlatforms = new Set(['youtube', 'tiktok']);
const validCommands = new Set(['back', 'forward', 'reload', 'home']);

contextBridge.exposeInMainWorld('arionDesktop', Object.freeze({
  available: true,
  openWebPlatform(platform, bounds) {
    if (!validPlatforms.has(platform)) return Promise.reject(new Error('Plataforma inválida'));
    return ipcRenderer.invoke('arion:web:open', { platform, bounds });
  },
  hideWebView() {
    return ipcRenderer.invoke('arion:web:hide');
  },
  resizeWebView(bounds) {
    return ipcRenderer.invoke('arion:web:resize', bounds);
  },
  navigateWeb(command) {
    if (!validCommands.has(command)) return Promise.reject(new Error('Comando inválido'));
    return ipcRenderer.invoke('arion:web:navigate', command);
  },
  setSessionMode(mode) {
    if (mode !== 'private' && mode !== 'persistent') return Promise.reject(new Error('Modo inválido'));
    return ipcRenderer.invoke('arion:web:session-mode', mode);
  },
  clearWebData(platform) {
    if (!validPlatforms.has(platform)) return Promise.reject(new Error('Plataforma inválida'));
    return ipcRenderer.invoke('arion:web:clear-data', platform);
  },
  onWebState(callback) {
    if (typeof callback !== 'function') return () => {};
    const listener = (_event, state) => callback(state);
    ipcRenderer.on('arion:web:state', listener);
    return () => ipcRenderer.removeListener('arion:web:state', listener);
  }
}));
