# Segurança de provedores

Descoberta e ativação são etapas distintas. No provedor local, descobrir significa ler um manifesto limitado e ativar autoriza executar o programa indicado. No provedor web, descobrir faz somente um `GET` no manifesto HTTPS padronizado; ativar autoriza chamadas JSON-RPC sem cookies ao endpoint de mesma origem.

O Arion protege o protocolo com allowlist de métodos e capacidades, validação de caminhos e links simbólicos, ambiente reduzido, limite de saída, timeout e encerramento de processo. URLs de capas e streams são verificadas antes do uso.

Para sites, o Arion exige domínio público, HTTPS na porta padrão, bloqueia SSRF e DNS rebinding, não segue redirecionamentos e não interpreta HTML. Um site sem manifesto compatível não ganha acesso de provedor.

Limite importante: o processo ainda herda as permissões da conta do usuário no sistema operacional. O protocolo reduz acoplamento e vazamento acidental de segredos, mas não transforma código malicioso em código seguro. Sandboxing nativo permanece trabalho futuro.
