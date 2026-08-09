# Exemplo de provedor web

Este servidor demonstra o kit público sem integrar ou citar qualquer catálogo real.

```bash
go run ./examples/web-provider
```

Por padrão ele escuta somente em `127.0.0.1:9080`. Esse modo serve para desenvolvimento e testes locais; o aplicativo Arion rejeita HTTP, localhost e IPs por segurança.

Para um teste público, coloque o processo atrás de um proxy reverso HTTPS no domínio do provedor. O manifesto ficará em `/.well-known/arion-provider.json` e o RPC em `/arion/rpc`. Não redirecione essas rotas.

`ARION_EXAMPLE_MEDIA_URL` pode indicar uma mídia HTTPS de teste controlada pelo desenvolvedor. A URL padrão usa o TLD reservado `.invalid` e, intencionalmente, não reproduz conteúdo.

Antes da publicação:

```bash
go run ./cmd/arion-provider-validator -url https://provedor.example
```

O exemplo não inclui autenticação, rate limiting, armazenamento nem catálogo. Essas responsabilidades permanecem com o operador do provedor.
