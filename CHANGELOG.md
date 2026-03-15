# Changelog

Alle wichtigen Änderungen an diesem Projekt werden in dieser Datei festgehalten.

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
