// SPDX-License-Identifier: GPL-3.0-only

(() => {
  'use strict';

  const TRANSLATIONS = {
    en: {
      app: { title: 'Arion — Media Gallery' },
      brand: { subtitle: 'MEDIA GALLERY' },
      nav: {
        ariaLabel: 'Main navigation',
        home: 'Home',
        library: 'Library',
        web: 'Web Videos',
        personal: 'Personal Videos',
        providers: 'Providers',
        settings: 'Settings',
        detail: 'Collection'
      },
      privacy: {
        localAndPrivate: 'Local and private',
        telemetryDisabled: 'Telemetry disabled'
      },
      search: {
        placeholder: 'Search library and providers',
        inYourLibrary: 'IN YOUR LIBRARY',
        configuredProviders: 'CONFIGURED PROVIDERS',
        searching: 'SEARCHING…',
        loadingSources: 'Checking available sources…',
        noResults: 'Nothing found in library or providers.',
        emptyLibrary: 'Nothing found in library',
        localItemCount: '{count} item(s) • In your library',
        providerSources: '{count} source(s) found',
        adding: 'Adding…',
        addedToLibrary: '"{title}" was added to the library.'
      },
      home: {
        heroLabel: 'YOUR MEDIA. YOUR SPACE.',
        heroTitle: 'A gallery built for what is yours.',
        heroDesc: 'Organize personal videos and sources chosen by you without uploading your library to the cloud.',
        addFolder: 'Add folder',
        addProvider: 'Add provider',
        recentEyebrow: 'RECENT',
        recentTitle: 'Your collections',
        viewLibrary: 'View library'
      },
      library: {
        title: 'All collections',
        desc: 'Local folders, web collections and providers appear in the same place.',
        emptyTitle: 'Your gallery is empty',
        emptyDesc: 'Add a video folder to get started.',
        chooseFolder: 'Choose folder'
      },
      web: {
        title: 'Choose where to watch',
        desc: 'Remote sites will run in separate Chromium sessions from your library.',
        back: 'Back',
        forward: 'Forward',
        reload: 'Reload',
        home: 'Home',
        isolatedState: 'Isolated Chromium',
        clearSession: 'Clear session data',
        selectPlatform: 'Select a platform',
        platformNotice: 'The site will open in this space without access to the file system or backend.',
        privacyTitle: 'Transparent privacy',
        privacyDesc: 'Arion isolates sites from the local library. Platforms themselves may still log your activity while you use them.',
        youtubeIsolated: 'YouTube • isolated session',
        tiktokIsolated: 'TikTok • isolated session',
        notRunning: 'Chromium shell not started',
        loading: 'Loading…',
        isolatedSession: 'Isolated session',
        openPlatformFirst: 'Open a platform first.',
        dataCleared: 'Platform cookies and storage have been cleared.',
        shellRequired: 'Open Arion through the Chromium shell to use Web Videos.'
      },
      personal: {
        eyebrow: 'LOCAL LIBRARY',
        title: 'Personal videos',
        desc: 'You choose the folders. Nothing is copied, moved, or uploaded.',
        addFolderTitle: 'Add media folder',
        addFolderDesc: 'First-level folders will be organized as collections.',
        readOnlyBadge: 'Read-only',
        pathPlaceholder: '/home/user/Videos',
        scanButton: 'Scan folder',
        indexingTitle: 'Indexing...',
        indexedEyebrow: 'INDEXED',
        authorizedFolders: 'Authorized folders',
        emptyIndexed: 'No indexed folders',
        chooseFolderToast: 'Choose a folder to index.',
        indexingRunning: 'Indexing your library…',
        indexingIncomplete: 'Incomplete indexing',
        indexingUpdated: 'Library updated',
        runningCopy: '{indexed} of {found} file(s) processed',
        updatedCopy: '{count} video(s) found',
        updatedToast: 'Local library updated.'
      },
      providers: {
        extensibleEyebrow: 'EXTENSIBLE & NEUTRAL',
        mediaProvidersTitle: 'Media providers',
        neutralDesc: 'Arion neither offers nor recommends sources. It only recognizes compatible manifests chosen by you.',
        findInFolder: 'Find provider in a folder',
        findInFolderDesc: 'We will look for arion-provider.json with limited depth.',
        protocolBadge: 'Protocol 1',
        pathPlaceholder: '/path/to/provider',
        scanFolderButton: 'Scan folder',
        addSourcesTitle: 'Add sources and catalogs',
        addSourcesDesc: 'Arion uses neutral manifests and isolated connectors. No default sources are built-in.',
        optionAEyebrow: 'OPTION A — LOCAL PROVIDERS',
        optionATitle: 'Providers on your computer',
        optionADesc: 'Discover local executables containing the arion-provider.json manifest.',
        folderLabel: 'Providers folder',
        localFolderPlaceholder: '/home/user/.arion/providers or C:\\arion\\providers',
        searchLocalButton: 'Search local providers',
        optionBEyebrow: 'OPTION B — WEB PROVIDERS',
        optionBTitle: 'Connect website via HTTPS address',
        optionBDesc: 'Connect websites that publish a neutral manifest at /.well-known/arion-provider.json.',
        websiteLabel: 'Website address',
        websitePlaceholder: 'https://example.com',
        verifyManifestButton: 'Verify website manifest',
        manageEyebrow: 'MANAGE PROVIDERS',
        installedTitle: 'Installed providers',
        activate: 'Activate',
        addDisabled: 'Add disabled',
        added: 'Provider added.',
        enterPath: 'Enter the provider folder.',
        noManifestFound: 'No compatible manifest was found.',
        enterWebUrl: 'Enter the HTTPS address of the website.',
        probingWebManifest: 'Checking the public website manifest…',
        incompatibleWeb: 'This website does not expose a compatible integration and was not activated.',
        activateWebsite: 'Activate website',
        activatedWeb: 'Compatible website activated as provider.',
        websiteHttps: 'Website HTTPS',
        localProcess: 'Local process',
        active: 'Active',
        waitingExecutable: 'Awaiting executable',
        check: 'Check',
        inactive: 'Inactive',
        remove: 'Remove',
        removeAria: 'Remove provider',
        noneConfigured: 'No providers configured',
        statusState: 'Status: {status}',
        removedToast: 'Provider removed. Already imported collections have been preserved.'
      },
      settings: {
        controlEyebrow: 'CONTROL',
        title: 'Settings and privacy',
        desc: 'Explicit choices, no hidden settings.',
        language: 'Interface language',
        languageDesc: 'Select your preferred display language.',
        webSessionsTitle: 'Web Videos sessions',
        webSessionsDesc: 'Private clears data upon exit; persistent preserves logins.',
        modePrivate: 'Private',
        modePersistent: 'Persistent',
        keepHistoryTitle: 'Keep web history',
        keepHistoryDesc: 'Saves only URLs visited inside Arion.',
        telemetryTitle: 'Telemetry',
        telemetryDesc: 'Arion has no telemetry transport.',
        defaultPlayerTitle: 'Default player',
        defaultPlayerDesc: 'Player used for local videos.',
        playerIntegrated: 'Integrated',
        saveButton: 'Save settings',
        dataStorageTitle: 'Data stored on this computer',
        dataStorageDesc: 'Arion uses the configuration and cache folders of your profile. On Linux: ~/.config/arion and ~/.cache/arion. On Windows: %APPDATA%\\arion and %LOCALAPPDATA%\\arion.',
        saved: 'Settings saved.'
      },
      detail: {
        localCollection: 'LOCAL COLLECTION',
        providerCollection: 'PROVIDER COLLECTION',
        itemCount: '{count} item(s) • {source}',
        headerTitle: 'TITLE',
        headerDuration: 'DURATION',
        noVideos: 'This collection has no videos'
      },
      media: {
        localVideo: 'Local video',
        providerMedia: 'Provider media',
        watched: ' • Watched',
        inProgress: ' • In progress'
      },
      card: {
        onThisComputer: 'On this computer',
        videoCount: '{count} video(s) • {source}',
        emptyHome: 'No collections yet'
      },
      stats: {
        collections: 'Collections',
        localVideos: 'Local videos',
        watched: 'Watched'
      },
      audio: {
        dubbed: 'Dubbed',
        subbed: 'Subtitled'
      },
      player: {
        nowPlaying: 'NOW PLAYING',
        openExternal: 'Open in external player',
        loadingVideo: 'Loading video: {title}…',
        playing: 'Playing {title}',
        startedExternal: 'Playback started in {player}.',
        externalPlayer: 'external player',
        sentToExternal: 'Video sent to external player.',
        copiedTitle: 'Title copied to clipboard.'
      },
      itemModal: {
        eyebrow: 'MEDIA OPTIONS',
        episodeDefault: 'Episode',
        playEpisode: 'Play Episode',
        downloadFile: 'Download file to computer',
        copyTitle: 'Copy Title'
      },
      download: {
        default: 'Downloading file...',
        starting: 'Starting download of "{title}"...',
        downloadingTitle: 'Downloading {title}...',
        complete: 'Download complete',
        completedToast: 'Download of "{title}" completed and saved to local library.',
        error: 'Error downloading video.'
      },
      common: {
        back: 'Back',
        close: 'Close',
        errorPrefix: 'Error {status}',
        secureLauncherRequired: 'Open Arion through the secure launcher.'
      }
    },
    'pt-BR': {
      app: { title: 'Arion — Galeria de mídia' },
      brand: { subtitle: 'GALERIA DE MÍDIA' },
      nav: {
        ariaLabel: 'Navegação principal',
        home: 'Início',
        library: 'Biblioteca',
        web: 'Web Videos',
        personal: 'Vídeos pessoais',
        providers: 'Provedores',
        settings: 'Configurações',
        detail: 'Coleção'
      },
      privacy: {
        localAndPrivate: 'Local e privado',
        telemetryDisabled: 'Telemetria desativada'
      },
      search: {
        placeholder: 'Buscar na biblioteca e provedores',
        inYourLibrary: 'NA SUA BIBLIOTECA',
        configuredProviders: 'PROVEDORES CONFIGURADOS',
        searching: 'BUSCANDO…',
        loadingSources: 'Consultando as fontes disponíveis…',
        noResults: 'Nada encontrado na biblioteca ou nos provedores.',
        emptyLibrary: 'Nada encontrado na biblioteca',
        localItemCount: '{count} item(ns) • Na sua biblioteca',
        providerSources: '{count} fonte(s) encontrada(s)',
        adding: 'Adicionando…',
        addedToLibrary: '"{title}" foi adicionado à biblioteca.'
      },
      home: {
        heroLabel: 'SUA MÍDIA. SEU ESPAÇO.',
        heroTitle: 'Uma galeria feita para o que é seu.',
        heroDesc: 'Organize vídeos pessoais e fontes escolhidas por você sem enviar sua biblioteca para a nuvem.',
        addFolder: 'Adicionar pasta',
        addProvider: 'Adicionar provedor',
        recentEyebrow: 'RECENTES',
        recentTitle: 'Suas coleções',
        viewLibrary: 'Ver biblioteca'
      },
      library: {
        title: 'Todas as coleções',
        desc: 'Pastas locais, coleções web e provedores aparecem no mesmo lugar.',
        emptyTitle: 'Sua galeria está vazia',
        emptyDesc: 'Adicione uma pasta de vídeos para começar.',
        chooseFolder: 'Escolher pasta'
      },
      web: {
        title: 'Escolha onde assistir',
        desc: 'Sites remotos serão executados em sessões Chromium separadas da sua biblioteca.',
        back: 'Voltar',
        forward: 'Avançar',
        reload: 'Atualizar',
        home: 'Início',
        isolatedState: 'Chromium isolado',
        clearSession: 'Limpar dados da sessão',
        selectPlatform: 'Selecione uma plataforma',
        platformNotice: 'O site será aberto neste espaço sem acesso ao sistema de arquivos ou ao backend.',
        privacyTitle: 'Privacidade transparente',
        privacyDesc: 'O Arion isola os sites da biblioteca local. As próprias plataformas ainda podem registrar sua atividade enquanto você as utiliza.',
        youtubeIsolated: 'YouTube • sessão isolada',
        tiktokIsolated: 'TikTok • sessão isolada',
        notRunning: 'Shell Chromium não iniciado',
        loading: 'Carregando…',
        isolatedSession: 'Sessão isolada',
        openPlatformFirst: 'Abra uma plataforma primeiro.',
        dataCleared: 'Cookies e armazenamento da plataforma foram apagados.',
        shellRequired: 'Abra o Arion pelo shell Chromium para usar Web Videos.'
      },
      personal: {
        eyebrow: 'BIBLIOTECA LOCAL',
        title: 'Vídeos pessoais',
        desc: 'Você escolhe as pastas. Nada é copiado, movido ou enviado.',
        addFolderTitle: 'Adicionar pasta de mídia',
        addFolderDesc: 'Pastas de primeiro nível serão organizadas como coleções.',
        readOnlyBadge: 'Somente leitura',
        pathPlaceholder: '/home/usuario/Vídeos',
        scanButton: 'Indexar pasta',
        indexingTitle: 'Indexando...',
        indexedEyebrow: 'INDEXADAS',
        authorizedFolders: 'Pastas autorizadas',
        emptyIndexed: 'Nenhuma pasta indexada',
        chooseFolderToast: 'Escolha uma pasta para indexar.',
        indexingRunning: 'Indexando sua biblioteca…',
        indexingIncomplete: 'Indexação incompleta',
        indexingUpdated: 'Biblioteca atualizada',
        runningCopy: '{indexed} de {found} arquivo(s) processados',
        updatedCopy: '{count} vídeo(s) encontrado(s)',
        updatedToast: 'Biblioteca local atualizada.'
      },
      providers: {
        extensibleEyebrow: 'EXTENSÍVEL E NEUTRO',
        mediaProvidersTitle: 'Provedores de mídia',
        neutralDesc: 'O Arion não oferece nem recomenda fontes. Ele reconhece apenas manifestos compatíveis escolhidos por você.',
        findInFolder: 'Encontrar provedor em uma pasta',
        findInFolderDesc: 'Procuraremos por arion-provider.json com profundidade limitada.',
        protocolBadge: 'Protocolo 1',
        pathPlaceholder: '/caminho/do/provedor',
        scanFolderButton: 'Verificar pasta',
        addSourcesTitle: 'Adicionar fontes e catálogos',
        addSourcesDesc: 'O Arion usa manifestos neutros e conectores isolados. Nenhuma fonte padrão vem embutida.',
        optionAEyebrow: 'OPÇÃO A — PROVEDORES LOCAIS',
        optionATitle: 'Provedores no seu computador',
        optionADesc: 'Descubra executáveis locais contendo o manifesto arion-provider.json.',
        folderLabel: 'Pasta dos provedores',
        localFolderPlaceholder: '/home/usuario/.arion/providers ou C:\\arion\\providers',
        searchLocalButton: 'Buscar provedores locais',
        optionBEyebrow: 'OPÇÃO B — PROVEDORES WEB',
        optionBTitle: 'Conectar site por endereço HTTPS',
        optionBDesc: 'Conecte sites que publiquem um manifesto neutro em /.well-known/arion-provider.json.',
        websiteLabel: 'Endereço do site',
        websitePlaceholder: 'https://exemplo.com',
        verifyManifestButton: 'Verificar manifesto do site',
        manageEyebrow: 'GERENCIAR PROVEDORES',
        installedTitle: 'Provedores instalados',
        activate: 'Ativar',
        addDisabled: 'Adicionar desativado',
        added: 'Provedor adicionado.',
        enterPath: 'Informe a pasta do provedor.',
        noManifestFound: 'Nenhum manifesto compatível foi encontrado.',
        enterWebUrl: 'Informe o endereço HTTPS do site.',
        probingWebManifest: 'Verificando o manifesto público do site…',
        incompatibleWeb: 'Este site não expõe uma integração compatível e não foi ativado.',
        activateWebsite: 'Ativar site',
        activatedWeb: 'Site compatível ativado como provedor.',
        websiteHttps: 'Website HTTPS',
        localProcess: 'Processo local',
        active: 'Ativo',
        waitingExecutable: 'Aguardando executável',
        check: 'Verificar',
        inactive: 'Inativo',
        remove: 'Remover',
        removeAria: 'Remover provedor',
        noneConfigured: 'Nenhum provedor configurado',
        statusState: 'Estado: {status}',
        removedToast: 'Provedor removido. As coleções já importadas foram preservadas.'
      },
      settings: {
        controlEyebrow: 'CONTROLE',
        title: 'Configurações e privacidade',
        desc: 'Escolhas explícitas, sem configurações escondidas.',
        language: 'Idioma da interface',
        languageDesc: 'Selecione o idioma de exibição do aplicativo.',
        webSessionsTitle: 'Sessões Web Videos',
        webSessionsDesc: 'Privada apaga dados ao fechar; persistente mantém logins.',
        modePrivate: 'Privada',
        modePersistent: 'Persistente',
        keepHistoryTitle: 'Manter histórico web',
        keepHistoryDesc: 'Salva apenas endereços visitados dentro do Arion.',
        telemetryTitle: 'Telemetria',
        telemetryDesc: 'O Arion não possui transporte de telemetria.',
        defaultPlayerTitle: 'Player padrão',
        defaultPlayerDesc: 'Player usado para vídeos locais.',
        playerIntegrated: 'Integrado',
        saveButton: 'Salvar configurações',
        dataStorageTitle: 'Dados armazenados neste computador',
        dataStorageDesc: 'O Arion usa as pastas de configuração e cache do seu perfil. No Linux: ~/.config/arion e ~/.cache/arion. No Windows: %APPDATA%\\arion e %LOCALAPPDATA%\\arion.',
        saved: 'Configurações salvas.'
      },
      detail: {
        localCollection: 'COLEÇÃO LOCAL',
        providerCollection: 'COLEÇÃO DE PROVEDOR',
        itemCount: '{count} item(ns) • {source}',
        headerTitle: 'TÍTULO',
        headerDuration: 'DURAÇÃO',
        noVideos: 'Esta coleção não possui vídeos'
      },
      media: {
        localVideo: 'Vídeo local',
        providerMedia: 'Mídia do provedor',
        watched: ' • Assistido',
        inProgress: ' • Em andamento'
      },
      card: {
        onThisComputer: 'Neste computador',
        videoCount: '{count} vídeo(s) • {source}',
        emptyHome: 'Nenhuma coleção ainda'
      },
      stats: {
        collections: 'Coleções',
        localVideos: 'Vídeos locais',
        watched: 'Assistidos'
      },
      audio: {
        dubbed: 'Dublado',
        subbed: 'Legendado'
      },
      player: {
        nowPlaying: 'REPRODUZINDO',
        openExternal: 'Abrir no player externo',
        loadingVideo: 'Carregando vídeo: {title}…',
        playing: 'Reproduzindo {title}',
        startedExternal: 'Reprodução iniciada no {player}.',
        externalPlayer: 'player externo',
        sentToExternal: 'Vídeo enviado ao player externo.',
        copiedTitle: 'Título copiado para a área de transferência.'
      },
      itemModal: {
        eyebrow: 'OPÇÕES DE MÍDIA',
        episodeDefault: 'Episódio',
        playEpisode: 'Reproduzir Episódio',
        downloadFile: 'Baixar arquivo para o computador',
        copyTitle: 'Copiar Título'
      },
      download: {
        default: 'Baixando arquivo...',
        starting: 'Iniciando download de "{title}"...',
        downloadingTitle: 'Baixando {title}...',
        complete: 'Download concluído',
        completedToast: 'Download de "{title}" concluído e salvo na biblioteca local.',
        error: 'Erro ao realizar download do vídeo.'
      },
      common: {
        back: 'Voltar',
        close: 'Fechar',
        errorPrefix: 'Erro {status}',
        secureLauncherRequired: 'Abra o Arion pelo inicializador seguro.'
      }
    }
  };

  const state = {
    token: new URLSearchParams(location.hash.slice(1)).get('session') || '',
    language: 'en',
    currentView: 'home',
    collections: [],
    settings: null,
    activeCollection: null,
    activeItem: null,
    webPlatform: null,
    indexTimer: null,
    searchTimer: null,
    searchAbortController: null,
    providerMatches: []
  };

  if (state.token) history.replaceState(null, '', location.pathname + location.search);

  const $ = selector => document.querySelector(selector);
  const $$ = selector => [...document.querySelectorAll(selector)];

  function t(key, params = {}) {
    const lang = state.language === 'pt-BR' || state.language === 'pt' ? 'pt-BR' : 'en';
    const resolve = (table, path) => path.split('.').reduce((acc, part) => acc?.[part], table);
    let val = resolve(TRANSLATIONS[lang], key);
    if (val === undefined) val = resolve(TRANSLATIONS.en, key);
    if (val === undefined) return key;
    return String(val).replace(/\{(\w+)\}/g, (_, k) => params[k] !== undefined ? params[k] : `{${k}}`);
  }

  function translateDOM() {
    document.documentElement.lang = state.language === 'pt-BR' ? 'pt-BR' : 'en';
    document.title = t('app.title');
    $$('[data-i18n]').forEach(node => {
      const key = node.dataset.i18n;
      if (key) node.textContent = t(key);
    });
    $$('[data-i18n-placeholder]').forEach(node => {
      const key = node.dataset.i18nPlaceholder;
      if (key) node.placeholder = t(key);
    });
    $$('[data-i18n-title]').forEach(node => {
      const key = node.dataset.i18nTitle;
      if (key) node.title = t(key);
    });
    $$('[data-i18n-aria-label]').forEach(node => {
      const key = node.dataset.i18nAriaLabel;
      if (key) node.setAttribute('aria-label', t(key));
    });
    const pageTitle = $('#page-title');
    if (pageTitle) pageTitle.textContent = t(`nav.${state.currentView}`) || 'Arion';
  }

  async function setLanguage(lang, save = false) {
    const normalized = lang === 'pt-BR' || lang === 'pt' ? 'pt-BR' : 'en';
    state.language = normalized;
    const select = $('#setting-language');
    if (select && select.value !== normalized) select.value = normalized;
    translateDOM();
    renderCollections();
    if (state.currentView === 'detail' && state.activeCollection) {
      openCollection(state.activeCollection.id);
    }
    const query = $('#global-search')?.value.trim();
    if (query) runGlobalSearch(query);
    if (save && state.settings) {
      const next = structuredClone(state.settings);
      next.language = normalized;
      try {
        state.settings = await api('/api/settings', { method: 'POST', body: JSON.stringify(next) });
        showToast(t('settings.saved'));
      } catch (err) {
        showToast(err.message, true);
      }
    }
  }

  async function api(path, options = {}) {
    const headers = new Headers(options.headers || {});
    headers.set('Authorization', `Bearer ${state.token}`);
    if (options.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
    const response = await fetch(path, { ...options, headers });
    const type = response.headers.get('content-type') || '';
    const payload = type.includes('application/json') ? await response.json() : await response.text();
    if (!response.ok) throw new Error(payload?.error || payload || t('common.errorPrefix', { status: response.status }));
    return payload;
  }

  function mediaURL(path, id) {
    return `${path}?id=${encodeURIComponent(id)}&session=${encodeURIComponent(state.token)}`;
  }

  function artworkURL(rawURL) {
    if (!rawURL) return '';
    if (/^https?:\/\//i.test(rawURL)) return `/api/media/artwork?url=${encodeURIComponent(rawURL)}&session=${encodeURIComponent(state.token)}`;
    if (rawURL.startsWith('/api/')) return `${rawURL}${rawURL.includes('?') ? '&' : '?'}session=${encodeURIComponent(state.token)}`;
    return '';
  }

  function escapeHTML(value) {
    const div = document.createElement('div');
    div.textContent = value ?? '';
    return div.innerHTML.replaceAll('"', '&quot;').replaceAll("'", '&#39;');
  }

  function formatDuration(seconds) {
    if (!seconds) return '—';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    return hours ? `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}` : `${minutes}:${String(secs).padStart(2, '0')}`;
  }

  function showToast(message, error = false) {
    const toast = $('#toast');
    toast.textContent = message;
    toast.className = `toast${error ? ' error' : ''}`;
    clearTimeout(showToast.timer);
    showToast.timer = setTimeout(() => toast.classList.add('hidden'), 3400);
  }

  function navigate(view) {
    state.currentView = view;
    $$('.view').forEach(node => node.classList.toggle('active', node.id === `view-${view}`));
    $$('.nav-item').forEach(node => node.classList.toggle('active', node.dataset.view === view));
    $('#page-title').textContent = t(`nav.${view}`) || 'Arion';
    const content = $('.content');
    content.classList.toggle('web-mode', view === 'web');
    content.scrollTop = 0;
    if (view !== 'web' && window.arionDesktop?.available) {
      window.arionDesktop.hideWebView().catch(() => {});
    } else if (state.webPlatform && window.arionDesktop?.available) {
      requestAnimationFrame(() => openWebPlatform(state.webPlatform));
    }
  }

  const PLAY_SVG = `<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" style="display:inline-block; vertical-align:middle;"><path d="M8 5v14l11-7z"/></svg>`;

  function collectionCard(collection) {
    const first = collection.items?.[0];
    const artwork = artworkURL(collection.artwork_url) || (first?.source_id === 'local' ? mediaURL('/api/media/thumbnail', first.id) : '');
    const art = artwork ? `<img src="${escapeHTML(artwork)}" alt="" loading="lazy">` : '';
    const sourceLabel = collection.source_id === 'local' ? t('card.onThisComputer') : escapeHTML(collection.source_id);
    const countLabel = t('card.videoCount', { count: collection.items?.length || 0, source: sourceLabel });
    return `<button class="collection-card" data-collection="${escapeHTML(collection.id)}">
      <span class="collection-art">${art}<span class="art-fallback">${escapeHTML(collection.title.slice(0, 1).toUpperCase())}</span><span class="play-bubble">${PLAY_SVG}</span></span>
      <strong>${escapeHTML(collection.title)}</strong>
      <small>${countLabel}</small>
    </button>`;
  }

  function bindCollectionCards(container) {
    container.querySelectorAll('[data-collection]').forEach(button => button.addEventListener('click', () => openCollection(button.dataset.collection)));
    bindImageFallbacks(container);
  }

  function bindImageFallbacks(container) {
    container.querySelectorAll('img').forEach(image => image.addEventListener('error', () => image.remove(), { once: true }));
  }

  function renderCollections() {
    const cards = state.collections.map(collectionCard).join('');
    $('#home-collections').innerHTML = cards || emptyInline(t('card.emptyHome'));
    $('#library-grid').innerHTML = cards;
    $('#library-empty').classList.toggle('hidden', state.collections.length > 0);
    const personal = state.collections.filter(collection => collection.kind === 'local_folder');
    $('#personal-collections').innerHTML = personal.map(collectionCard).join('') || emptyInline(t('personal.emptyIndexed'));
    [$('#home-collections'), $('#library-grid'), $('#personal-collections')].forEach(bindCollectionCards);

    const itemCount = state.collections.reduce((sum, collection) => sum + (collection.items?.length || 0), 0);
    const watched = state.collections.reduce((sum, collection) => sum + (collection.items || []).filter(item => item.watched).length, 0);
    $('#home-stats').innerHTML = `<article><strong>${state.collections.length}</strong><span>${t('stats.collections')}</span></article><article><strong>${itemCount}</strong><span>${t('stats.localVideos')}</span></article><article><strong>${watched}</strong><span>${t('stats.watched')}</span></article>`;
  }

  function emptyInline(text) {
    return `<div class="empty-inline">${escapeHTML(text)}</div>`;
  }

  function mergeProviderMatches(sources) {
    const merged = new Map();
    for (const source of sources || []) {
      for (const item of source.items || []) {
        const key = item.title.toLowerCase().normalize('NFKD').replace(/[\u0300-\u036f]/g, '').replace(/[^a-z0-9]+/g, '-');
        let target = merged.get(key);
        if (!target) {
          target = { ...item, variants: [] };
          merged.set(key, target);
        } else {
          if (!target.artwork_url && item.artwork_url) target.artwork_url = item.artwork_url;
          if (!target.description && item.description) target.description = item.description;
        }
        for (const variant of item.variants || []) {
          const enriched = { ...variant, provider_id: source.provider_id, provider_name: source.provider_name };
          if (!target.variants.some(existing => existing.provider_id === enriched.provider_id && existing.id === enriched.id && existing.reference === enriched.reference)) {
            target.variants.push(enriched);
          }
        }
      }
    }
    return [...merged.values()].slice(0, 8);
  }

  function variantLabel(variant) {
    const audio = (variant.audio || []).map(value => value === 'dubbed' ? t('audio.dubbed') : value === 'subbed' ? t('audio.subbed') : value);
    const details = [...(variant.languages || []), ...audio];
    return `${variant.label}${details.length ? ` • ${details.join(' • ')}` : ''}`;
  }

  function renderGlobalSearch(query, localMatches, loading = false) {
    const popover = $('#global-search-results');
    if (!query) {
      popover.classList.add('hidden');
      return;
    }
    const localHTML = localMatches.slice(0, 5).map(collection => `<button class="search-result-row" data-search-collection="${escapeHTML(collection.id)}"><span class="search-result-art">${escapeHTML(collection.title.slice(0, 1).toUpperCase())}</span><span class="search-result-copy"><strong>${escapeHTML(collection.title)}</strong><small>${t('search.localItemCount', { count: collection.items?.length || 0 })}</small></span><span>></span></button>`).join('');
    const providerHTML = state.providerMatches.map((item, itemIndex) => {
      const artwork = artworkURL(item.artwork_url);
      const variants = item.variants.map((variant, variantIndex) => `<button class="search-variant" data-provider-match="${itemIndex}" data-provider-variant="${variantIndex}">+ ${escapeHTML(variantLabel(variant))}</button>`).join('');
      return `<article class="search-provider-result"><div class="search-provider-main"><span class="search-result-art">${artwork ? `<img src="${escapeHTML(artwork)}" alt="" loading="lazy">` : escapeHTML(item.title.slice(0, 1).toUpperCase())}</span><span class="search-result-copy"><strong>${escapeHTML(item.title)}</strong><small>${t('search.providerSources', { count: item.variants.length })}</small></span></div><div class="search-variants">${variants}</div></article>`;
    }).join('');
    const hasResults = localMatches.length > 0 || state.providerMatches.length > 0;
    popover.innerHTML = `${localHTML ? `<section class="search-popover-section"><div class="search-popover-heading"><span>${t('search.inYourLibrary')}</span><span>${localMatches.length}</span></div>${localHTML}</section>` : ''}${providerHTML || loading ? `<section class="search-popover-section"><div class="search-popover-heading"><span>${t('search.configuredProviders')}</span><span>${loading ? t('search.searching') : state.providerMatches.length}</span></div>${providerHTML}${loading ? `<div class="search-loading">${t('search.loadingSources')}</div>` : ''}</section>` : ''}${!hasResults && !loading ? `<div class="search-no-results">${t('search.noResults')}</div>` : ''}`;
    popover.classList.remove('hidden');
    popover.querySelectorAll('[data-search-collection]').forEach(button => button.addEventListener('click', () => {
      popover.classList.add('hidden');
      openCollection(button.dataset.searchCollection);
    }));
    popover.querySelectorAll('[data-provider-match]').forEach(button => button.addEventListener('click', () => importProviderMatch(Number(button.dataset.providerMatch), Number(button.dataset.providerVariant), button)));
    bindImageFallbacks(popover);
  }

  async function importProviderMatch(itemIndex, variantIndex, button) {
    const item = state.providerMatches[itemIndex];
    const variant = item?.variants?.[variantIndex];
    if (!variant) return;
    button.disabled = true;
    button.textContent = t('search.adding');
    try {
      const collection = await api('/api/providers/import', { method: 'POST', body: JSON.stringify({ provider_id: variant.provider_id, reference: variant.reference }) });
      await loadCollections();
      $('#global-search-results').classList.add('hidden');
      showToast(t('search.addedToLibrary', { title: collection.title }));
      openCollection(collection.id);
    } catch (error) {
      button.disabled = false;
      button.textContent = `+ ${variantLabel(variant)}`;
      showToast(error.message, true);
    }
  }

  function runGlobalSearch(rawQuery) {
    const query = rawQuery.trim();
    clearTimeout(state.searchTimer);
    state.searchAbortController?.abort();
    const normalized = query.toLowerCase();
    const localMatches = query ? state.collections.filter(collection => collection.title.toLowerCase().includes(normalized) || collection.items.some(item => item.title.toLowerCase().includes(normalized))) : state.collections;
    $('#library-grid').innerHTML = localMatches.map(collectionCard).join('') || emptyInline(t('search.emptyLibrary'));
    bindCollectionCards($('#library-grid'));
    state.providerMatches = [];
    if (!query) {
      renderGlobalSearch('', localMatches);
      return;
    }
    navigate('library');
    renderGlobalSearch(query, localMatches, query.length >= 2);
    if (query.length < 2) return;
    state.searchTimer = setTimeout(async () => {
      const controller = new AbortController();
      state.searchAbortController = controller;
      try {
        const payload = await api('/api/providers/search', { method: 'POST', signal: controller.signal, body: JSON.stringify({ query, limit: 8, preview: true }) });
        if ($('#global-search').value.trim() !== query) return;
        state.providerMatches = mergeProviderMatches(payload.sources);
        renderGlobalSearch(query, localMatches, true);
        const complete = await api('/api/providers/search', { method: 'POST', signal: controller.signal, body: JSON.stringify({ query, limit: 20, preview: false }) });
        if ($('#global-search').value.trim() !== query) return;
        state.providerMatches = mergeProviderMatches(complete.sources);
        renderGlobalSearch(query, localMatches, false);
      } catch (error) {
        if (error.name !== 'AbortError' && $('#global-search').value.trim() === query) renderGlobalSearch(query, localMatches, false);
      }
    }, 220);
  }

  async function loadCollections() {
    const payload = await api('/api/collections');
    state.collections = payload.collections || [];
    renderCollections();
  }

  function openItemMenu(item) {
    if (!item) return;
    state.activeItem = item;
    $('#item-action-title').textContent = item.title;
    $('#item-action-modal').classList.remove('hidden');
  }

  function closeItemMenu() {
    $('#item-action-modal').classList.add('hidden');
  }

  function openCollection(id) {
    const collection = state.collections.find(item => item.id === id);
    if (!collection) return;
    state.activeCollection = collection;
    const detailArtwork = artworkURL(collection.artwork_url) || (collection.items?.[0]?.source_id === 'local' ? mediaURL('/api/media/thumbnail', collection.items[0].id) : '');
    const descriptionHTML = collection.description ? `<p style="margin-top:8px; color:var(--muted); font-size:12px; line-height:1.5; max-width:750px;">${escapeHTML(collection.description)}</p>` : '';
    const kindBadge = collection.kind === 'local_folder' ? t('detail.localCollection') : t('detail.providerCollection');
    const countInfo = t('detail.itemCount', { count: collection.items.length, source: escapeHTML(collection.root_path || collection.source_id) });

    $('#detail-header').innerHTML = `<div class="detail-art">${detailArtwork ? `<img src="${escapeHTML(detailArtwork)}" alt="">` : ''}<span>${escapeHTML(collection.title.slice(0, 1))}</span></div><div><small>${kindBadge}</small><h2>${escapeHTML(collection.title)}</h2><p style="font-weight:600;">${countInfo}</p>${descriptionHTML}</div>`;
    $('#media-list').innerHTML = collection.items.map((item, index) => {
      const typeLabel = item.source_id === 'local' ? (item.width ? `${item.width}×${item.height}` : t('media.localVideo')) : t('media.providerMedia');
      const statusLabel = item.watched ? t('media.watched') : item.playback_time ? t('media.inProgress') : '';
      return `<div class="media-row">
        <button class="media-main" data-play="${escapeHTML(item.id)}"><span class="media-index">${index + 1}</span><span class="media-thumb">${item.source_id === 'local' ? `<img src="${mediaURL('/api/media/thumbnail', item.id)}" alt="">` : ''}<i>${PLAY_SVG}</i></span><span><strong>${escapeHTML(item.title)}</strong><small>${typeLabel}${statusLabel}</small></span></button>
        <span>${formatDuration(item.duration_seconds)}</span><button class="icon-button" data-more="${escapeHTML(item.id)}">•••</button>
      </div>`;
    }).join('') || emptyInline(t('detail.noVideos'));

    $$('#media-list [data-play]').forEach(button => button.addEventListener('click', () => playItem(button.dataset.play)));
    $$('#media-list [data-more]').forEach(button => button.addEventListener('click', () => {
      const item = collection.items.find(candidate => candidate.id === button.dataset.more);
      openItemMenu(item);
    }));
    bindImageFallbacks($('#view-detail'));
    navigate('detail');
  }

  function showDownloadProgress(text, percent = 0) {
    const badge = $('#download-progress-badge');
    const label = $('#download-progress-text');
    const pctLabel = $('#download-progress-percent');
    const fill = $('#download-progress-fill');

    const floatCard = $('#download-floating-card');
    const floatTitle = $('#download-floating-title');
    const floatPct = $('#download-floating-percent');
    const floatFill = $('#download-floating-fill');

    const clampPct = Math.min(100, Math.max(0, Math.round(percent)));
    const defaultText = t('download.default');

    if (badge && label && fill) {
      label.textContent = text || defaultText;
      if (pctLabel) pctLabel.textContent = `${clampPct}%`;
      fill.style.width = `${clampPct}%`;
      badge.classList.remove('hidden');
    }
    if (floatCard && floatTitle && floatFill) {
      floatTitle.textContent = text || defaultText;
      if (floatPct) floatPct.textContent = `${clampPct}%`;
      floatFill.style.width = `${clampPct}%`;
      floatCard.classList.remove('hidden');
    }
  }

  function hideDownloadProgress() {
    const badge = $('#download-progress-badge');
    if (badge) badge.classList.add('hidden');
    const floatCard = $('#download-floating-card');
    if (floatCard) floatCard.classList.add('hidden');
  }

  async function trackDownloadTask(taskID, itemTitle) {
    showDownloadProgress(t('download.downloadingTitle', { title: itemTitle }), 0);
    const interval = setInterval(async () => {
      try {
        const task = await api(`/api/media/download/status?id=${encodeURIComponent(taskID)}`);
        if (task.status === 'downloading') {
          showDownloadProgress(t('download.downloadingTitle', { title: itemTitle }), task.progress_percent || 0);
        } else if (task.status === 'completed') {
          clearInterval(interval);
          showDownloadProgress(t('download.complete'), 100);
          setTimeout(() => hideDownloadProgress(), 1500);
          showToast(t('download.completedToast', { title: itemTitle }));
          await loadCollections();
        } else if (task.status === 'failed') {
          clearInterval(interval);
          hideDownloadProgress();
          showToast(task.error || t('download.error'), true);
        }
      } catch (err) {
        clearInterval(interval);
        hideDownloadProgress();
      }
    }, 500);
  }

  async function playItem(id) {
    const item = state.activeCollection?.items.find(candidate => candidate.id === id);
    if (!item) return;
    state.activeItem = item;
    if (item.source_id !== 'local') {
      showDownloadProgress(t('player.loadingVideo', { title: item.title }));
      try {
        const result = await api('/api/providers/play', { method: 'POST', body: JSON.stringify({ item_id: item.id, player: state.settings?.default_player || 'integrated' }) });
        hideDownloadProgress();
        if (result.player === 'integrated' && result.url) {
          $('#player-title').textContent = item.title;
          const player = $('#local-player');
          const streamUrl = result.url.startsWith('http') ? result.url : (location.origin + result.url);
          player.src = streamUrl;
          $('#player-modal').classList.remove('hidden');
          player.play().catch(() => {});
          showToast(t('player.playing', { title: item.title }));
        } else {
          showToast(t('player.startedExternal', { player: result.player || t('player.externalPlayer') }));
        }
      } catch (error) {
        hideDownloadProgress();
        showToast(error.message, true);
      }
      return;
    }
    $('#player-title').textContent = item.title;
    const player = $('#local-player');
    const mediaUrl = mediaURL('/api/media/local', item.id);
    player.src = mediaUrl.startsWith('http') ? mediaUrl : (location.origin + mediaUrl);
    player.onloadedmetadata = () => {
      if (item.playback_time > 0 && item.playback_time < player.duration - 20) player.currentTime = item.playback_time;
    };
    $('#player-modal').classList.remove('hidden');
    player.play().catch(() => {});
  }

  async function saveProgress() {
    const player = $('#local-player');
    if (!state.activeItem || !Number.isFinite(player.currentTime)) return;
    const watched = Number.isFinite(player.duration) && player.duration > 0 && player.currentTime / player.duration > .9;
    try {
      await api('/api/history/update', { method: 'POST', body: JSON.stringify({ item_id: state.activeItem.id, seconds: Math.floor(player.currentTime), watched }) });
    } catch (_) {}
  }

  async function closePlayer() {
    await saveProgress();
    const player = $('#local-player');
    player.pause();
    player.removeAttribute('src');
    player.load();
    $('#player-modal').classList.add('hidden');
    await loadCollections();
    if (state.activeCollection) openCollection(state.activeCollection.id);
  }

  async function startIndex() {
    const root = $('#media-root-input').value.trim();
    if (!root) return showToast(t('personal.chooseFolderToast'), true);
    try {
      await api('/api/index/scan', { method: 'POST', body: JSON.stringify({ roots: [root] }) });
      $('#index-progress').classList.remove('hidden');
      pollIndex();
    } catch (error) {
      showToast(error.message, true);
    }
  }

  async function pollIndex() {
    clearTimeout(state.indexTimer);
    try {
      const status = await api('/api/index/status');
      $('#index-status-title').textContent = status.running ? t('personal.indexingRunning') : status.last_error ? t('personal.indexingIncomplete') : t('personal.indexingUpdated');
      $('#index-status-copy').textContent = status.running ? t('personal.runningCopy', { indexed: status.files_indexed, found: status.files_found }) : t('personal.updatedCopy', { count: status.files_indexed });
      if (status.running) state.indexTimer = setTimeout(pollIndex, 900);
      else {
        await loadCollections();
        if (status.last_error) showToast(status.last_error, true); else showToast(t('personal.updatedToast'));
      }
    } catch (error) {
      showToast(error.message, true);
    }
  }

  async function loadSuggestions() {
    const payload = await api('/api/index/suggestions');
    $('#suggested-roots').innerHTML = (payload.suggestions || []).map(path => `<button class="chip" data-root="${escapeHTML(path)}">${escapeHTML(path)}</button>`).join('');
    $$('#suggested-roots [data-root]').forEach(button => button.addEventListener('click', () => { $('#media-root-input').value = button.dataset.root; }));
  }

  async function discoverProviders() {
    const localInput = $('#provider-root-input-local') || $('#provider-root-input');
    const path = localInput ? localInput.value.trim() : '';
    if (!path) return showToast(t('providers.enterPath'), true);
    try {
      const payload = await api('/api/providers/discover', { method: 'POST', body: JSON.stringify({ path }) });
      renderCandidates(payload.candidates || []);
      if (!payload.candidates?.length) showToast(t('providers.noManifestFound'), true);
    } catch (error) {
      showToast(error.message, true);
    }
  }

  function renderCandidates(candidates) {
    $('#provider-candidates').innerHTML = candidates.map(candidate => `<article class="provider-row"><span class="provider-logo">${escapeHTML(candidate.manifest.name.slice(0, 1))}</span><div><strong>${escapeHTML(candidate.manifest.name)}</strong><small>v${escapeHTML(candidate.manifest.version)} • ${escapeHTML(candidate.status)}</small><code>${escapeHTML(candidate.manifest_path)}</code></div><button class="button ${candidate.ready ? 'primary' : 'secondary'}" data-register="${escapeHTML(candidate.manifest_path)}">${candidate.ready ? t('providers.activate') : t('providers.addDisabled')}</button></article>`).join('');
    $$('#provider-candidates [data-register]').forEach(button => button.addEventListener('click', async () => {
      try {
        await api('/api/providers/register', { method: 'POST', body: JSON.stringify({ manifest_path: button.dataset.register }) });
        showToast(t('providers.added'));
        await loadProviders();
      } catch (error) { showToast(error.message, true); }
    }));
  }

  async function probeWebsiteProvider() {
    const input = $('#website-provider-url');
    const button = $('#probe-website-provider');
    const url = input.value.trim();
    if (!url) return showToast(t('providers.enterWebUrl'), true);
    button.disabled = true;
    $('#website-provider-candidate').innerHTML = `<div class="empty-inline">${t('providers.probingWebManifest')}</div>`;
    try {
      const candidate = await api('/api/providers/web/probe', { method: 'POST', body: JSON.stringify({ url }) });
      renderWebsiteProviderCandidate(candidate);
    } catch (error) {
      $('#website-provider-candidate').innerHTML = emptyInline(t('providers.incompatibleWeb'));
      showToast(error.message, true);
    } finally {
      button.disabled = false;
    }
  }

  function renderWebsiteProviderCandidate(candidate) {
    const container = $('#website-provider-candidate');
    container.innerHTML = `<article class="provider-row"><span class="provider-logo">${escapeHTML(candidate.manifest.name.slice(0, 1))}</span><div><strong>${escapeHTML(candidate.manifest.name)}</strong><small>Site compatível • v${escapeHTML(candidate.manifest.version)} • ${escapeHTML(candidate.status)}</small><code>${escapeHTML(candidate.origin)}</code></div><button class="button primary" data-register-website="${escapeHTML(candidate.origin)}" data-manifest-fingerprint="${escapeHTML(candidate.fingerprint)}">${t('providers.activateWebsite')}</button></article>`;
    container.querySelector('[data-register-website]').addEventListener('click', async event => {
      const button = event.currentTarget;
      button.disabled = true;
      try {
        await api('/api/providers/web/register', { method: 'POST', body: JSON.stringify({ url: button.dataset.registerWebsite, fingerprint: button.dataset.manifestFingerprint }) });
        showToast(t('providers.activatedWeb'));
        container.innerHTML = '';
        await loadProviders();
      } catch (error) {
        showToast(error.message, true);
      } finally {
        button.disabled = false;
      }
    });
  }

  async function loadProviders() {
    const payload = await api('/api/providers');
    const providers = payload.providers || [];
    $('#installed-providers').innerHTML = providers.map(provider => {
      const website = provider.kind === 'website';
      const location = website ? provider.origin : provider.root_path;
      const stateLabel = provider.enabled ? t('providers.active') : t('providers.waitingExecutable');
      return `<article class="provider-row"><span class="provider-logo">${escapeHTML(provider.name.slice(0, 1))}</span><div><strong>${escapeHTML(provider.name)}</strong><small>${website ? t('providers.websiteHttps') : t('providers.localProcess')} • v${escapeHTML(provider.version)} • ${stateLabel}</small><code>${escapeHTML(location || '')}</code></div><div class="provider-actions">${provider.enabled ? `<button class="button secondary" data-provider-health="${escapeHTML(provider.id)}">${t('providers.check')}</button>` : `<span class="badge">${t('providers.inactive')}</span>`}<button class="button secondary" data-provider-remove="${escapeHTML(provider.id)}" aria-label="${t('providers.removeAria')}">${t('providers.remove')}</button></div></article>`;
    }).join('') || emptyInline(t('providers.noneConfigured'));
    $$('#installed-providers [data-provider-health]').forEach(button => button.addEventListener('click', async () => {
      button.disabled = true;
      try {
        const health = await api('/api/providers/health', { method: 'POST', body: JSON.stringify({ provider_id: button.dataset.providerHealth }) });
        showToast(health.message || t('providers.statusState', { status: health.status }));
      } catch (error) { showToast(error.message, true); }
      finally { button.disabled = false; }
    }));
    $$('#installed-providers [data-provider-remove]').forEach(button => button.addEventListener('click', async () => {
      button.disabled = true;
      try {
        await api(`/api/providers?id=${encodeURIComponent(button.dataset.providerRemove)}`, { method: 'DELETE' });
        showToast(t('providers.removedToast'));
        await loadProviders();
      } catch (error) { showToast(error.message, true); }
      finally { button.disabled = false; }
    }));
  }

  async function loadSettings() {
    state.settings = await api('/api/settings');
    const lang = state.settings.language || 'en';
    const normalizedLang = lang === 'pt-BR' || lang === 'pt' ? 'pt-BR' : 'en';
    $('#setting-language').value = normalizedLang;
    state.language = normalizedLang;
    translateDOM();
    $('#web-session-mode').value = state.settings.privacy?.web_session_mode || 'private';
    $('#keep-web-history').checked = Boolean(state.settings.privacy?.keep_web_history);
    $('#default-player').value = state.settings.default_player || 'integrated';
    if (!$('#media-root-input').value && state.settings.media_roots?.[0]) $('#media-root-input').value = state.settings.media_roots[0];
    if (window.arionDesktop?.available) await window.arionDesktop.setSessionMode(state.settings.privacy?.web_session_mode || 'private');
  }

  async function saveSettings() {
    const next = structuredClone(state.settings || {});
    next.language = $('#setting-language').value;
    next.default_player = $('#default-player').value;
    next.privacy = { telemetry_enabled: false, web_session_mode: $('#web-session-mode').value, keep_web_history: $('#keep-web-history').checked };
    try {
      state.settings = await api('/api/settings', { method: 'POST', body: JSON.stringify(next) });
      state.language = next.language;
      translateDOM();
      renderCollections();
      if (state.currentView === 'detail' && state.activeCollection) openCollection(state.activeCollection.id);
      if (window.arionDesktop?.available) await window.arionDesktop.setSessionMode(next.privacy.web_session_mode);
      showToast(t('settings.saved'));
    } catch (error) { showToast(error.message, true); }
  }

  function setupEvents() {
    $$('.nav-item[data-view]').forEach(button => button.addEventListener('click', () => navigate(button.dataset.view)));
    $$('[data-go]').forEach(button => button.addEventListener('click', () => navigate(button.dataset.go)));
    $('#detail-back').addEventListener('click', () => navigate('library'));
    $('#scan-media').addEventListener('click', startIndex);
    $('#discover-providers').addEventListener('click', discoverProviders);
    const discoverLocalBtn = $('#discover-providers-local');
    if (discoverLocalBtn) discoverLocalBtn.addEventListener('click', discoverProviders);
    $('#probe-website-provider').addEventListener('click', probeWebsiteProvider);
    $('#website-provider-url').addEventListener('keydown', event => { if (event.key === 'Enter') probeWebsiteProvider(); });
    $('#setting-language').addEventListener('change', event => setLanguage(event.target.value, false));
    $('#save-settings').addEventListener('click', saveSettings);
    $$('[data-web-platform]').forEach(button => button.addEventListener('click', () => openWebPlatform(button.dataset.webPlatform)));
    $$('[data-web-command]').forEach(button => button.addEventListener('click', () => window.arionDesktop?.navigateWeb(button.dataset.webCommand)));
    $('#clear-web-data').addEventListener('click', async () => {
      if (!state.webPlatform || !window.arionDesktop?.available) return showToast(t('web.openPlatformFirst'), true);
      try { await window.arionDesktop.clearWebData(state.webPlatform); showToast(t('web.dataCleared')); }
      catch (error) { showToast(error.message, true); }
    });
    $('#close-player').addEventListener('click', closePlayer);
    $('#player-modal').addEventListener('click', event => { if (event.target === $('#player-modal')) closePlayer(); });
    $('#close-item-action').addEventListener('click', closeItemMenu);
    $('#item-action-modal').addEventListener('click', event => { if (event.target === $('#item-action-modal')) closeItemMenu(); });
    $('#action-btn-play').addEventListener('click', () => {
      closeItemMenu();
      if (state.activeItem) playItem(state.activeItem.id);
    });
    $('#action-btn-download').addEventListener('click', async () => {
      closeItemMenu();
      if (!state.activeItem) return;
      const item = state.activeItem;
      showToast(t('download.starting', { title: item.title }));
      try {
        const task = await api('/api/media/download', {
          method: 'POST',
          body: JSON.stringify({
            item_id: item.id,
            provider_id: item.source_id,
            reference: item.provider_reference,
            title: item.title,
            collection_id: state.activeCollection?.id,
            collection_title: state.activeCollection?.title,
            artwork_url: state.activeCollection?.artwork_url
          })
        });
        if (task && task.id) {
          trackDownloadTask(task.id, item.title);
        }
      } catch (error) {
        hideDownloadProgress();
        showToast(error.message, true);
      }
    });
    $('#action-btn-copy').addEventListener('click', () => {
      closeItemMenu();
      if (!state.activeItem) return;
      navigator.clipboard.writeText(state.activeItem.title).then(() => showToast(t('player.copiedTitle'))).catch(() => {});
    });
    $('#open-external-player').addEventListener('click', async () => {
      if (!state.activeItem) return;
      const preferred = state.settings?.default_player;
      try { await api('/api/player/play', { method: 'POST', body: JSON.stringify({ item_id: state.activeItem.id, player: preferred && preferred !== 'integrated' ? preferred : 'mpv' }) }); showToast(t('player.sentToExternal')); }
      catch (error) { showToast(error.message, true); }
    });
    $('#local-player').addEventListener('timeupdate', () => {
      const now = Math.floor($('#local-player').currentTime);
      if (now > 0 && now % 15 === 0 && now !== saveProgress.last) { saveProgress.last = now; saveProgress(); }
    });
    $('#global-search').addEventListener('input', event => runGlobalSearch(event.target.value));
    $('#global-search').addEventListener('focus', event => {
      if (event.target.value.trim()) runGlobalSearch(event.target.value);
    });
    document.addEventListener('click', event => {
      if (!event.target.closest('.global-search-shell')) $('#global-search-results').classList.add('hidden');
    });
    document.addEventListener('keydown', event => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') { event.preventDefault(); $('#global-search').focus(); }
      if (event.key === 'Escape') {
        $('#global-search-results').classList.add('hidden');
        if (!$('#player-modal').classList.contains('hidden')) closePlayer();
      }
    });
  }

  function webSurfaceBounds() {
    const rect = $('#web-surface').getBoundingClientRect();
    return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
  }

  async function openWebPlatform(platform) {
    if (!window.arionDesktop?.available) return showToast(t('web.shellRequired'), true);
    try {
      state.webPlatform = platform;
      await window.arionDesktop.openWebPlatform(platform, webSurfaceBounds());
      $$('[data-web-platform]').forEach(button => {
        button.classList.toggle('primary', button.dataset.webPlatform === platform);
        button.classList.toggle('secondary', button.dataset.webPlatform !== platform);
      });
      $('#web-state-label').textContent = platform === 'youtube' ? t('web.youtubeIsolated') : t('web.tiktokIsolated');
    } catch (error) { showToast(error.message, true); }
  }

  function setupDesktopBridge() {
    if (!window.arionDesktop?.available) {
      $('#web-state-label').textContent = t('web.notRunning');
      return;
    }
    window.arionDesktop.onWebState(webState => {
      if (webState.platform !== state.webPlatform) return;
      $('#web-state-label').textContent = webState.loading ? t('web.loading') : (webState.title || t('web.isolatedSession'));
    });
    const observer = new ResizeObserver(() => {
      if (state.webPlatform && $('#view-web').classList.contains('active')) window.arionDesktop.resizeWebView(webSurfaceBounds()).catch(() => {});
    });
    observer.observe($('#web-surface'));
    const syncWebSurface = () => {
      if (state.webPlatform && $('#view-web').classList.contains('active')) {
        window.arionDesktop.resizeWebView(webSurfaceBounds()).catch(() => {});
      }
    };
    window.addEventListener('resize', syncWebSurface);
    $('.content').addEventListener('scroll', syncWebSurface, { passive: true });
  }

  async function boot() {
    setupEvents();
    setupDesktopBridge();
    translateDOM();
    if (!state.token) {
      showToast(t('common.secureLauncherRequired'), true);
      return;
    }
    try {
      await Promise.all([loadCollections(), loadSuggestions(), loadProviders(), loadSettings()]);
      const status = await api('/api/index/status');
      if (status.running) { $('#index-progress').classList.remove('hidden'); pollIndex(); }
    } catch (error) {
      showToast(error.message, true);
    }
  }

  boot();
})();
