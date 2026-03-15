# Changelog

Alle wichtigen Änderungen an diesem Projekt werden in dieser Datei festgehalten.

## [v0.4.0] - 2026-03-15

### Hinzugefügt
- **Gutschriften-Detailansicht**: Detaillierte Ansicht für Gutschriften inklusive PDF-Export.
- **Umfangreiches strukturiertes Logging**:    - Vollständige Umstellung auf `log/slog` im gesamten Projekt (Handler, Modelle, Services).
    - Alle Log-Ausgaben werden nun permanent in die Datei `app.log` geschrieben (zusätzlich zur Konsole).
    - **Farbige Konsolenausgabe**: Die Log-Ausgaben im Terminal sind nun zur besseren Übersicht farblich hervorgehoben (Debug=grau, Info=cyan, Error=rot).
    - **Debug-Modus standardmäßig aktiv**: Ausführliche Informationen werden ab sofort ohne zusätzliche Konfiguration erfasst.
    - Neue HTTP-Middleware für detaillierte Request-Logs inklusive Fehlermeldungen und Performance-Daten.
    - Detaillierte Protokollierung von PDF-Generierungsprozessen und Datenbanktransaktionen.
    - Steuerbar über Umgebungsvariablen: `DEBUG=1` für Debug-Level und `JSON_LOG=1` für Maschinen-lesbares Format.
- **Konfigurierbare Nummernschemata**:
    - Rechnungen, Angebote und Gutschriften unterstützen nun benutzerdefinierte Schemata (z.B. `RE-{YYYY}-{N:4}`).
    - **Echte Kundennummern**: Kunden haben nun eine eigene `customer_number` in der Datenbank.
    - Das Nummernschema für Kunden ist ebenfalls konfigurierbar (z.B. `KD-{N:4}`).
    - **EÜR Dateinamen**: Das Namensschema für EÜR-Exporte (PDF/CSV) ist einstellbar (z.B. `EÜR-{YYYY}`).
    - Live-Vorschau der konfigurierten Schemata in den Einstellungen.
- **Rechnungssuche & Filter**:
    - Freitext-Suche über Rechnungsnummer, Empfängername, Kundennummer und Kunden-ID.
    - Status-Filter (Entwurf/Offen/Bezahlt/Storniert).
    - Sortierbare Spalten in der Rechnungsliste mit Richtungsanzeige.
- **Datenbank-Backup-System**:
    - Manuelles Erstellen, Herunterladen, Wiederherstellen und Löschen von Backups.
    - Automatisches **Pre-Migration-Backup** vor Schema-Änderungen.
    - Automatisches **Jahresabschluss-Backup** beim ersten Start im neuen Jahr.
    - Konfigurierbare Rotation und Mindestintervalle für Backups.
- **Demo-Modus**: Automatische Erstellung realistischer Beispieldaten via `--demo` Flag für neue Datenbanken.

### Geändert
- **Zahlungseingaben**: Beträge, Preise und Steuersätze können nun flexibel mit Komma oder Punkt eingegeben werden (z.B. `1.234,56`).
- **Kunden-Anzeige**: Kundennummern werden nun konsistent in allen Listen, Formularen und Dokumenten (PDF/HTML) angezeigt.

### Behoben
- **PDF-Verbesserungen**: 
    - Korrekte Seitenzahlen („Seite X von Y") für alle Export-Typen.
    - Fehler behoben, bei dem Texte am Ende langer Rechnungen abgeschnitten wurden.
- **Stabilitat**: WAL/SHM-Dateien werden nun korrekt bei Backups berücksichtigt.

---

## [v0.1.0] - 2026-03-14

### Hinzugefügt
- **Angebotsverwaltung**: Erstellen und Bearbeiten von Angeboten inklusive Umwandlung in Rechnungen.
- **Gutschriften**: Erstellen von Gutschriften aus Rechnungen mit automatischer EÜR-Berücksichtigung.
- **Lagerverwaltung Pro**: 
    - Mindestbestand pro Produkt mit visueller Warnung.
    - Inventurliste als PDF-Export inklusive Lagerbewertung.
    - Automatische Lager-Rückbuchung bei Stornierung.
- **EÜR Erweiterungen**:
    - Umsatzsteuer-Berechnung und USt-Zahllast Ermittlung.
    - Kategorien-Auswertung mit grafischer Darstellung.
    - Jahresfilter und CSV-Export.
- **Wiederkehrende Ausgaben**: Automatische Buchung von Fixkosten (Miete, Abos).

### Geändert
- **Datenbank-Schema**: Normalisierung der Kategorien und Einführung von Foreign Keys.
- **UI/UX**: Optimierte Navigation und verbesserte Tabellenansichten.

---
Letzte Version vor diesen Änderungen: v0.0.4
