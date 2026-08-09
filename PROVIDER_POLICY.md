# Política de provedores

Arion é uma galeria e um hospedeiro local de extensões. O projeto principal não hospeda, fornece, verifica, recomenda ou endossa catálogos e provedores de terceiros.

## Responsabilidade do usuário

Ao adicionar um provedor, o usuário deve verificar sua origem, licença, segurança e legitimidade, além de possuir autorização para acessar, reproduzir ou armazenar o conteúdo utilizado.

Um manifesto compatível não significa que o Arion auditou ou aprovou o provedor.

## Limites do projeto principal

O repositório e as distribuições oficiais do Arion:

- não mantêm diretório público de provedores;
- não baixam provedores automaticamente;
- não incluem credenciais, cookies ou catálogos de terceiros;
- não oferecem mecanismos para contornar DRM, autenticação ou paywalls;
- não executam um provedor durante a etapa de descoberta;
- não convertem páginas arbitrárias em scrapers nem interpretam o HTML de sites colados;
- exigem ação explícita antes de registrar ou ativar um executável externo.

Um site somente recebe o papel de provedor quando publica um manifesto compatível na rota HTTPS padronizada e o usuário confirma a ativação. A consulta do manifesto não compartilha cookies, logins ou dados da biblioteca.

Provedores executam como programas separados e são considerados não confiáveis. Metadados, URLs e caminhos retornados devem ser validados pelo núcleo antes do uso.

Adaptadores mantidos em outros repositórios não passam a fazer parte do Arion apenas por implementarem o protocolo. Cada adaptador conserva autoria, licença, política de distribuição e responsabilidade próprias.

## Marcas e conteúdo

Arion não é afiliado a plataformas, estúdios ou distribuidores de mídia. Nomes e marcas pertencem aos seus respectivos titulares.

Esta política explica o papel técnico do Arion e não substitui aconselhamento jurídico nem altera as permissões concedidas pela GPLv3. Ela também não garante, por si só, ausência de responsabilidade em todas as jurisdições.
