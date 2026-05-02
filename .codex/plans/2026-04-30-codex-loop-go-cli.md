# Accepted Plan: `codex-loop` Open Source CLI + Codex Plugin

Accepted on 2026-04-30 after the user asked to implement the proposed plan.

## Summary
- Transformar o repo em um projeto open source Go usando o boilerplate de `~/dev/projects/go-devstack`, com binário público `codex-loop`, módulo `github.com/compozy/codex-loop`, licença MIT e verificação via `make verify`.
- Implementar o projeto como greenfield em Go: CLI Cobra, runtime dos hooks, instalador/desinstalador, status, parsing de ativação, persistência de estado e geração de prompts de continuação.
- A v1 pública usa somente `codex-loop`; não deve mencionar, migrar ou manter compatibilidade com nomes/artefatos anteriores.
- Manter funcionamento correto como plugin do Codex usando a estrutura oficial atual: plugin com `.codex-plugin/plugin.json`, skill em `skills/`, lifecycle config em `hooks/hooks.json`, marketplace em `.agents/plugins/marketplace.json`, e instalação via `codex plugin marketplace add`.

## Key Changes
- Adotar o scaffold Go do `go-devstack` de forma adaptada:
  - adicionar `go.mod`, `go.sum`, `Makefile`, `magefile.go`, `.editorconfig`, `.golangci.yml`, `internal/version`, `internal/logger` e estrutura de testes;
  - trocar o `cmd/app` genérico por `cmd/codex-loop`;
  - usar Cobra conforme docs atuais do pacote `github.com/spf13/cobra` para root command, subcommands, flags, help e testes de CLI com `SetOut`, `SetErr` e `SetArgs`.
- Criar comandos públicos:
  - `codex-loop install`: prepara `~/.codex/codex-loop/`, copia/instala o binário runtime em `~/.codex/codex-loop/bin/codex-loop`, cria `loops/`, cria `config.toml` opcional se ausente, habilita `[features] codex_hooks = true` em `~/.codex/config.toml` e imprime próximos passos para instalar/ativar o plugin no Codex;
  - `codex-loop uninstall`: remove somente artefatos gerenciados de `~/.codex/codex-loop/`, preserva `~/.codex/config.toml` e preserva outros hooks/plugins;
  - `codex-loop status`: lista loops ativos por padrão, com flags `--all`, `--session-id`, `--workspace-root` e saída JSON estável;
  - `codex-loop version`: usa `internal/version`;
  - comandos internos `codex-loop hook user-prompt-submit` e `codex-loop hook stop`: leem JSON do stdin e produzem exatamente os formatos aceitos pelos hooks do Codex.
- Reestruturar o plugin:
  - criar `plugins/codex-loop` como o único bundle de plugin do projeto;
  - atualizar `.codex-plugin/plugin.json` com nome `codex-loop`, versão inicial `1.0.0`, metadata OSS, licença MIT, repository/homepage, `skills: "./skills/"` e `hooks: "./hooks/hooks.json"`;
  - substituir o skill atual por instruções que mandam instalar o binário com `go install github.com/compozy/codex-loop/cmd/codex-loop@latest`, rodar `codex-loop install`, reiniciar Codex e usar o header `[[CODEX_LOOP ...]]`;
  - criar `plugins/codex-loop/hooks/hooks.json` com `UserPromptSubmit` e `Stop` apontando para `~/.codex/codex-loop/bin/codex-loop hook ...`; o comando deve ser tolerante quando o binário ainda não existe, saindo sem bloquear até `codex-loop install` ser executado.
- Atualizar marketplace e docs:
  - mudar `.agents/plugins/marketplace.json` para apontar `./plugins/codex-loop`;
  - documentar instalação primária via `go install`, instalação/refresh de marketplace via `codex plugin marketplace add owner/repo` ou caminho local, e necessidade de reiniciar Codex após instalar/atualizar plugin;
  - adicionar `LICENSE`, README de OSS, exemplos de ativação, troubleshooting e política de dados local-first.

## Runtime Behavior
- Preservar comportamento funcional atual:
  - header precisa estar na primeira linha;
  - `name="..."` obrigatório;
  - exatamente um limitador: `min="..."` ou `rounds="..."`;
  - durações aceitam formas atuais como `30m`, `30min`, `1h 30m`, `2 hours`, `45sec`;
  - `rounds` precisa ser inteiro positivo;
  - loops são isolados por `session_id`;
  - reativar loop na mesma sessão supersede loops ativos anteriores;
  - modo tempo completa após deadline;
  - modo rounds incrementa rodadas no `Stop` e completa ao atingir o alvo;
  - rapid-stop escalation continua uma vez com prompt mais forte e depois marca `cut_short`;
  - config opcional preserva `optional_skill_name`, `optional_skill_path` e `extra_continuation_guidance`.
- Persistir estado em JSON sob `~/.codex/codex-loop/loops/` com escrita atômica e shape estável para status e debug.
- Localizar workspace root procurando `.codex/` e depois `.git`, preservando o comportamento atual.
- Não editar `~/.codex/hooks.json` como caminho principal da v1. O funcionamento como plugin deve vir do lifecycle config empacotado no plugin, conforme docs atuais do Codex. `codex-loop install` apenas prepara o runtime local e habilita `features.codex_hooks`.

## Test Plan
- Portar a suíte Python atual para Go com testes table-driven:
  - parsing de duração, rounds e header;
  - rejeição de header inválido, limitadores ausentes/duplicados e rounds inválidos;
  - criação de loop no `UserPromptSubmit`;
  - supersede de loop ativo por sessão;
  - prompt sem header ignorado;
  - continuação antes do deadline com guidance opcional;
  - conclusão por deadline;
  - conclusão por rounds;
  - rapid-stop escalation e `cut_short`.
- Adicionar testes de instalador com `t.TempDir()` e `CODEX_HOME` isolado:
  - cria runtime, loops, config opcional e cópia do binário;
  - habilita `features.codex_hooks = true` preservando outras configurações;
  - rejeita ou trata com erro claro config inválida;
  - uninstall remove apenas artefatos gerenciados e preserva config.
- Adicionar testes de CLI Cobra:
  - help/root command;
  - saída de `version`;
  - `status --all`, `--session-id`, `--workspace-root`;
  - erros em stderr e exit code não-zero para uso inválido;
  - hooks lendo stdin e escrevendo JSON compatível.
- Adicionar verificação de plugin/metadata:
  - validar JSON de `.codex-plugin/plugin.json`, `hooks/hooks.json` e `.agents/plugins/marketplace.json`;
  - garantir que `source.path` começa com `./` e aponta para `./plugins/codex-loop`;
  - garantir que manifest paths são relativos ao plugin root.
- Gate final obrigatório:
  - `go mod tidy`;
  - `make verify`;
  - teste manual com `CODEX_HOME` temporário executando `codex-loop install`, `codex-loop hook user-prompt-submit`, `codex-loop hook stop`, `codex-loop status` e `codex-loop uninstall`.

## Assumptions
- Licença: MIT.
- Nome público: `codex-loop`.
- Módulo Go: `github.com/compozy/codex-loop`.
- Distribuição primária da v1: `go install github.com/compozy/codex-loop/cmd/codex-loop@latest`.
- Plugin público da v1: `codex-loop`, sem camada de compatibilidade/migração.
- Go baseline: manter o `go 1.24` do boilerplate, compatível com o ambiente local atual (`go1.26.1`).
- Referências usadas para decisões de plugin/hooks: docs oficiais OpenAI Codex “Build plugins” e “Hooks”; referência Cobra atual em `pkg.go.dev/github.com/spf13/cobra`.
