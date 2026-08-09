# Arion

Arion é uma galeria de mídia local, privada e extensível. Ele organiza vídeos do computador em coleções e oferece um protocolo neutro para provedores escolhidos pelo usuário.

O núcleo não inclui catálogo de fontes, não recomenda serviços de terceiros e não envia a biblioteca para a nuvem.

Versões oficiais para Linux e Windows ficam em [Releases](https://github.com/kyuubyN/Arion-Project/releases).

## Estado atual

- Projeto e módulo Go independentes.
- Biblioteca genérica de coleções e itens de mídia.
- Descoberta local de vídeos com FFprobe e thumbnails com FFmpeg.
- Reprodução HTML5, MPV e Celluloid.
- Descoberta e execução de provedores locais por `arion-provider.json` e JSON-RPC 2.0.
- Provedores web opt-in por manifesto HTTPS padronizado, sem scraping de URLs arbitrárias.
- Kit Go público, servidor neutro de exemplo e validador de conformidade para provedores web.
- Busca unificada: biblioteca imediata, prévia rápida e enriquecimento completo dos provedores.
- Capas externas validadas contra SSRF, limitadas e armazenadas em cache local.
- API local autenticada por token efêmero.
- Telemetria inexistente e desativada por definição.
- Shell Electron/Chromium com sessões web isoladas para `Web Videos`.
- Fallback GTK/WebKit quando o runtime Electron ainda não estiver instalado.
- Pacotes x64 para Linux e Windows, incluindo executável portátil para Windows.

## Executar em desenvolvimento

Requisitos de desenvolvimento: Go 1.22+ e Node.js. FFmpeg/FFprobe são opcionais, mas recomendados para metadados e miniaturas. GTK 3 e WebKit2GTK são usados somente pelo fallback.

```bash
npm install
```

```bash
./arion-launcher.sh
```

No Windows, use o instalador ou o executável portátil publicados para a versão. Consulte [Instalação e desenvolvimento](docs/installation.md) para requisitos, caminhos de dados e compilação.

Testes:

```bash
go test ./...
go vet ./...
node --check frontend/app.js
npm run sbom
```

## Dados locais

- Configurações e catálogo: `~/.config/arion`
- Miniaturas: `~/.cache/arion`

O Arion somente indexa pastas explicitamente autorizadas. Arquivos não são copiados, movidos ou enviados.

## Provedores

O Arion reconhece um provedor apenas por seu manifesto. Provedores web precisam expor `/.well-known/arion-provider.json`; colar uma URL comum não concede privilégios nem inicia scraping. Consulte [o protocolo de provedores](docs/provider-protocol.md), [o protocolo web](docs/website-provider-protocol.md) e a [política de provedores](PROVIDER_POLICY.md).

O núcleo não contém integrações específicas. Adaptadores compatíveis vivem fora do Arion e conversam com ele somente pelo protocolo público.

## Documentação

- [Arquitetura](docs/architecture.md)
- [Instalação e desenvolvimento](docs/installation.md)
- [Modelo de dados](docs/data-model.md)
- [Protocolo de provedores web](docs/website-provider-protocol.md)
- [Kit de desenvolvimento de provedores](docs/provider-development-kit.md)
- [Segurança do modo web](docs/security/web-mode.md)
- [Segurança de provedores](docs/security/providers.md)
- [Prova de reinstalação](docs/reclone-proof.md)
- [Checklist de lançamento](docs/release-checklist.md)

## Licença

Arion é software livre sob a [GNU General Public License v3.0 only](LICENSE). Componentes reaproveitados e dependências mantêm seus próprios avisos em [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

<sub>Transparência: algumas imagens da identidade visual do Arion foram geradas ou transformadas com auxílio de IA a partir de materiais originais do projeto.</sub>
