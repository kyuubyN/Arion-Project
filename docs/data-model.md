# Modelo de dados

`Collection` é a unidade neutra da biblioteca. `kind` diferencia `local_folder`, `web` e `provider`; nenhuma delas recebe privilégios de domínio.

`MediaItem` representa vídeo, episódio, vídeo curto ou item web. Arquivos locais usam `path`; provedores usam `provider_reference`, uma referência opaca devolvida somente ao processo que a criou.

O catálogo fica em `~/.config/arion/gallery.json`. Configurações ficam em `settings-v2.json`. Miniaturas e capas são derivadas e ficam em `~/.cache/arion`, podendo ser recriadas.

Remover uma coleção não apaga o arquivo original. Uma nova indexação preserva progresso quando o identificador estável do item não muda.
