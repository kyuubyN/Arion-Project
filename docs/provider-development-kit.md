# Kit de desenvolvimento de provedores

O pacote Go público `github.com/kyuubyN/Arion-Project/providerkit` mantém os tipos, limites e validações usados pelo próprio Arion. Isso evita implementar manualmente o envelope JSON-RPC e reduz divergências entre o aplicativo e um provedor.

## Servidor mínimo

```go
service := providerkit.Service{
    Manifest: providerkit.Manifest{
        SchemaVersion: providerkit.SchemaVersion,
        Kind: providerkit.WebsiteKind,
        ID: "example.web",
        Name: "Example Provider",
        Version: "1.0.0",
        ProtocolVersion: providerkit.ProtocolVersion,
        RPCPath: "/arion/rpc",
        Capabilities: []string{"catalog.search"},
    },
    Search: searchCatalog,
}

handler, err := providerkit.NewHandler(service)
```

O handler serve o manifesto em `/.well-known/arion-provider.json` e o RPC no caminho declarado. Parâmetros desconhecidos, métodos não declarados, corpos excessivos e resultados inválidos são rejeitados.

Consulte o [servidor completo de exemplo](../examples/web-provider/main.go). Ele usa somente dados fictícios e escuta em loopback para desenvolvimento. A descoberta do aplicativo exige publicação em domínio HTTPS público.

## Validador

Validar apenas um arquivo local:

```bash
go run ./cmd/arion-provider-validator -file ./examples/web-provider/arion-provider.example.json
```

Validar um provedor publicado:

```bash
go run ./cmd/arion-provider-validator -url https://provedor.example
```

No segundo modo a ferramenta verifica:

1. HTTPS público e proteção contra destinos privados;
2. manifesto na rota `/.well-known` e tipo de conteúdo;
3. schema, capacidades, tamanhos e caminho RPC;
4. ausência de redirecionamentos e permanência na mesma origem;
5. equivalência entre manifesto e `provider.describe`;
6. envelope e resultado de `provider.health`.

O pacote Linux inclui o binário em `resources/tools/arion-provider-validator`.

## Responsabilidades do operador

O kit não implementa scraping, autenticação, catálogo, rate limiting, autorização ou armazenamento. O operador deve disponibilizar apenas conteúdo que tenha direito de oferecer, proteger o endpoint contra abuso e publicar política de privacidade adequada. Segredos nunca devem ser colocados no manifesto ou em referências retornadas ao cliente.
