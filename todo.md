# Audit: EÜR & Inventar — Fehlende Funktionen

## Context

Audit der bestehenden EÜR- und Inventar-Funktionen des Invoice Managers. Ziel: Identifizierung fehlender Funktionen, die für eine vollständige Buchhaltung/Lagerverwaltung eines deutschen Kleinunternehmens wichtig wären.

---

## Bestehende Funktionen (Ist-Zustand)

### EÜR

- Einnahmen aus bezahlten Rechnungen (Netto/Brutto/USt)
- Ausgaben erfassen mit Beleg-Upload (Base64 in DB)
- Ausgaben bearbeiten & löschen
- Ausgabenkategorien & Kategorien-Auswertung
- Gewinn-Berechnung (Einnahmen - Ausgaben, Netto-Basis)
- USt-Zahllast Berechnung
- PDF-Export der Übersicht (inkl. USt-Details)
- Verknüpfung Ausgabe → Lagerbewegung (Einkauf)

### Inventar

- Produkte CRUD (Name, Preis, Beschreibung, Einheit, Bestand)
- Lagerbewegungen (INVOICE, PURCHASE, MANUAL_ADD, MANUAL_REMOVE, CANCELLATION)
- Bestand hinzufügen/entfernen mit optionaler Ausgabenbuchung
- Bewegungshistorie pro Produkt
- Storno-Logik mit automatischer Lager-Rückbuchung

---

## Fehlende Funktionen (nach Priorität)

### Priorität 1 — Kritisch für korrekte Buchhaltung

| #   | Funktion                           | Status      | Beschreibung                                                                                                                               |
| --- | ---------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **Jahres-/Zeitraumfilter für EÜR** | **DONE**    | Filter nach Jahr implementiert.                                                                                                            |
| 2   | **Ausgaben bearbeiten**            | **DONE**    | Edit-Formular mit Beleg-Update implementiert.                                                                                              |
| 3   | **Storno mit Lagerkorrektur**      | **DONE**    | `CancelInvoice()` erstellt jetzt CANCELLATION-Lagerbewegungen.                                                                             |
| 4   | **USt-Berechnung in EÜR**          | **DONE**    | Netto/USt/Brutto Aufschlüsselung und Zahllast-Berechnung für EÜR und PDF.                                                                  |

### Priorität 2 — Wichtig für vollständige Funktionalität

| #   | Funktion                             | Status      | Beschreibung                                                                                                                  |
| --- | ------------------------------------ | ----------- | ----------------------------------------------------------------------------------------------------------------------------- |
| 5   | **Kategorien-Auswertung**            | **DONE**    | Ausgaben nach Kategorie gruppiert anzeigen (Summe pro Kategorie, Prozentanteil).                                              |
| 6   | **Mindestbestand / Bestandswarnung** | **DONE**    | Feld min_stock hinzugefügt, Warn-Icons in Liste implementiert.                                                                |
| 7   | **Inventar-PDF / Inventurliste**     | **DONE**    | PDF-Export der aktuellen Bestände inkl. Gesamtwert des Lagers.                                                                |
| 8   | **Wiederkehrende Ausgaben**          | **DONE**    | Automatische Buchung von Fixkosten (Miete etc.) beim Aufruf der EÜR.                                                          |
| 9   | **CSV/DATEV-Export**                 | **DONE**    | CSV-Export für Einnahmen/Ausgaben (Semicolon-separiert für Excel).                                                            |

### Priorität 3 — Nice-to-have

| #   | Funktion                             | Status      | Beschreibung                                                                               |
| --- | ------------------------------------ | ----------- | ------------------------------------------------------------------------------------------ |
| 10  | **Zahlungserinnerung / Mahnung**     | OFFEN       | Automatische Erinnerung für offene Rechnungen nach X Tagen. PDF-Mahnung generieren.        |
| 11  | **Angebote / Kostenvoranschläge**    | OFFEN       | Angebote erstellen, die später in Rechnungen umgewandelt werden können.                    |
| 12  | **Gutschriften**                     | OFFEN       | Teilweise oder vollständige Gutschrift zu einer Rechnung erstellen.                        |
| 13  | **Abschreibungen (AfA)**             | OFFEN       | Anlagegüter mit Abschreibungsdauer erfassen, jährliche AfA automatisch als Ausgabe buchen. |
| 14  | **Inventar-Bewertung**               | **DONE**    | Berechnet im Inventar-PDF (Menge × Einkaufspreis).                                         |
| 15  | **Bewegungshistorie filtern/suchen** | OFFEN       | Lagerbewegungen nach Datum, Typ oder Produkt filtern.                                      |

---

## Bekannte Bugs / Inkonsistenzen (Erledigt)

1. **Storno ohne Lagerkorrektur** — Behoben in `models/invoice.go`
2. **EÜR ohne Datumsfilter** — Behoben in `models/euer.go` und Handlern
3. **Doppelte Ausgabenliste** — Behoben in `views/euer.templ`

---

## Nächste Schritte

1. Angebote / Kostenvoranschläge (#11)
2. Gutschriften (#12)
3. Zahlungserinnerung / Mahnung (#10)
4. Abschreibungen (AfA) (#13)
