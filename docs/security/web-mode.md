# Segurança do modo Web Videos

YouTube e TikTok são carregados em `WebContentsView` separados do renderer local. `nodeIntegration` fica desativado, `contextIsolation` e sandbox ficam ativos, e o preload com IPC existe somente na interface local.

Cada plataforma possui partição própria. O modo privado usa partição efêmera; o persistente mantém login. Navegações e janelas são limitadas aos domínios necessários de cada plataforma. Downloads, câmera, microfone, geolocalização e captura de tela são negados; fullscreen é permitido somente para origem autorizada.

A view nativa fica acima do HTML por característica do Electron. Por isso a tela Web Videos não rola: o contêiner é fixo e sincronizado ao redimensionamento. A view é desanexada ao sair da tela para não cobrir outras áreas.

Esse isolamento protege o Arion do site, mas não torna o acesso anônimo perante a plataforma.
