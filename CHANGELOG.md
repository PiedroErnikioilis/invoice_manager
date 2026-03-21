# Changelog

Alle wichtigen Änderungen an diesem Projekt werden in dieser Datei festgehalten.

## [v0.4.3] - 2026-03-21

### Behoben

- **PDF-Layout & Multi-Page-Handling**:
  - Implementierung einer robusten Tabellen-Container-Strategie (`thead`/`tfoot`) zur konsistenten Reservierung von Header- und Footer-Bereichen auf allen Seiten.
  - Fix: Der Footer wird nun nicht mehr vom Inhalt überschrieben und bleibt auf jeder Seite am unteren Rand fixiert.
  - Fix: Fehlendes Padding am oberen Rand auf Folgeseiten (Seite 2+) korrigiert.
  - Korrektur der DIN 5008 Abstände auf der ersten Seite durch Einsatz eines dedizierten Spacers.
  - Deaktivierung der Standard-Browser-Margins im Go-Service für volle CSS-Layout-Kontrolle.

## [v0.4.2] - 2026-03-21

### Hinzugefügt

- **Notiz-Funktionen für Dokumente**:
  - **Interne Notizen**: Neues Feld für Rechnungen, Angebote und Gutschriften, das nur im System sichtbar ist (ideal für interne Vermerke).
  - **Dokument-Kommentare**: Frei gestaltbare Texte, die direkt auf dem PDF-Dokument (vor dem Fußbereich) angedruckt werden.
- **Erweiterte Demo-Daten**:
  - Alle Demo-Dokumente enthalten nun Beispiel-Notizen und Kommentare.
  - Neue **20-Positionen-Testrechnung** (`RE-BIG-2026`) zum Testen von mehrseitigen Layouts und langen Artikellisten.

### Geändert

- **PDF-Templates**: Unterstützung für den Andruck von Dokument-Kommentaren in Rechnungen, Angeboten und Gutschriften.
- **Benutzeroberfläche**: Neue Textbereiche in allen Erstellungs- und Bearbeitungsformularen für Notizen.

## [v0.4.1] - 2026-03-21

### Hinzugefügt

- **Maximale Transparenz durch Deep-Logging**:
  - Detaillierte Einstiegs-Logs für alle HTTP-Handler inklusive HTTP-Methoden und IDs zur besseren Nachverfolgbarkeit von Benutzerinteraktionen.
  - Schritt-für-Schritt Protokollierung komplexer Datenbank-Operationen (z.B. automatisierte Lagerbestands-Korrekturen bei Rechnungs-Updates).
  - Erweiterte Status-Abfragen bei Lagerbewegungen: Der neue Lagerbestand wird nach jeder Buchung direkt im Log validiert und angezeigt.
  - Lückenlose Aufzeichnung der Datenbank-Initialisierung und aller Migrationsschritte.
  - Detaillierte Protokollierung des Demo-Daten-Seedings zur schnelleren Fehlerdiagnose bei Erstinstallationen.

### Geändert

- **Harmonisierung der Dezimaltrennzeichen**:
  - Alle Beträge, Preise und Steuersätze werden nun konsistent mit Komma als Dezimaltrenner angezeigt und in Eingabefeldern vorausgefüllt.
  - Neue Hilfsfunktionen `FormatDecimal` und `FormatDecimalSimple` für eine saubere Lokalisierung.
  - Die Eingabe akzeptiert weiterhin flexibel sowohl Komma als auch Punkt (wird beim Speichern automatisch normalisiert).

### Behoben

- **Stabilität & Build**:
  - Korrektur von internen Namenskonflikten in den Datenbank-Modellen (`s.DB` vs `s.db`).
  - Fehlerhafte Pointer-Übergaben bei Kunden-Operationen in Handlern und Demo-Seeds behoben.
  - Bereinigung ungenutzter Imports und Stabilisierung der Kompilierung.

## [v0.4.0] - 2026-03-15

### Hinzugefügt

- **Gutschriften-Detailansicht**: Detaillierte Ansicht für Gutschriften inklusive PDF-Export.
- **Angebots-Detailansicht**: Detaillierte Ansicht für Angebote inklusive PDF-Export und Umwandlungs-Funktion.
- **Smart PDF-Generierung**: Bereits erstellte PDFs werden bei finalem Status (z.B. Bezahlt) nicht mehr automatisch neu generiert, was die Ladezeiten erheblich verkürzt.
- **Manueller PDF-Refresh**: Neue Schaltfläche "PDF neu erzeugen" in der Detailansicht, um bei Bedarf eine Aktualisierung zu erzwingen.
- **Umfangreiches strukturiertes Logging**:
  - Vollständige Umstellung auf `log/slog` im gesamten Projekt (Handler, Modelle, Services).
  - Alle Log-Ausgaben werden nun permanent in die Datei `app.log` geschrieben (zusätzlich zur Konsole).
  - **Farbige Konsolenausgabe**: Die Log-Ausgaben im Terminal sind nun zur besseren Übersicht farblich hervorgehoben (Debug=grau, Info=cyan, Error=rot).
  - **Debug-Modus standardmäßig aktiv**: Ausführliche Informationen werden ab sofort ohne zusätzliche Konfiguration erfasst.
  - Neue HTTP-Middleware für detaillierte Request-Logs inklusive Fehlermeldungen und Performance-Daten.
  - Detaillierte Protokollierung von PDF-Generierungsprozessen und Datenbanktransaktionen.
  - Steuerbar über Umgebungsvariablen: `DEBUG=1` für Debug-Level und `JSON_LOG=1` für Maschinen-lesbares Format.
- **Konfigurierbare Dateinamen**: PDF-Exporte für Rechnungen, Angebote, Gutschriften und Inventarlisten können nun über eigene Schemata benannt werden (z.B. `{ID}.pdf` oder `Rechnung_{YYYY}_{ID}.pdf`).
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

- **Build-Fehler korrigiert**: Syntaxfehler in Handlern und fehlende Imports behoben.
- **PDF-Verbesserungen**:
  - Korrekte Seitenzahlen („Seite X von Y") für alle Export-Typen.
  - Fehler behoben, bei dem Texte am Ende langer Rechnungen abgeschnitten wurden.
- **Stabilität**: WAL/SHM-Dateien werden nun korrekt bei Backups berücksichtigt.

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
