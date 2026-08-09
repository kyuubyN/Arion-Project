// SPDX-License-Identifier: GPL-3.0-only

'use strict';

const { app, BrowserWindow, WebContentsView, ipcMain, Menu, session } = require('electron');
const { spawn } = require('node:child_process');
const crypto = require('node:crypto');
const http = require('node:http');
const path = require('node:path');
const readline = require('node:readline');

const PLATFORMS = Object.freeze({
  youtube: {
    home: 'https://www.youtube.com/',
    hosts: ['youtube.com', 'youtu.be', 'google.com']
  },
  tiktok: {
    home: 'https://www.tiktok.com/',
    hosts: ['tiktok.com']
  }
});

let mainWindow;
let backendProcess;
let backendPort;
let backendToken;
let activePlatform = null;
let sessionMode = 'private';
let privateSessionNonce = crypto.randomBytes(12).toString('hex');
let activeBounds = { x: 280, y: 210, width: 900, height: 540 };
const webViews = new Map();

app.setName('Arion');
if (process.platform === 'win32') app.setAppUserModelId('io.github.kyuubyn.arion');
if (process.platform === 'linux') {
  app.setDesktopName('arion.desktop');
  app.commandLine.appendSwitch('class', 'arion');
}
app.enableSandbox();

function backendExecutable() {
	const executableName = process.platform === 'win32' ? 'arion-backend.exe' : 'arion-backend';
	if (app.isPackaged) return path.join(process.resourcesPath, 'backend', executableName);
	return path.join(__dirname, '..', 'backend', executableName);
}

function runtimeWorkingDirectory() {
  return app.isPackaged ? process.resourcesPath : path.join(__dirname, '..');
}

function startBackend() {
  backendToken = crypto.randomBytes(32).toString('hex');
  backendProcess = spawn(backendExecutable(), [], {
    cwd: runtimeWorkingDirectory(),
    env: { ...process.env, ARION_PORT: '0', ARION_SESSION_TOKEN: backendToken },
	stdio: ['ignore', 'pipe', 'pipe'],
	windowsHide: true
  });

  backendProcess.stderr.on('data', chunk => {
    const safe = String(chunk).replaceAll(backendToken, '[REDACTED]');
    process.stderr.write(safe);
  });

  return new Promise((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error('O backend do Arion não iniciou a tempo.')), 15000);
    const lines = readline.createInterface({ input: backendProcess.stdout });
    lines.on('line', line => {
      const match = line.match(/^ARION_SERVER_READY_URL=http:\/\/127\.0\.0\.1:(\d+)\//);
      if (!match) return;
      clearTimeout(timeout);
      backendPort = Number(match[1]);
      resolve(`http://127.0.0.1:${backendPort}/#session=${backendToken}`);
    });
    backendProcess.once('error', error => {
      clearTimeout(timeout);
      reject(error);
    });
    backendProcess.once('exit', code => {
      if (!backendPort) {
        clearTimeout(timeout);
        reject(new Error(`O backend encerrou antes de ficar pronto (${code}).`));
      }
    });
  });
}

function createMainWindow(localURL) {
  mainWindow = new BrowserWindow({
    title: 'Arion — Galeria de mídia',
    width: 1280,
    height: 800,
    minWidth: 900,
    minHeight: 620,
    icon: path.join(__dirname, '..', 'assets', 'icon.png'),
    backgroundColor: '#000000',
    show: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: true,
      webSecurity: true,
      allowRunningInsecureContent: false,
      devTools: !app.isPackaged
    }
  });
  Menu.setApplicationMenu(null);
  mainWindow.loadURL(localURL);
  mainWindow.once('ready-to-show', () => mainWindow.show());
  mainWindow.on('closed', () => { mainWindow = null; });
}

function partitionFor(platform) {
  return sessionMode === 'persistent' ? `persist:arion-${platform}` : `arion-${platform}-private-${privateSessionNonce}`;
}

function allowedNavigation(platform, rawURL) {
  try {
    const parsed = new URL(rawURL);
    if (parsed.protocol !== 'https:') return false;
    return PLATFORMS[platform].hosts.some(host => parsed.hostname === host || parsed.hostname.endsWith(`.${host}`));
  } catch {
    return false;
  }
}

function configureRemoteSession(remoteSession, platform) {
  remoteSession.setPermissionCheckHandler((_webContents, permission, requestingOrigin) => permission === 'fullscreen' && allowedNavigation(platform, requestingOrigin));
  remoteSession.setPermissionRequestHandler((webContents, permission, callback, details) => {
    const requestingURL = details?.requestingUrl || webContents.getURL();
    callback(permission === 'fullscreen' && allowedNavigation(platform, requestingURL));
  });
  remoteSession.on('will-download', event => event.preventDefault());
}

function createRemoteView(platform) {
  const partition = partitionFor(platform);
  const remoteSession = session.fromPartition(partition, { cache: sessionMode === 'persistent' });
  configureRemoteSession(remoteSession, platform);
  const view = new WebContentsView({
    webPreferences: {
      partition,
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: true,
      webSecurity: true,
      allowRunningInsecureContent: false,
      devTools: false,
      navigateOnDragDrop: false
    }
  });
  view.webContents.setWindowOpenHandler(({ url }) => {
    if (allowedNavigation(platform, url)) view.webContents.loadURL(url);
    return { action: 'deny' };
  });
  view.webContents.on('will-navigate', (event, url) => {
    if (!allowedNavigation(platform, url)) event.preventDefault();
  });
  const sendState = () => {
    if (!mainWindow || mainWindow.isDestroyed()) return;
    mainWindow.webContents.send('arion:web:state', {
      platform,
      url: view.webContents.getURL(),
      title: view.webContents.getTitle(),
      loading: view.webContents.isLoading(),
      canGoBack: view.webContents.navigationHistory.canGoBack(),
      canGoForward: view.webContents.navigationHistory.canGoForward()
    });
  };
  view.webContents.on('did-start-loading', sendState);
  view.webContents.on('did-stop-loading', sendState);
  view.webContents.on('page-title-updated', sendState);
  view.webContents.loadURL(PLATFORMS[platform].home);
  return view;
}

