package pdfcpu

import "testing"

const sampleImagesList = `pages: all

/tmp/multi.pdf:
4 images available (8.9 MB)
Page │ Obj# │ Id  │ Type  SoftMask ImgMask │ Width │ Height │ ColorSpace Comp bpc Interp │   Size │ Filters
━━━━━┿━━━━━━┿━━━━━┿━━━━━━━━━━━━━━━━━━━━━━━━┿━━━━━━━┿━━━━━━━━┿━━━━━━━━━━━━━━━━━━━━━━━━━━━━┿━━━━━━━━┿━━━━━━━━━━━━
   1 │    6 │  X6 │ image                  │  2400 │   1800 │   ICCBased    3   8        │ 5.5 MB │ FlateDecode
     │    8 │  X8 │ image    *             │  1400 │   1000 │   ICCBased    3   8        │  63 KB │ FlateDecode
     │    9 │  X9 │ image                  │  2400 │   1800 │   ICCBased    3   8        │ 194 KB │ DCTDecode
     │   10 │ X10 │ image                  │   120 │     90 │ DeviceCMYK    4   8        │  14 KB │ FlateDecode
`

func TestParseImagesList(t *testing.T) {
	images := parseImagesList(sampleImagesList)

	if len(images) != 4 {
		t.Fatalf("expected 4 images, got %d", len(images))
	}

	for _, tc := range []struct {
		index  int
		obj    int
		id     string
		masked bool
		comp   int
		filter string
	}{
		{0, 6, "X6", false, 3, "FlateDecode"},
		{1, 8, "X8", true, 3, "FlateDecode"},
		{2, 9, "X9", false, 3, "DCTDecode"},
		{3, 10, "X10", false, 4, "FlateDecode"},
	} {
		img := images[tc.index]
		if img.obj != tc.obj || img.id != tc.id || img.masked != tc.masked || img.comp != tc.comp || img.filter != tc.filter {
			t.Errorf("image %d = %+v, want obj=%d id=%s masked=%v comp=%d filter=%s",
				tc.index, img, tc.obj, tc.id, tc.masked, tc.comp, tc.filter)
		}
	}
}

func TestParseHumanSize(t *testing.T) {
	for _, tc := range []struct {
		cell string
		want int64
	}{
		{"5.5 MB", int64(5.5 * (1 << 20))},
		{"194 KB", 194 << 10},
		{" 14 KB ", 14 << 10},
		{"512 B", 512},
		{"2 GB", 2 << 30},
		{"", 0},
		{"garbage", 0},
	} {
		if got := parseHumanSize(tc.cell); got != tc.want {
			t.Errorf("parseHumanSize(%q) = %d, want %d", tc.cell, got, tc.want)
		}
	}
}

func TestOptimizableImage(t *testing.T) {
	base := pdfcpuImage{obj: 1, id: "X1", masked: false, comp: 3, bytes: 1 << 20, filter: "FlateDecode"}

	for _, tc := range []struct {
		scenario string
		mutate   func(pdfcpuImage) pdfcpuImage
		want     bool
	}{
		{"lossless RGB above threshold", func(i pdfcpuImage) pdfcpuImage { return i }, true},
		{"already compressed", func(i pdfcpuImage) pdfcpuImage { i.filter = "DCTDecode"; return i }, false},
		{"CMYK", func(i pdfcpuImage) pdfcpuImage { i.comp = 4; return i }, false},
		{"masked", func(i pdfcpuImage) pdfcpuImage { i.masked = true; return i }, false},
		{"below threshold", func(i pdfcpuImage) pdfcpuImage { i.bytes = minOptimizeImageSize - 1; return i }, false},
		{"grayscale above threshold", func(i pdfcpuImage) pdfcpuImage { i.comp = 1; return i }, true},
	} {
		if got := optimizableImage(tc.mutate(base)); got != tc.want {
			t.Errorf("%s: optimizableImage = %v, want %v", tc.scenario, got, tc.want)
		}
	}
}
