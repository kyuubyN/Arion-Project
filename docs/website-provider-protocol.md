# Provedores web Arion — protocolo 1

Um endereço informado pelo usuário não vira automaticamente um scraper. O Arion deriva apenas a origem HTTPS e consulta um único recurso de descoberta:

```text
https://provedor.example/.well-known/arion-provider.json
```

Se esse recurso não existir ou não for compatível, o site não é ativado como provedor. Ele ainda pode ser acessado no modo Web Videos, que possui uma fronteira de segurança separada.

## Manifesto

O servidor responde com `application/json` ou `application/arion-provider+json` e no máximo 256 KiB:

```json
{
  "schema_version": 1,
  "kind": "website",
  "id": "example.web",
  "name": "Example Web Provider",
  "version": "1.0.0",
  "protocol_version": 1,
  "rpc_path": "/arion/rpc",
  "capabilities": [
    "catalog.search",
    "collection.resolve",
    "item.resolve"
  ],
  "author": "Example Author",
  "license": "GPL-3.0-only"
}
```

`rpc_path` é um caminho absoluto na mesma origem. URLs completas, query strings, fragmentos, travessia por `..` e redirecionamentos são rejeitados.

## Transporte

As estruturas e os métodos são os mesmos descritos no [protocolo principal](provider-protocol.md), mas a requisição JSON-RPC usa `POST` HTTPS. O endpoint deve responder diretamente com HTTP 200 e conteúdo JSON.

O Arion não envia cookies, login, token da API local, cabeçalho `Authorization`, biblioteca, caminhos ou variáveis de ambiente. Somente os parâmetros da operação iniciada pelo usuário — por exemplo, o texto de uma busca — chegam ao provedor.

## Fronteiras de segurança

- somente domínio público e porta HTTPS padrão;
- endereços IP, nomes locais e redes privadas são bloqueados;
- DNS é verificado novamente durante a conexão para reduzir risco de rebinding;
- manifesto e RPC permanecem na mesma origem;
- nenhum redirecionamento é seguido;
- respostas possuem limite de tamanho, timeout e validação estrutural;
- a ativação é explícita, o manifesto é consultado novamente pelo backend e sua impressão SHA-256 precisa coincidir com a versão exibida;
- IDs não podem substituir silenciosamente outra instalação ou origem.

O protocolo não executa JavaScript do site, não interpreta HTML e não tenta descobrir seletores de páginas. Uma integração para um site sem manifesto deve ser publicada pelo próprio site ou instalada separadamente como um adaptador local, mantendo o núcleo do Arion neutro.

Desenvolvedores podem usar o [kit público, servidor de exemplo e validador de conformidade](provider-development-kit.md).
