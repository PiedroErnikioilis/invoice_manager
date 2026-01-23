Danke für dein Interesse, zum Projekt beizutragen! 🎉

Kurz (Do / Don't)
- Do: Eröffne Issues für größere Änderungen, erstelle kleine, fokussierte PRs
- Don't: große ungeprüfte Änderungen ohne RFC

Lokale Entwicklung
- Tests: `go test ./...`
- Lint: `golangci-lint run`
- Format: `gofmt -w .` (oder `gofumpt`)

Branch- & Commit-Name
- Branch: `feat/<kurz-beschreibung>` | `fix/<ticket>` | `chore/<aufgabe>`
- Commit: Conventional Commits empfohlen (feat/fix/chore)

PR-Checkliste
- Tests vorhanden / angepasst
- Linter grünes Licht
- CHANGELOG-Eintrag, falls sichtbare Änderung

Code‑Style
- Kleine Funktionen, klare Responsibility
- Exporte nur wenn nötig

Fragen
- Für größere Änderungen bitte zuerst ein Issue oder RFC öffnen.