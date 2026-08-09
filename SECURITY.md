# Segurança

## Modelo de confiança

- Interface local: confiável, mas sem acesso irrestrito ao sistema.
- Backend local: confiável e autenticado por token efêmero.
- Sites web: não confiáveis e isolados do IPC e dos arquivos.
- Provedores: processos locais ou endpoints HTTPS externos não confiáveis.
- Arquivos de mídia: dados não confiáveis processados com limites e timeouts.

## Garantias atuais

- Servidor vinculado somente a `127.0.0.1` em porta dinâmica.
- Toda rota sensível exige `Authorization: Bearer`.
- Requisições cross-origin são negadas.
- Token não é persistido em disco.
- Catálogo e configurações são gravados atomicamente com permissão `0600`.
- Raiz do filesystem e pasta pessoal inteira não podem ser indexadas.
- Links simbólicos não são seguidos pelo indexador.
- Manifestos têm limite de tamanho e profundidade de busca.
- Executáveis de provedores precisam permanecer dentro da pasta do manifesto.
- Links simbólicos de executáveis não podem escapar da pasta do manifesto.
- Chamadas de provedores têm capacidades explícitas, timeout, resposta máxima de 8 MiB e encerramento do grupo de processos.
- O ambiente do processo remove o token da API e outras variáveis não autorizadas.
- Provedores web exigem manifesto `/.well-known`, HTTPS público e endpoint RPC de mesma origem.
- Chamadas web não usam cookies ou credenciais, não seguem redirecionamentos e passam por proteção contra SSRF e DNS rebinding.
- Capas remotas passam por bloqueio de endereços privados, limite de 6 MiB e validação de formato raster.

## Isolamento do Chromium

- `nodeIntegration: false`;
- `contextIsolation: true`;
- sandbox de renderização;
- sessões separadas por plataforma;
- IPC indisponível para conteúdo remoto;
- permissões de câmera, microfone, localização, captura de tela e filesystem negadas por padrão;
- navegação e novas janelas controladas por allowlist.

## Limites conhecidos

Os pacotes Windows de desenvolvimento ainda não são assinados digitalmente. A autenticidade deve ser conferida pelo checksum publicado. A assinatura de código é requisito para uma versão estável, mas não substitui a revisão dos binários e dependências.

Separação de processo não equivale a sandbox completa do sistema operacional. Um provedor local ativado ainda executa com as permissões da conta do usuário. Somente instale código cuja origem e licença você verificou. Sandboxing com mecanismos como Bubblewrap, namespaces ou perfis nativos é trabalho futuro e deverá ser opcional para não quebrar portabilidade.

Um provedor web não acessa arquivos, mas recebe as consultas e referências enviadas a ele. HTTPS protege o transporte, não torna o operador do site confiável nem garante a legitimidade do conteúdo oferecido.

O modo Web Videos reduz o acesso do site ao Arion, mas não esconde IP, conta, cookies persistentes ou atividade da própria plataforma acessada. O conteúdo remoto usa uma camada nativa `WebContentsView`; a interface fixa seu contêiner e remove essa camada ao trocar de tela.

## Relato de vulnerabilidade

Não publique detalhes exploráveis antes de uma correção. Abra inicialmente um canal privado com os mantenedores do repositório e inclua versão, plataforma, impacto e passos mínimos de reprodução.
