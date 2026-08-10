<div align="center">

**Idioma / Language:** Português | [English](README.md)

<img src="./assets/icon.png" alt="Ícone" width="140">

# Arion

**Galeria de mídia local, privada e extensível para desktop.**

Arion organiza vídeos armazenados no computador em coleções e define um protocolo neutro (`arion-provider.json` + JSON-RPC 2.0) para que provedores de metadados — locais ou web — sejam escolhidos e conectados pelo próprio usuário.

O núcleo **não** embute catálogo de fontes, **não** recomenda serviços de terceiros e **não** envia a biblioteca para nuvem alguma.

[![License: GPL v3](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)
[![Releases](https://img.shields.io/badge/releases-linux%20%7C%20windows-informational)](https://github.com/kyuubyN/Arion-Project/releases)

</div>

## Sumário

- [Interface](#interface)
- [Estado atual](#estado-atual)
- [Arquitetura em resumo](#arquitetura-em-resumo)
- [Requisitos](#requisitos)
- [Executar em desenvolvimento](#executar-em-desenvolvimento)
- [Testes e verificação](#testes-e-verificação)
- [Dados locais](#dados-locais)
- [Provedores](#provedores)
- [Documentação](#documentação)
- [Licença](#licença)

## Interface

<div align="center">

### Início & Galeria de Mídia
<img src="./assets/arion-home-preview.png" alt="Arion - Tela Inicial" width="800">

<br><br>

### Web Videos (Sessões Isoladas) & Configurações
<img src="./assets/arion-webvideos-preview.png" alt="Arion - Web Videos" width="800">
<br><br>
<img src="./assets/arion-settings-preview.png" alt="Arion - Configurações e Privacidade" width="800">

</div>

## Estado atual

| Área | Detalhe |
|---|---|
| Projeto | Módulo Go independente |
| Biblioteca | Modelo genérico de coleções e itens de mídia |
| Descoberta local | Varredura de vídeos com **FFprobe**, geração de thumbnails com **FFmpeg** |
| Reprodução | HTML5, **MPV** e **Celluloid** |
| Provedores locais | Descoberta e execução via `arion-provider.json` + **JSON-RPC 2.0** |
| Provedores web | Opt-in, manifesto HTTPS padronizado — sem scraping de URLs arbitrárias |
| Kit de desenvolvimento | Kit Go público, servidor neutro de exemplo e validador de conformidade para provedores web |
| Busca | Unificada em três estágios: biblioteca imediata → prévia rápida → enriquecimento completo via provedores |
| Capas externas | Validação anti-SSRF, limite de tamanho/taxa e cache local |
| API | Local, autenticada por **token efêmero** por sessão |
| Telemetria | Inexistente — desativada por definição, não por configuração |
| Shell desktop | Electron/Chromium com sessões web **isoladas** para a aba `Web Videos` |
| Fallback | GTK 3 / WebKit2GTK quando o runtime Electron não está instalado |
| Empacotamento | Pacotes x64 para Linux e Windows, incluindo executável portátil Windows |

## Arquitetura em resumo

```
┌─────────────┐      JSON-RPC 2.0       ┌──────────────────────┐
│   Núcleo Go  │ ◄─────────────────────► │ Provedores locais     │
│  (biblioteca,│      arion-provider.json│ (processo próprio)    │
│   API local, │                         └──────────────────────┘
│   cache)     │      HTTPS + manifesto  ┌──────────────────────┐
│              │ ◄─────────────────────► │ Provedores web         │
└──────┬───────┘      /.well-known/...   │ (opt-in por domínio)   │
       │                                 └──────────────────────┘
       │ Token efêmero (API local)
       ▼
┌─────────────┐
│ Shell Electron/Chromium │ ── fallback ── GTK3/WebKit2GTK
│ (UI, sessões isoladas)  │
└─────────────┘
```

- **Núcleo Go**: dono da biblioteca, do cache de miniaturas/capas e da API local autenticada.
- **Provedores**: processos ou serviços externos, descobertos apenas por manifesto — o núcleo nunca acessa uma origem sem manifesto válido.
- **Shell**: camada de apresentação; a aba `Web Videos` roda em sessões Chromium isoladas do restante do app.

Detalhes completos em [Arquitetura](docs/architecture.md) e [Modelo de dados](docs/data-model.md).

## Requisitos

| Componente | Necessário para | Obrigatório |
|---|---|---|
| Go 1.22+ | Build do núcleo | Sim |
| Node.js | Build/execução do shell Electron | Sim |
| FFmpeg / FFprobe | Metadados e miniaturas de vídeo | Recomendado |
| GTK 3 / WebKit2GTK | Fallback quando o Electron não está instalado | Somente no fallback |

## Executar em desenvolvimento

```bash
npm install
./arion-launcher.sh
```

No Windows, use o instalador ou o executável portátil publicados para a versão desejada em [Releases](https://github.com/kyuubyN/Arion-Project/releases).

Para requisitos detalhados, caminhos de dados por plataforma e instruções de compilação, veja [Instalação e desenvolvimento](docs/installation.md).

## Testes e verificação

```bash
go test ./...          # testes unitários/integração do núcleo Go
go vet ./...            # análise estática do código Go
node --check frontend/app.js   # validação sintática do frontend
npm run sbom             # geração da SBOM do projeto
```

## Dados locais

| Tipo de dado | Caminho |
|---|---|
| Configurações e catálogo | `~/.config/arion` |
| Miniaturas | `~/.cache/arion` |

Arion indexa **apenas** pastas explicitamente autorizadas pelo usuário. Os arquivos de mídia nunca são copiados, movidos ou enviados para fora da máquina.

## Provedores

Um provedor só é reconhecido pelo seu **manifesto**, nunca por heurística ou URL solta:

- **Provedores locais**: descobertos e executados via `arion-provider.json` e chamadas JSON-RPC 2.0.
- **Provedores web**: precisam expor `/.well-known/arion-provider.json` em HTTPS. Colar uma URL comum na interface não concede privilégio algum nem dispara scraping — sem manifesto válido, não há integração.

O núcleo **não contém integrações específicas de terceiros**. Adaptadores compatíveis são desenvolvidos fora do repositório do Arion e conversam com ele exclusivamente pelo protocolo público.

Referências técnicas:

- [Protocolo de provedores (local)](docs/provider-protocol.md)
- [Protocolo de provedores web](docs/website-provider-protocol.md)
- [Política de provedores](PROVIDER_POLICY.md)
- [Kit de desenvolvimento de provedores](docs/provider-development-kit.md)

## Documentação

| Documento | Conteúdo |
|---|---|
| [Arquitetura](docs/architecture.md) | Componentes internos e fluxo de dados |
| [Instalação e desenvolvimento](docs/installation.md) | Requisitos, build e caminhos por plataforma |
| [Modelo de dados](docs/data-model.md) | Esquema de coleções, itens e metadados |
| [Protocolo de provedores web](docs/website-provider-protocol.md) | Especificação do manifesto HTTPS |
| [Kit de desenvolvimento de provedores](docs/provider-development-kit.md) | SDK Go, servidor de exemplo e validador de conformidade |
| [Segurança do modo web](docs/security/web-mode.md) | Isolamento de sessões e superfície de ataque do shell |
| [Segurança de provedores](docs/security/providers.md) | Modelo de confiança e mitigação de SSRF |
| [Prova de reinstalação](docs/reclone-proof.md) | Verificação de reprodutibilidade do build |
| [Checklist de lançamento](docs/release-checklist.md) | Passos para publicar uma release |

## Licença

Arion é software livre sob a [GNU General Public License v3.0 only](LICENSE). Componentes reaproveitados e dependências mantêm seus próprios avisos em [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

<sub>Transparência: algumas imagens da identidade visual do Arion foram geradas ou transformadas com auxílio de IA a partir de materiais originais do projeto.</sub>
