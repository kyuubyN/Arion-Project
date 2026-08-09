# Checklist de lançamento

- [ ] Revisar alterações locais e separar commits do Arion e dos adaptadores.
- [ ] Executar `go test ./...`, `go vet ./...` e `npm run check`.
- [ ] Gerar `SBOM.spdx.json` com `npm run sbom`.
- [ ] Confirmar GPL-3.0-only, créditos e avisos de terceiros no artefato.
- [ ] Validar ícone, `StartupWMClass`, nome do aplicativo e entrada desktop.
- [ ] Executar o workflow de Windows e testar instalador e portátil em um Windows 10/11 x64 limpo.
- [ ] Confirmar ícone no executável, na barra de tarefas, no menu Iniciar e no desinstalador do Windows.
- [ ] Confirmar backend, validador, indexação, player integrado e MPV no Windows.
- [ ] Testar indexação em pasta vazia, mídia corrompida e diretório sem permissão.
- [ ] Testar Web Videos em sessão privada e persistente, limpeza de dados e fullscreen.
- [ ] Testar provedor saudável, lento, com saída excessiva, manifesto alterado e URL SSRF.
- [ ] Executar o validador contra o exemplo local e um endpoint HTTPS de homologação.
- [ ] Testar pacote em um perfil de usuário limpo.
- [ ] Assinar checksum dos artefatos e publicar notas de versão.
- [ ] Assinar os executáveis Windows antes de classificá-los como uma versão estável.
