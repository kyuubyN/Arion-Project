# Protocolo de provedores Arion — versão 1

## Descoberta local

O usuário seleciona uma pasta. O Arion procura `arion-provider.json` com profundidade máxima de quatro diretórios, ignorando `.git`, `node_modules`, `vendor`, `dist` e pastas ocultas. A descoberta apenas lê JSON; nenhum programa é executado nessa etapa.

```json
{
  "schema_version": 1,
  "id": "example.local",
  "name": "Example Provider",
  "version": "1.0.0",
  "protocol_version": 1,
  "executable": "bin/example-provider",
  "capabilities": [
    "catalog.search",
    "collection.resolve",
    "item.resolve"
  ],
  "author": "Example Author",
  "license": "GPL-3.0-only"
}
```

O executável deve ser relativo ao manifesto, possuir permissão de execução e permanecer dentro da pasta mesmo após a resolução de links simbólicos. IDs usam letras minúsculas, números, ponto, hífen ou sublinhado.

## Transporte

Cada chamada inicia um processo novo. O Arion envia uma única requisição JSON-RPC 2.0 terminada por nova linha em `stdin`; o provedor devolve uma única resposta em `stdout` e encerra. Logs pertencem exclusivamente a `stderr`.

```json
{"jsonrpc":"2.0","id":"1","method":"provider.health","params":null}
```

```json
{"jsonrpc":"2.0","id":"1","result":{"status":"ok","message":"Pronto"}}
```

O núcleo limita respostas a 8 MiB, `stderr` a 32 KiB, aplica timeout por método e encerra o grupo do processo. O ambiente contém somente variáveis operacionais autorizadas e `ARION_PROVIDER_PROTOCOL=1`; o token da API local não é repassado.

## Métodos

### `provider.describe`

Retorna identidade, versão do protocolo e capacidades. Não exige capacidade declarada.

### `provider.health`

Retorna `status` igual a `ok`, `degraded` ou `unavailable`, com `message` opcional.

### `catalog.search`

Exige `catalog.search`.

```json
{"query":"exemplo","limit":20,"mode":"preview"}
```

`mode` pode ser `preview` ou `complete`. O resultado contém `items`; cada item exige `id`, `title` e pelo menos uma variante utilizável. Uma variante contém `id`, `label` e uma `reference` opaca que será devolvida ao mesmo provedor.

### `collection.resolve`

Exige `collection.resolve` e recebe a referência de uma variante. Retorna coleção com `id`, `title`, capa opcional e até 5.000 itens. Cada item exige referência opaca própria.

### `item.resolve`

Exige `item.resolve`. Recebe referência e qualidade opcional. Retorna URL HTTP(S), MIME, headers opcionais e expiração opcional. O Arion valida o destino e os headers antes de iniciar um player externo.

## Compatibilidade

Campos desconhecidos são rejeitados no manifesto e no envelope JSON-RPC. Mudanças incompatíveis exigem incremento de `protocol_version`. Capacidades não declaradas nunca podem ser chamadas.

## Transporte web

O mesmo modelo de dados também pode ser exposto por um site que adote explicitamente o protocolo. A descoberta usa exclusivamente `/.well-known/arion-provider.json` em HTTPS; não há scraping genérico. Consulte [Provedores web Arion](website-provider-protocol.md).
