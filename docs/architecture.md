# Arquitetura do Arion

```text
Interface local
    │
    ├── API local autenticada ── Backend Go
    │                              ├── Galeria
    │                              ├── Indexador local
    │                              ├── Players
    │                              └── Gerenciador de provedores
    │
    └── Web Videos
           ├── Sessão Chromium isolada A
           └── Sessão Chromium isolada B

Provedores externos ── protocolo versionado
                      ├── processo local separado
                      └── endpoint HTTPS opt-in na mesma origem
```

O núcleo não possui tipo privilegiado de conteúdo. Uma pasta local, um canal web ou uma série retornada por um provedor são coleções com itens de mídia.

## Persistência

O schema inicial usa JSON atômico para permitir migração e inspeção simples. O catálogo novo fica em `gallery.json`; arquivos antigos não são sobrescritos. Uma migração para SQLite poderá ocorrer depois que o schema genérico estabilizar.

## Fronteiras

Sites remotos nunca compartilham renderer, preload, sessão ou IPC com a interface local. Provedores locais não são carregados como bibliotecas: são processos externos. Provedores web usam requisições HTTPS sem cookies a um endpoint opt-in, limitado à mesma origem do manifesto. Nenhuma URL colada recebe acesso de provedor automaticamente.

## Fluxo de busca

1. Resultados da biblioteca são filtrados no renderer sem rede.
2. Após 220 ms sem nova digitação, o backend consulta provedores com modo `preview`.
3. A prévia é mostrada assim que uma fonte responde.
4. Uma consulta `complete` continua em segundo plano e substitui a prévia com todas as variantes disponíveis.
5. Títulos equivalentes são agrupados e as variantes mantêm referências opacas específicas de cada provedor.

## Capas

URLs externas nunca são colocadas diretamente no renderer privilegiado. O backend valida DNS e endereço de destino, bloqueia redes locais, limita o download, aceita somente formatos raster e armazena o resultado em cache com permissão privada.
