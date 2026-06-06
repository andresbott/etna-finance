package backup

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/toolsdata"
	"golang.org/x/text/currency"
)

// TestInstrumentNotesRoundTrip verifies the instrument Notes field survives export -> import.
func TestInstrumentNotesRoundTrip(t *testing.T) {
	src := newScheduleTestStores(t, "file:notesSource?mode=memory&cache=shared")

	if _, err := src.marketdata.CreateInstrument(t.Context(), marketdata.Instrument{
		Symbol: "MSFT", Name: "Microsoft", Currency: currency.USD, Notes: "long-term hold",
	}); err != nil {
		t.Fatalf("create instrument: %v", err)
	}

	target := filepath.Join(t.TempDir(), "notes.zip")
	if err := export(t.Context(), src.accounting, src.marketdata, src.csvimport, src.filestore, src.toolsdata, src.schedules, target); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	dst := newScheduleTestStores(t, "file:notesDest?mode=memory&cache=shared")
	if err := Import(t.Context(), dst.accounting, dst.marketdata, dst.csvimport, dst.filestore, dst.toolsdata, dst.schedules, target); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	instruments, err := dst.marketdata.ListInstruments(t.Context())
	if err != nil {
		t.Fatalf("list instruments: %v", err)
	}
	if len(instruments) != 1 {
		t.Fatalf("expected 1 instrument, got %d", len(instruments))
	}
	if instruments[0].Notes != "long-term hold" {
		t.Errorf("expected notes %q, got %q", "long-term hold", instruments[0].Notes)
	}
}

// TestCaseStudyAttachmentRoundTrip verifies a case study's attachment (link + binary)
// survives export -> import.
func TestCaseStudyAttachmentRoundTrip(t *testing.T) {
	src := newScheduleTestStores(t, "file:csAttSource?mode=memory&cache=shared")

	cs, err := src.toolsdata.Create(t.Context(), toolsdata.CaseStudy{
		ToolType: "buy_vs_rent", Name: "with-doc", Description: "has attachment",
	})
	if err != nil {
		t.Fatalf("create case study: %v", err)
	}
	pdfContent := append([]byte("%PDF-1.4"), bytes.Repeat([]byte{0x00}, 20)...)
	attID, err := src.filestore.SaveRaw(t.Context(), getDate("2024-01-01"), pdfContent, "study.pdf", "application/pdf")
	if err != nil {
		t.Fatalf("save attachment: %v", err)
	}
	if err := src.toolsdata.SetAttachmentID(t.Context(), cs.ToolType, cs.ID, &attID); err != nil {
		t.Fatalf("link attachment: %v", err)
	}

	target := filepath.Join(t.TempDir(), "csatt.zip")
	if err := export(t.Context(), src.accounting, src.marketdata, src.csvimport, src.filestore, src.toolsdata, src.schedules, target); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	dst := newScheduleTestStores(t, "file:csAttDest?mode=memory&cache=shared")
	if err := Import(t.Context(), dst.accounting, dst.marketdata, dst.csvimport, dst.filestore, dst.toolsdata, dst.schedules, target); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	studies, err := dst.toolsdata.ListAll(t.Context())
	if err != nil {
		t.Fatalf("list case studies: %v", err)
	}
	if len(studies) != 1 {
		t.Fatalf("expected 1 case study, got %d", len(studies))
	}
	if studies[0].AttachmentID == nil {
		t.Fatalf("expected case study to have an attachment after restore, got nil")
	}

	att, err := dst.filestore.Get(t.Context(), *studies[0].AttachmentID)
	if err != nil {
		t.Fatalf("get restored attachment: %v", err)
	}
	if att.OriginalName != "study.pdf" {
		t.Errorf("expected original name %q, got %q", "study.pdf", att.OriginalName)
	}
}
