package qpdf

import (
	"errors"
	"strings"
	"testing"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

func TestValidateFacturX(t *testing.T) {
	valid := gotenberg.FacturX{
		ConformanceLevel: gotenberg.FacturXConformanceEN16931,
		DocumentType:     gotenberg.FacturXDocumentTypeInvoice,
		DocumentFileName: "factur-x.xml",
		Version:          "1.0",
	}

	tests := []struct {
		name      string
		mutate    func(f *gotenberg.FacturX)
		wantError bool
	}{
		{name: "valid", mutate: func(*gotenberg.FacturX) {}},
		{
			name:   "valid BASIC WL conformance",
			mutate: func(f *gotenberg.FacturX) { f.ConformanceLevel = gotenberg.FacturXConformanceBasicWL },
		},
		{
			name:   "valid ORDER document type",
			mutate: func(f *gotenberg.FacturX) { f.DocumentType = gotenberg.FacturXDocumentTypeOrder },
		},
		{
			name:      "unsupported conformance level",
			mutate:    func(f *gotenberg.FacturX) { f.ConformanceLevel = "FOO" },
			wantError: true,
		},
		{
			name:      "empty conformance level",
			mutate:    func(f *gotenberg.FacturX) { f.ConformanceLevel = "" },
			wantError: true,
		},
		{
			name:      "unsupported document type",
			mutate:    func(f *gotenberg.FacturX) { f.DocumentType = "RECEIPT" },
			wantError: true,
		},
		{
			name:      "empty document file name",
			mutate:    func(f *gotenberg.FacturX) { f.DocumentFileName = "" },
			wantError: true,
		},
		{
			name:      "empty version",
			mutate:    func(f *gotenberg.FacturX) { f.Version = "" },
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facturX := valid
			tt.mutate(&facturX)

			err := validateFacturX(facturX)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				if !errors.Is(err, gotenberg.ErrPdfFacturXValueNotSupported) {
					t.Errorf("expected ErrPdfFacturXValueNotSupported, got %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestFindMetadataStream(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantXMP   string
		wantError bool
	}{
		{
			name: "metadata stream found",
			// "PFhtcD4=" is base64 for "<xmp>".
			input:   `{"qpdf":[{},{"obj:1 0 R":{"value":{"/Type":"/Catalog","/Metadata":"4 0 R"}},"obj:4 0 R":{"stream":{"dict":{"/Type":"/Metadata","/Subtype":"/XML"},"data":"PHhtcD4="}}}]}`,
			wantKey: "obj:4 0 R",
			wantXMP: "<xmp>",
		},
		{
			name:      "catalog without metadata reference",
			input:     `{"qpdf":[{},{"obj:1 0 R":{"value":{"/Type":"/Catalog"}}}]}`,
			wantError: true,
		},
		{
			name:      "metadata object missing",
			input:     `{"qpdf":[{},{"obj:1 0 R":{"value":{"/Type":"/Catalog","/Metadata":"4 0 R"}}}]}`,
			wantError: true,
		},
		{
			name:      "metadata object is not a stream",
			input:     `{"qpdf":[{},{"obj:1 0 R":{"value":{"/Type":"/Catalog","/Metadata":"4 0 R"}},"obj:4 0 R":{"value":{"/Type":"/Metadata"}}}]}`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects, err := parsePdfObjects([]byte(tt.input))
			if err != nil {
				t.Fatalf("parse objects: %v", err)
			}

			key, dict, xmp, err := findMetadataStream(objects)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
			if xmp != tt.wantXMP {
				t.Errorf("xmp = %q, want %q", xmp, tt.wantXMP)
			}
			if dict == nil {
				t.Error("expected a non-nil dict")
			}
		})
	}
}

func TestInjectFacturXIntoXMP(t *testing.T) {
	facturX := gotenberg.FacturX{
		ConformanceLevel: gotenberg.FacturXConformanceEN16931,
		DocumentType:     gotenberg.FacturXDocumentTypeInvoice,
		DocumentFileName: "factur-x.xml",
		Version:          "1.0",
	}

	// The packet a LibreOffice PDF/A-3b export produces: no pdfaExtension bag.
	libreOfficeXMP := `<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about="" xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/">
   <pdfaid:part>3</pdfaid:part>
   <pdfaid:conformance>B</pdfaid:conformance>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`

	t.Run("creates extension schema when bag absent", func(t *testing.T) {
		got, changed := injectFacturXIntoXMP(libreOfficeXMP, facturX)
		if !changed {
			t.Fatal("expected changed = true")
		}
		assertContains(t, got, facturXNamespaceURI)
		assertContains(t, got, "<fx:ConformanceLevel>EN 16931</fx:ConformanceLevel>")
		assertContains(t, got, "<fx:DocumentType>INVOICE</fx:DocumentType>")
		assertContains(t, got, "<fx:DocumentFileName>factur-x.xml</fx:DocumentFileName>")
		assertContains(t, got, "<fx:Version>1.0</fx:Version>")
		assertContains(t, got, "pdfaExtension:schemas")
		assertContains(t, got, "http://www.aiim.org/pdfa/ns/extension/")
		// The fx blocks must land inside the RDF container.
		if strings.Index(got, facturXNamespaceURI) > strings.LastIndex(got, "</rdf:RDF>") {
			t.Error("fx content injected outside the rdf:RDF container")
		}
	})

	t.Run("idempotent when fx already present", func(t *testing.T) {
		once, _ := injectFacturXIntoXMP(libreOfficeXMP, facturX)
		twice, changed := injectFacturXIntoXMP(once, facturX)
		if changed {
			t.Error("expected changed = false on a packet that already declares fx")
		}
		if twice != once {
			t.Error("expected the packet to be left untouched")
		}
		if strings.Count(twice, "<pdfaExtension:schemas>") != 1 {
			t.Error("expected exactly one pdfaExtension:schemas declaration")
		}
	})

	t.Run("appends entry to an existing bag", func(t *testing.T) {
		withBag := `<x:xmpmeta xmlns:x="adobe:ns:meta/">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about="" xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/" xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#" xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#">
   <pdfaExtension:schemas>
    <rdf:Bag>
     <rdf:li rdf:parseType="Resource"><pdfaSchema:prefix>other</pdfaSchema:prefix></rdf:li>
    </rdf:Bag>
   </pdfaExtension:schemas>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>`
		got, changed := injectFacturXIntoXMP(withBag, facturX)
		if !changed {
			t.Fatal("expected changed = true")
		}
		assertContains(t, got, facturXNamespaceURI)
		if strings.Count(got, "<pdfaExtension:schemas>") != 1 {
			t.Error("expected the existing pdfaExtension:schemas bag to be reused, not duplicated")
		}
		if strings.Count(got, "<pdfaSchema:prefix>") != 2 {
			t.Error("expected both the existing and the fx schema entries")
		}
	})

	t.Run("no rdf:RDF anchor leaves packet unchanged", func(t *testing.T) {
		got, changed := injectFacturXIntoXMP("not an xmp packet", facturX)
		if changed {
			t.Error("expected changed = false")
		}
		if got != "not an xmp packet" {
			t.Error("expected the packet to be left untouched")
		}
	})

	t.Run("escapes runtime values", func(t *testing.T) {
		escaped := facturX
		escaped.DocumentFileName = "a&b<c>.xml"
		got, _ := injectFacturXIntoXMP(libreOfficeXMP, escaped)
		assertContains(t, got, "a&amp;b&lt;c&gt;.xml")
		if strings.Contains(got, "<fx:DocumentFileName>a&b<c>.xml") {
			t.Error("expected the document file name to be XML-escaped")
		}
	})
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q", needle)
	}
}
