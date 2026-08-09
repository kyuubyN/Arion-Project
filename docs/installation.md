# Instalação e desenvolvimento

## Requisitos

- Go 1.22 ou superior, somente para desenvolvimento;
- Node.js 22.12 ou superior e npm, somente para desenvolvimento;
- FFmpeg/FFprobe para metadados e miniaturas;
- MPV para reprodução externa no Linux ou Windows;
- Celluloid como alternativa exclusiva do Linux.

## Windows

O Arion oferece duas variantes x64:

- `Arion-Setup-<versão>-x64.exe`: instala por usuário, cria atalhos e permite escolher a pasta;
- `Arion-Portable-<versão>-x64.exe`: abre sem instalação.

As configurações e o catálogo ficam em `%APPDATA%\arion`. Miniaturas e capas em cache ficam em `%LOCALAPPDATA%\arion`. Mídia pessoal nunca é movida para essas pastas.

Os artefatos de desenvolvimento ainda não possuem assinatura de código. Por isso, o Microsoft Defender SmartScreen pode pedir confirmação antes da primeira abertura. Confira a versão e o checksum publicado; não desative a proteção do Windows.

Para construir no Windows:

```powershell
npm ci
go test ./...
npm run check
npm run dist:windows
```

O build gera o instalador e a edição portátil em `dist`. O workflow `windows-build.yml` repete esses testes em Windows nativo.

Para produzir somente o `.exe` portátil a partir de Linux x64, use `npm run dist:windows-portable`. O instalador deve ser publicado apenas depois do teste nativo automatizado.

## Linux

```bash
npm install
./arion-launcher.sh
```

O inicializador recompila o backend quando um arquivo Go é mais novo que o binário, instala a entrada desktop no perfil do usuário e inicia Electron. `ELECTRON_RUN_AS_NODE` é removida somente para o processo iniciado.

As configurações e o catálogo ficam em `~/.config/arion`; miniaturas e capas em cache ficam em `~/.cache/arion`.

## Adicionar mídia pessoal

Abra “Vídeos pessoais”, informe uma pasta específica e confirme a indexação. A raiz do filesystem, a pasta pessoal inteira e links simbólicos são recusados.

## Adicionar um provedor

Abra “Provedores”, informe a pasta escolhida e verifique o manifesto. Leia nome, licença, caminho e estado antes de ativar. O Arion não baixa nem recomenda provedores.

Para um provedor web, informe o endereço principal HTTPS em “Adicionar site compatível”. O site precisa publicar o manifesto padronizado; páginas comuns não são interpretadas nem raspadas.

## Desenvolver um provedor web

O repositório inclui o pacote `providerkit`, um servidor neutro de exemplo e o validador:

```bash
go run ./cmd/arion-provider-validator -file ./examples/web-provider/arion-provider.example.json
go run ./examples/web-provider
```

Consulte o [kit de desenvolvimento](provider-development-kit.md) antes de publicar o endpoint em HTTPS.
