# Changelog

Alle wichtigen Änderungen an diesem Projekt werden in dieser Datei festgehalten.

## [v0.3.3] - 2026-03-15

### Hinzugefügt
- **Rechnungssuche**: Freitext-Suche über Rechnungsnummer, Empfängername und Kunden-ID.
- **Status-Filter**: Dropdown-Filter nach Rechnungsstatus (Entwurf/Offen/Bezahlt/Storniert).
- **Sortierbare Spalten**: Alle Spalten der Rechnungsliste (Nr., Datum, Empfänger, Positionen, Status) sind per Klick sortierbar mit Richtungsanzeige.
- **Positionsanzahl**: Neue Spalte „Pos." zeigt die Anzahl der Rechnungspositionen.
- **Kunden-ID in Liste**: Kundennummer wird unter dem Empfängernamen angezeigt (falls zugeordnet).
- **Ergebniszähler**: Anzeige der Gesamtanzahl gefundener Rechnungen unter der Tabelle.

---

## [v0.3.2] - 2026-03-15

### Behoben
- **PDF Seitenzahlen**: Alle PDFs (Rechnungen, EÜR, Inventar) zeigen jetzt korrekte Seitenzahlen „Seite X von Y" an.
    - EÜR-PDF zeigte bisher immer nur „Seite 1", auch bei mehrseitigen Dokumenten.
    - Rechnungs-PDF hatte keine Seitenzahl.
    - Nutzt Chrome's eingebaute `pageNumber`/`totalPages` für zuverlässige Zählung.
- **PDF Textabschnitt**: Zahlungshinweis auf Rechnungen wurde bei vielen Positionen abgeschnitten (`overflow: hidden` entfernt).
- **Zahlungshinweis**: Vollständiger Text „Bitte überweisen Sie den Gesamtbetrag innerhalb von 14 Tagen auf das unten genannte Konto unter Angabe der Rechnungsnummer."

---

## [v0.3.1] - 2026-03-15

### Behoben
- **Komma als Dezimaltrennzeichen**: Beträge, Preise und Steuersätze können nun mit Komma eingegeben werden (z.B. `1.234,56` oder `123,45`).
    - Betrifft alle Formulare: Ausgaben, Rechnungen, Angebote, Gutschriften, Produkte, wiederkehrende Ausgaben.
    - Deutsches Format (`1.234,56`), englisches Format (`1,234.56`) und einfache Komma-Notation (`123,45`) werden automatisch erkannt.
    - HTML-Inputs von `type="number"` auf `type="text" inputmode="decimal"` umgestellt, damit Browser Komma-Eingabe nicht blockieren.

---

## [v0.3.0] - 2026-03-15

### Hinzugefügt
- **Demo-Modus**: Automatische Erstellung realistischer Beispieldaten für Tests und Performance-Benchmarks.
    - 20 Kunden, 15 Produkte, 50 Rechnungen, 15 Angebote, 5 Gutschriften, 32 Ausgaben, 6 wiederkehrende Ausgaben.
    - Aktivierung via `--demo` Flag beim Start (z.B. `go run . --demo`).
    - **Sicherheit**: Daten werden nur bei einer neu erstellten Datenbank eingefügt – bestehende Daten werden nie überschrieben.
    - Deterministischer Seed für reproduzierbare Testdaten.

---

## [v0.2.0] - 2026-03-15

### Hinzugefügt
- **Datenbank-Backup-System**:
    - Manuelles Erstellen, Herunterladen, Wiederherstellen und Löschen von Backups über `/backups`.
    - **Pre-Migration-Backup**: Automatisches Backup vor Schema-Änderungen beim Serverstart.
    - **Jahresabschluss-Backup**: Automatisches `jahresabschluss_YYYY.db` beim ersten Start im neuen Jahr.
    - **Mindestabstand**: Konfigurierbarer Mindestzeitraum (Standard: 24h) verhindert zu häufige Backups bei Neustarts.
    - **Automatische Rotation**: Älteste Backups werden bei Überschreitung der Maximalanzahl gelöscht.
    - Sicherheitsbackup vor jeder Wiederherstellung (`vor_wiederherstellung_*.db`).
    - WAL/SHM-Dateien werden bei Pre-Migration- und Jahresabschluss-Backups mitkopiert.
- **Backup-Einstellungen** in `/settings`: Backup-Verzeichnis, max. Anzahl, Mindestabstand, Auto-Backup ein/aus.

---

## [v0.1.1] - 2026-03-15

### Behoben
- **Kundennummer auf Rechnung**: Kundennummer wurde in der HTML-Ansicht immer angezeigt (basierend auf Rechnungs-ID), auch wenn kein Kunde zugeordnet war. Jetzt wird sie nur noch bei zugeordnetem Kunden angezeigt und verwendet die echte Kunden-ID (konsistent mit der PDF-Ansicht).

---

## [v0.1.0] - 2026-03-14

### Hinzugefügt
- **Angebotsverwaltung**: Erstellen und Bearbeiten von Angeboten/Kostenvoranschlägen inklusive Umwandlung in Rechnungen.
- **Gutschriften**: Erstellen von Gutschriften aus Rechnungen mit automatischer (negativer) Berücksichtigung in der EÜR.
- **Lagerverwaltung Pro**: 
    - Mindestbestand pro Produkt mit visueller Warnung in der Liste.
    - Inventurliste als PDF-Export inklusive automatischer Lagerbewertung.
    - Stornierung von Rechnungen führt nun zur automatischen Lager-Rückbuchung.
- **EÜR Erweiterungen**:
    - Vollständige Umsatzsteuer-Berechnung (Netto/USt/Brutto) für Einnahmen und Ausgaben.
    - Berechnung der USt-Zahllast / Vorsteuer-Überhang.
    - Kategorien-Auswertung der Ausgaben mit grafischer Darstellung (Prozentanteile).
    - Jahresfilter für alle EÜR-Statistiken.
    - CSV-Export der Buchungsdaten (für Excel/DATEV).
- **Wiederkehrende Ausgaben**: Automatisches Buchen von Fixkosten (Miete, Abos) basierend auf Intervallen (monatlich, quartalsweise, jährlich).
- **Ausgaben bearbeiten**: Bestehende Ausgaben können nun nachträglich korrigiert und Belege ersetzt werden.

### Geändert
- **Datenbank-Schema**: Normalisierung der Kategorien und Einführung strenger Referenzintegrität (Foreign Keys).
- **UI/UX**: Optimierte Navigation und verbesserte Tabellenansichten für mehr Übersichtlichkeit.
- **PDF-Design**: EÜR-PDF enthält nun detaillierte Steueraufschlüsselungen.

### Behoben
- Doppelte Anzeige der Ausgabenliste im EÜR-Dashboard.
- Falsche Verlinkung bei Beleg-Anzeigen.
- Inkonsistente Lagerbestände nach Rechnungsstorno.

---
Letzte Version vor diesen Änderungen: v0.0.4
