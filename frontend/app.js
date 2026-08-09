// SPDX-License-Identifier: GPL-3.0-only

(() => {
  'use strict';

  const state = {
    token: new URLSearchParams(location.hash.slice(1)).get('session') || '',
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

  async function api(path, options = {}) {
    const headers = new Headers(options.headers || {});
    headers.set('Authorization', `Bearer ${state.token}`);
    if (options.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
    const response = await fetch(path, { ...options, headers });
    const type = response.headers.get('content-type') || '';
    const payload = type.includes('application/json') ? await response.json() : await response.text();
    if (!response.ok) throw new Error(payload?.error || payload || `Erro ${response.status}`);
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

  const titles = { home: 'Início', library: 'Biblioteca', web: 'Web Videos', personal: 'Vídeos pessoais', providers: 'Provedores', settings: 'Configurações', detail: 'Coleção' };

  function navigate(view) {
    $$('.view').forEach(node => node.classList.toggle('active', node.id === `view-${view}`));
    $$('.nav-item').forEach(node => node.classList.toggle('active', node.dataset.view === view));
    $('#page-title').textContent = titles[view] || 'Arion';
    const content = $('.content');
    content.classList.toggle('web-mode', view === 'web');
    content.scrollTop = 0;
    if (view !== 'web' && window.arionDesktop?.available) {
      window.arionDesktop.hideWebView().catch(() => {});
    } else if (state.webPlatform && window.arionDesktop?.available) {
      requestAnimationFrame(() => openWebPlatform(state.webPlatform));
    }
  }

  function collectionCard(collection) {
    const first = collection.items?.[0];
    const artwork = artworkURL(collection.artwork_url) || (first?.source_id === 'local' ? mediaURL('/api/media/thumbnail', first.id) : '');
    const art = artwork ? `<img src="${escapeHTML(artwork)}" alt="" loading="lazy">` : '';
    return `<button class="collection-card" data-collection="${escapeHTML(collection.id)}">
      <span class="collection-art">${art}<span class="art-fallback">${escapeHTML(collection.title.slice(0, 1).toUpperCase())}</span><span class="play-bubble">▶</span></span>
      <strong>${escapeHTML(collection.title)}</strong>
      <small>${collection.items?.length || 0} vídeo(s) • ${collection.source_id === 'local' ? 'Neste computador' : escapeHTML(collection.source_id)}</small>
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
    $('#home-collections').innerHTML = cards || emptyInline('Nenhuma coleção ainda');
    $('#library-grid').innerHTML = cards;
    $('#library-empty').classList.toggle('hidden', state.collections.length > 0);
    const personal = state.collections.filter(collection => collection.kind === 'local_folder');
    $('#personal-collections').innerHTML = personal.map(collectionCard).join('') || emptyInline('Nenhuma pasta indexada');
    [$('#home-collections'), $('#library-grid'), $('#personal-collections')].forEach(bindCollectionCards);

    const itemCount = state.collections.reduce((sum, collection) => sum + (collection.items?.length || 0), 0);
    const watched = state.collections.reduce((sum, collection) => sum + (collection.items || []).filter(item => item.watched).length, 0);
    $('#home-stats').innerHTML = `<article><strong>${state.collections.length}</strong><span>Coleções</span></article><article><strong>${itemCount}</strong><span>Vídeos locais</span></article><article><strong>${watched}</strong><span>Assistidos</span></article>`;
  }

  function emptyInline(text) {
    return `<div class="empty-inline">${escapeHTML(text)}</div>`;
  }

  function mergeProviderMatches(sources) {
    const merged = new Map();
    for (const source of sources || []) {
      for (const item of source.items || []) {
        const key = item.title.toLocaleLowerCase('pt-BR').normalize('NFKD').replace(/[\u0300-\u036f]/g, '').replace(/[^a-z0-9]+/g, '-');
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
    const audio = (variant.audio || []).map(value => value === 'dubbed' ? 'Dublado' : value === 'subbed' ? 'Legendado' : value);
    const details = [...(variant.languages || []), ...audio];
    return `${variant.label}${details.length ? ` • ${details.join(' • ')}` : ''}`;
  }

  function renderGlobalSearch(query, localMatches, loading = false) {
    const popover = $('#global-search-results');
    if (!query) {
      popover.classList.add('hidden');
      return;
    }
    const localHTML = localMatches.slice(0, 5).map(collection => `<button class="search-result-row" data-search-collection="${escapeHTML(collection.id)}"><span class="search-result-art">${escapeHTML(collection.title.slice(0, 1).toUpperCase())}</span><span class="search-result-copy"><strong>${escapeHTML(collection.title)}</strong><small>${collection.items?.length || 0} item(ns) • Na sua biblioteca</small></span><span>›</span></button>`).join('');
    const providerHTML = state.providerMatches.map((item, itemIndex) => {
      const artwork = artworkURL(item.artwork_url);
      const variants = item.variants.map((variant, variantIndex) => `<button class="search-variant" data-provider-match="${itemIndex}" data-provider-variant="${variantIndex}">＋ ${escapeHTML(variantLabel(variant))}</button>`).join('');
      return `<article class="search-provider-result"><div class="search-provider-main"><span class="search-result-art">${artwork ? `<img src="${escapeHTML(artwork)}" alt="" loading="lazy">` : escapeHTML(item.title.slice(0, 1).toUpperCase())}</span><span class="search-result-copy"><strong>${escapeHTML(item.title)}</strong><small>${item.variants.length} fonte(s) encontrada(s)</small></span></div><div class="search-variants">${variants}</div></article>`;
    }).join('');
    const hasResults = localMatches.length > 0 || state.providerMatches.length > 0;
    popover.innerHTML = `${localHTML ? `<section class="search-popover-section"><div class="search-popover-heading"><span>NA SUA BIBLIOTECA</span><span>${localMatches.length}</span></div>${localHTML}</section>` : ''}${providerHTML || loading ? `<section class="search-popover-section"><div class="search-popover-heading"><span>PROVEDORES CONFIGURADOS</span><span>${loading ? 'BUSCANDO…' : state.providerMatches.length}</span></div>${providerHTML}${loading ? '<div class="search-loading">Consultando as fontes disponíveis…</div>' : ''}</section>` : ''}${!hasResults && !loading ? '<div class="search-no-results">Nada encontrado na biblioteca ou nos provedores.</div>' : ''}`;
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
    button.textContent = 'Adicionando…';
    try {
      const collection = await api('/api/providers/import', { method: 'POST', body: JSON.stringify({ provider_id: variant.provider_id, reference: variant.reference }) });
      await loadCollections();
      $('#global-search-results').classList.add('hidden');
      showToast(`${collection.title} foi adicionado à biblioteca.`);
      openCollection(collection.id);
    } catch (error) {
      button.disabled = false;
      button.textContent = `＋ ${variantLabel(variant)}`;
      showToast(error.message, true);
    }
  }

  function runGlobalSearch(rawQuery) {
    const query = rawQuery.trim();
    clearTimeout(state.searchTimer);
    state.searchAbortController?.abort();
    const normalized = query.toLocaleLowerCase('pt-BR');
    const localMatches = query ? state.collections.filter(collection => collection.title.toLocaleLowerCase('pt-BR').includes(normalized) || collection.items.some(item => item.title.toLocaleLowerCase('pt-BR').includes(normalized))) : state.collections;
    $('#library-grid').innerHTML = localMatches.map(collectionCard).join('') || emptyInline('Nada encontrado na biblioteca');
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

  function openCollection(id) {
    const collection = state.collections.find(item => item.id === id);
    if (!collection) return;
    state.activeCollection = collection;
    const detailArtwork = artworkURL(collection.artwork_url) || (collection.items?.[0]?.source_id === 'local' ? mediaURL('/api/media/thumbnail', collection.items[0].id) : '');
    $('#detail-header').innerHTML = `<div class="detail-art">${detailArtwork ? `<img src="${escapeHTML(detailArtwork)}" alt="">` : ''}<span>${escapeHTML(collection.title.slice(0, 1))}</span></div><div><small>${collection.kind === 'local_folder' ? 'COLEÇÃO LOCAL' : 'COLEÇÃO DE PROVEDOR'}</small><h2>${escapeHTML(collection.title)}</h2><p>${collection.items.length} item(ns) • ${escapeHTML(collection.root_path || collection.source_id)}</p></div>`;
    $('#media-list').innerHTML = collection.items.map((item, index) => `<div class="media-row">
      <button class="media-main" data-play="${escapeHTML(item.id)}"><span class="media-index">${index + 1}</span><span class="media-thumb">${item.source_id === 'local' ? `<img src="${mediaURL('/api/media/thumbnail', item.id)}" alt="">` : ''}<i>▶</i></span><span><strong>${escapeHTML(item.title)}</strong><small>${item.source_id === 'local' ? (item.width ? `${item.width}×${item.height}` : 'Vídeo local') : 'Mídia do provedor'}${item.watched ? ' • Assistido' : item.playback_time ? ' • Em andamento' : ''}</small></span></button>
      <span>${formatDuration(item.duration_seconds)}</span><button class="icon-button" data-more="${escapeHTML(item.id)}">•••</button>
    </div>`).join('') || emptyInline('Esta coleção não possui vídeos');
    $$('#media-list [data-play]').forEach(button => button.addEventListener('click', () => playItem(button.dataset.play)));
    bindImageFallbacks($('#view-detail'));
    navigate('detail');
  }

  async function playItem(id) {
    const item = state.activeCollection?.items.find(candidate => candidate.id === id);
    if (!item) return;
    state.activeItem = item;
    if (item.source_id !== 'local') {
      try {
        const result = await api('/api/providers/play', { method: 'POST', body: JSON.stringify({ item_id: item.id, player: state.settings?.default_player }) });
        showToast(`Reprodução iniciada no ${result.player}.`);
      } catch (error) { showToast(error.message, true); }
      return;
    }
    $('#player-title').textContent = item.title;
    const player = $('#local-player');
    player.src = mediaURL('/api/media/local', item.id);
    player.onloadedmetadata = () => {
      if (item.playback_time > 0 && item.playback_time < player.duration - 20) player.currentTime = item.playback_time;
    };
    $('#player-modal').classList.remove('hidden');
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
    if (!root) return showToast('Escolha uma pasta para indexar.', true);
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
      $('#index-status-title').textContent = status.running ? 'Indexando sua biblioteca…' : status.last_error ? 'Indexação incompleta' : 'Biblioteca atualizada';
      $('#index-status-copy').textContent = status.running ? `${status.files_indexed} de ${status.files_found} arquivo(s) processados` : `${status.files_indexed} vídeo(s) encontrado(s)`;
      if (status.running) state.indexTimer = setTimeout(pollIndex, 900);
      else {
        await loadCollections();
        if (status.last_error) showToast(status.last_error, true); else showToast('Biblioteca local atualizada.');
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
    const path = $('#provider-root-input').value.trim();
    if (!path) return showToast('Informe a pasta do provedor.', true);
    try {
      const payload = await api('/api/providers/discover', { method: 'POST', body: JSON.stringify({ path }) });
      renderCandidates(payload.candidates || []);
      if (!payload.candidates?.length) showToast('Nenhum manifesto compatível foi encontrado.', true);
    } catch (error) {
      showToast(error.message, true);
    }
  }

  function renderCandidates(candidates) {
    $('#provider-candidates').innerHTML = candidates.map(candidate => `<article class="provider-row"><span class="provider-logo">${escapeHTML(candidate.manifest.name.slice(0, 1))}</span><div><strong>${escapeHTML(candidate.manifest.name)}</strong><small>v${escapeHTML(candidate.manifest.version)} • ${escapeHTML(candidate.status)}</small><code>${escapeHTML(candidate.manifest_path)}</code></div><button class="button ${candidate.ready ? 'primary' : 'secondary'}" data-register="${escapeHTML(candidate.manifest_path)}">${candidate.ready ? 'Ativar' : 'Adicionar desativado'}</button></article>`).join('');
    $$('#provider-candidates [data-register]').forEach(button => button.addEventListener('click', async () => {
      try {
        await api('/api/providers/register', { method: 'POST', body: JSON.stringify({ manifest_path: button.dataset.register }) });
        showToast('Provedor adicionado.');
        await loadProviders();
      } catch (error) { showToast(error.message, true); }
    }));
  }

  async function probeWebsiteProvider() {
    const input = $('#website-provider-url');
    const button = $('#probe-website-provider');
    const url = input.value.trim();
    if (!url) return showToast('Informe o endereço HTTPS do site.', true);
    button.disabled = true;
    $('#website-provider-candidate').innerHTML = '<div class="empty-inline">Verificando o manifesto público do site…</div>';
    try {
      const candidate = await api('/api/providers/web/probe', { method: 'POST', body: JSON.stringify({ url }) });
      renderWebsiteProviderCandidate(candidate);
    } catch (error) {
      $('#website-provider-candidate').innerHTML = emptyInline('Este site não expõe uma integração compatível e não foi ativado.');
      showToast(error.message, true);
    } finally {
      button.disabled = false;
    }
  }

  function renderWebsiteProviderCandidate(candidate) {
    const container = $('#website-provider-candidate');
    container.innerHTML = `<article class="provider-row"><span class="provider-logo">${escapeHTML(candidate.manifest.name.slice(0, 1))}</span><div><strong>${escapeHTML(candidate.manifest.name)}</strong><small>Site compatível • v${escapeHTML(candidate.manifest.version)} • ${escapeHTML(candidate.status)}</small><code>${escapeHTML(candidate.origin)}</code></div><button class="button primary" data-register-website="${escapeHTML(candidate.origin)}" data-manifest-fingerprint="${escapeHTML(candidate.fingerprint)}">Ativar site</button></article>`;
    container.querySelector('[data-register-website]').addEventListener('click', async event => {
      const button = event.currentTarget;
      button.disabled = true;
      try {
        await api('/api/providers/web/register', { method: 'POST', body: JSON.stringify({ url: button.dataset.registerWebsite, fingerprint: button.dataset.manifestFingerprint }) });
        showToast('Site compatível ativado como provedor.');
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
      const stateLabel = provider.enabled ? 'Ativo' : 'Aguardando executável';
      return `<article class="provider-row"><span class="provider-logo">${escapeHTML(provider.name.slice(0, 1))}</span><div><strong>${escapeHTML(provider.name)}</strong><small>${website ? 'Website HTTPS' : 'Processo local'} • v${escapeHTML(provider.version)} • ${stateLabel}</small><code>${escapeHTML(location || '')}</code></div><div class="provider-actions">${provider.enabled ? `<button class="button secondary" data-provider-health="${escapeHTML(provider.id)}">Verificar</button>` : '<span class="badge">Inativo</span>'}<button class="button secondary" data-provider-remove="${escapeHTML(provider.id)}" aria-label="Remover provedor">Remover</button></div></article>`;
    }).join('') || emptyInline('Nenhum provedor configurado');
    $$('#installed-providers [data-provider-health]').forEach(button => button.addEventListener('click', async () => {
      button.disabled = true;
      try {
        const health = await api('/api/providers/health', { method: 'POST', body: JSON.stringify({ provider_id: button.dataset.providerHealth }) });
        showToast(health.message || `Estado: ${health.status}`);
      } catch (error) { showToast(error.message, true); }
      finally { button.disabled = false; }
    }));
    $$('#installed-providers [data-provider-remove]').forEach(button => button.addEventListener('click', async () => {
      button.disabled = true;
      try {
        await api(`/api/providers?id=${encodeURIComponent(button.dataset.providerRemove)}`, { method: 'DELETE' });
        showToast('Provedor removido. As coleções já importadas foram preservadas.');
        await loadProviders();
      } catch (error) { showToast(error.message, true); }
      finally { button.disabled = false; }
    }));
  }

  async function loadSettings() {
    state.settings = await api('/api/settings');
    $('#web-session-mode').value = state.settings.privacy?.web_session_mode || 'private';
    $('#keep-web-history').checked = Boolean(state.settings.privacy?.keep_web_history);
    $('#default-player').value = state.settings.default_player || 'integrated';
    if (!$('#media-root-input').value && state.settings.media_roots?.[0]) $('#media-root-input').value = state.settings.media_roots[0];
    if (window.arionDesktop?.available) await window.arionDesktop.setSessionMode(state.settings.privacy?.web_session_mode || 'private');
  }

  async function saveSettings() {
    const next = structuredClone(state.settings || {});
    next.default_player = $('#default-player').value;
    next.privacy = { telemetry_enabled: false, web_session_mode: $('#web-session-mode').value, keep_web_history: $('#keep-web-history').checked };
    try {
      state.settings = await api('/api/settings', { method: 'POST', body: JSON.stringify(next) });
      if (window.arionDesktop?.available) await window.arionDesktop.setSessionMode(next.privacy.web_session_mode);
      showToast('Configurações salvas.');
    } catch (error) { showToast(error.message, true); }
  }

  function setupEvents() {
    $$('.nav-item[data-view]').forEach(button => button.addEventListener('click', () => navigate(button.dataset.view)));
    $$('[data-go]').forEach(button => button.addEventListener('click', () => navigate(button.dataset.go)));
    $('#detail-back').addEventListener('click', () => navigate('library'));
    $('#scan-media').addEventListener('click', startIndex);
    $('#discover-providers').addEventListener('click', discoverProviders);
    $('#probe-website-provider').addEventListener('click', probeWebsiteProvider);
    $('#website-provider-url').addEventListener('keydown', event => { if (event.key === 'Enter') probeWebsiteProvider(); });
    $('#save-settings').addEventListener('click', saveSettings);
    $$('[data-web-platform]').forEach(button => button.addEventListener('click', () => openWebPlatform(button.dataset.webPlatform)));
    $$('[data-web-command]').forEach(button => button.addEventListener('click', () => window.arionDesktop?.navigateWeb(button.dataset.webCommand)));
    $('#clear-web-data').addEventListener('click', async () => {
      if (!state.webPlatform || !window.arionDesktop?.available) return showToast('Abra uma plataforma primeiro.', true);
      try { await window.arionDesktop.clearWebData(state.webPlatform); showToast('Cookies e armazenamento da plataforma foram apagados.'); }
      catch (error) { showToast(error.message, true); }
    });
    $('#close-player').addEventListener('click', closePlayer);
    $('#player-modal').addEventListener('click', event => { if (event.target === $('#player-modal')) closePlayer(); });
    $('#open-external-player').addEventListener('click', async () => {
      if (!state.activeItem) return;
      const preferred = state.settings?.default_player;
      try { await api('/api/player/play', { method: 'POST', body: JSON.stringify({ item_id: state.activeItem.id, player: preferred && preferred !== 'integrated' ? preferred : 'mpv' }) }); showToast('Vídeo enviado ao player externo.'); }
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
    if (!window.arionDesktop?.available) return showToast('Abra o Arion pelo shell Chromium para usar Web Videos.', true);
    try {
      state.webPlatform = platform;
      await window.arionDesktop.openWebPlatform(platform, webSurfaceBounds());
      $$('[data-web-platform]').forEach(button => {
        button.classList.toggle('primary', button.dataset.webPlatform === platform);
        button.classList.toggle('secondary', button.dataset.webPlatform !== platform);
      });
      $('#web-state-label').textContent = platform === 'youtube' ? 'YouTube • sessão isolada' : 'TikTok • sessão isolada';
    } catch (error) { showToast(error.message, true); }
  }

  function setupDesktopBridge() {
    if (!window.arionDesktop?.available) {
      $('#web-state-label').textContent = 'Shell Chromium não iniciado';
      return;
    }
    window.arionDesktop.onWebState(webState => {
      if (webState.platform !== state.webPlatform) return;
      $('#web-state-label').textContent = webState.loading ? 'Carregando…' : (webState.title || 'Sessão isolada');
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
    if (!state.token) {
      showToast('Abra o Arion pelo inicializador seguro.', true);
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
