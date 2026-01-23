# DIN Invoice App

Eine einfache App zum Erstellen von Rechnungen nach DIN 5008 (ähnlich).

## Voraussetzungen

- Go installiert
- SQLite (wird als Datei erstellt)

## Starten

1. Installiere templ:
   ```bash
   go install github.com/a-h/templ/cmd/templ@latest
   ```
2. Generiere Templates und starte die App:
   ```bash
   ~/go/bin/templ generate
   go run main.go
   ```

Die App ist unter [http://localhost:3000](http://localhost:3000) erreichbar.

## Funktionen

- Rechnungen erstellen
- Rechnungen auflisten
- Rechnung als DIN-konforme Ansicht anzeigen (Drucken als PDF möglich)