function normalizedBounds(bounds) {
  if (!mainWindow || mainWindow.isDestroyed()) return activeBounds;
  const content = mainWindow.getContentBounds();
  const number = value => Number.isFinite(Number(value)) ? Math.round(Number(value)) : 0;
  const x = Math.max(0, Math.min(number(bounds?.x), content.width - 200));
  const y = Math.max(0, Math.min(number(bounds?.y), content.height - 200));
  const width = Math.max(200, Math.min(number(bounds?.width), content.width - x));
  const height = Math.max(200, Math.min(number(bounds?.height), content.height - y));
  return { x, y, width, height };
}

function detachActiveView() {
  if (!activePlatform || !mainWindow) return;
  const view = webViews.get(activePlatform);
  if (view) {
    view.webContents.setAudioMuted(true);
    try { mainWindow.contentView.removeChildView(view); } catch {}
  }
  activePlatform = null;
}

function showPlatform(platform, bounds) {
  if (!PLATFORMS[platform]) throw new Error('Plataforma desconhecida.');
  detachActiveView();
  let view = webViews.get(platform);
  if (!view || view.webContents.isDestroyed()) {
    view = createRemoteView(platform);
    webViews.set(platform, view);
  }
  activeBounds = normalizedBounds(bounds);
  mainWindow.contentView.addChildView(view);
  view.setBounds(activeBounds);
  view.webContents.setAudioMuted(false);
  activePlatform = platform;
}

function validateLocalSender(event) {
  if (!mainWindow || event.sender !== mainWindow.webContents) return false;
  try {
    const parsed = new URL(event.senderFrame.url);
    return parsed.protocol === 'http:' && parsed.hostname === '127.0.0.1' && Number(parsed.port) === backendPort;
  } catch {
    return false;
  }
}

function registerIPC() {
  ipcMain.handle('arion:web:open', (event, payload) => {
    if (!validateLocalSender(event)) throw new Error('IPC negado.');
    showPlatform(payload?.platform, payload?.bounds);
    return { opened: true };
  });
  ipcMain.handle('arion:web:hide', event => {
    if (!validateLocalSender(event)) throw new Error('IPC negado.');
    detachActiveView();
    return { hidden: true };
  });
  ipcMain.handle('arion:web:resize', (event, bounds) => {
    if (!validateLocalSender(event)) throw new Error('IPC negado.');
    activeBounds = normalizedBounds(bounds);
    const view = activePlatform && webViews.get(activePlatform);
    if (view) view.setBounds(activeBounds);
    return activeBounds;
  });
  ipcMain.handle('arion:web:navigate', (event, command) => {
    if (!validateLocalSender(event)) throw new Error('IPC negado.');
    const view = activePlatform && webViews.get(activePlatform);
    if (!view) return { handled: false };
    const history = view.webContents.navigationHistory;
    if (command === 'back' && history.canGoBack()) history.goBack();
    else if (command === 'forward' && history.canGoForward()) history.goForward();
    else if (command === 'reload') view.webContents.reload();
    else if (command === 'home') view.webContents.loadURL(PLATFORMS[activePlatform].home);
    return { handled: true };
  });
  ipcMain.handle('arion:web:session-mode', (event, mode) => {
    if (!validateLocalSender(event)) throw new Error('IPC negado.');
    if (mode !== 'private' && mode !== 'persistent') throw new Error('Modo de sessão inválido.');
    if (mode !== sessionMode) {
      detachActiveView();
      for (const view of webViews.values()) if (!view.webContents.isDestroyed()) view.webContents.close();
      webViews.clear();
      sessionMode = mode;
      privateSessionNonce = crypto.randomBytes(12).toString('hex');
    }
    return { mode: sessionMode };
  });
  ipcMain.handle('arion:web:clear-data', async (event, platform) => {
    if (!validateLocalSender(event)) throw new Error('IPC negado.');
    if (!PLATFORMS[platform]) throw new Error('Plataforma desconhecida.');
    const partitions = [`persist:arion-${platform}`, `arion-${platform}-private-${privateSessionNonce}`];
    for (const partition of partitions) await session.fromPartition(partition).clearStorageData();
    return { cleared: true };
  });
}

function stopBackend() {
  if (!backendProcess || backendProcess.killed) return;
  if (backendPort && backendToken) {
    const request = http.request({ hostname: '127.0.0.1', port: backendPort, path: '/api/shutdown', method: 'POST', headers: { Authorization: `Bearer ${backendToken}` } });
    request.on('error', () => {});
    request.end();
  }
  setTimeout(() => {
    if (backendProcess && !backendProcess.killed) backendProcess.kill('SIGTERM');
  }, 800).unref();
}

app.whenReady().then(async () => {
  registerIPC();
  const localURL = await startBackend();
  createMainWindow(localURL);
}).catch(error => {
  console.error(error);
  app.quit();
});

app.on('before-quit', stopBackend);
app.on('window-all-closed', () => app.quit());
