# Política de privacidade do Arion

## Princípios

O Arion funciona localmente e adota minimização de dados:

- não possui telemetria;
- não exige conta Arion;
- não envia biblioteca, histórico ou caminhos de arquivos para um servidor Arion;
- somente indexa diretórios escolhidos pelo usuário;
- gera metadados e miniaturas localmente;
- não compartilha dados entre sessões web e a biblioteca local.

## Vídeos pessoais

O indexador lê nomes, tamanho, duração, resolução e data de modificação de arquivos dentro das pastas autorizadas. O aplicativo não move nem apaga os vídeos durante a indexação.

## Web Videos

Conteúdo web é carregado em sessões Chromium separadas da interface local. Quando um usuário acessa uma plataforma, essa plataforma pode coletar dados conforme sua própria política. O Arion não pode tornar privado perante a plataforma um acesso feito diretamente ao site dela.

O modo privado usa partições efêmeras. O modo persistente é opcional e mantém cookies e dados de login no dispositivo. O comando “Limpar dados da sessão” remove cookies, cache e armazenamento da plataforma selecionada.

## Provedores e capas

Provedores locais escolhidos pelo usuário são programas externos e podem fazer suas próprias conexões. O Arion não envia a biblioteca local a eles, mas o sistema operacional ainda permite que um processo autorizado leia o que as permissões da conta permitirem.

Provedores web recebem apenas os parâmetros das operações iniciadas pelo usuário, como o texto pesquisado ou uma referência opaca. As chamadas não incluem cookies, logins, token da API local, caminhos ou dados da biblioteca. O servidor do provedor ainda pode registrar IP, horário e conteúdo dessas operações conforme sua própria política.

Capas remotas solicitadas pela interface passam pelo backend, possuem limite de tamanho e são mantidas em `~/.cache/arion/artwork`.

## Exclusão

O usuário poderá remover coleções sem apagar os arquivos originais. Dados do Arion podem ser apagados removendo os diretórios `~/.config/arion` e `~/.cache/arion` enquanto o aplicativo estiver fechado.
