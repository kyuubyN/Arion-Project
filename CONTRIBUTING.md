# Contribuindo

Contribuições ao núcleo devem permanecer neutras quanto a catálogos e serviços. Integrações específicas pertencem ao repositório do respectivo adaptador e usam somente o protocolo público.

Antes de enviar uma mudança, execute testes Go, vet, verificações JavaScript e atualize documentação e testes de segurança afetados. Não inclua cookies, tokens, URLs privadas, mídia protegida ou dados reais de usuários em issues e fixtures.

Provedores web devem reutilizar `providerkit` quando possível e passar pelo `arion-provider-validator`. Exemplos no núcleo usam apenas domínios reservados e dados fictícios.

Ao contribuir, você concorda que sua contribuição seja distribuída sob GPL-3.0-only para o Arion. Código copiado de terceiros exige licença compatível, atribuição e registro em `THIRD_PARTY_NOTICES.md`.
