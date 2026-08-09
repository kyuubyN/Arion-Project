# Prova de reinstalação do provedor

Data: 9 de agosto de 2026.

1. O GoAnime foi clonado por SSH em uma pasta temporária, no commit `7ad63616918ab05ee39b57e0a6fca9abfb5d0306`.
2. Somente os três arquivos do adaptador foram aplicados: `arion-provider.json`, `bin/goanime-arion-provider` e `cmd/arion-provider/main.go`.
3. O comando do adaptador compilou contra o clone remoto, sem depender da antiga pasta `arion/`.
4. O Arion encontrou exatamente um manifesto, classificou-o como pronto, registrou-o e recebeu `status: ok` com quatro fontes ativas.

A cópia de trabalho original não foi apagada. O repositório remoto ainda precisa receber os três arquivos em um commit próprio antes que um clone remoto puro os contenha automaticamente.
