package pdfcpu

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Register the PNG decoder: pdfcpu extracts FlateDecode images as PNG.
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/gotenberg/gotenberg/v8/pkg/gotenberg"
)

// minOptimizeImageSize is the smallest encoded image worth re-encoding. Smaller
// images (thumbnails, icons, and line art that FlateDecode already keeps tiny)
// are left untouched: a JPEG pass would add artifacts for little or no gain.
const minOptimizeImageSize = 30 << 10 // 30 KiB

// pdfcpuListRowID matches the image Id (e.g. "X6") in a `pdfcpu images extract`
// filename such as "input_1_X6.png".
var pdfcpuListRowID = regexp.MustCompile(`_(X\d+)\.`)

// pdfcpuImage is one raster image XObject as reported by `pdfcpu images list`.
type pdfcpuImage struct {
	obj    int
	id     string
	masked bool
	comp   int
	bytes  int64
	filter string
}

// OptimizeImages re-encodes the raster images of inputPath to JPEG in place,
// shrinking image-heavy PDFs (a common case for Chromium output, which embeds
// non-JPEG images losslessly) while leaving text, vectors, fonts and structure
// untouched. See https://github.com/gotenberg/gotenberg/issues/359.
//
// Only lossless (FlateDecode), non-CMYK, non-masked images at or above
// [minOptimizeImageSize] are touched. Already-compressed, transparent, CMYK and
// small images are skipped so the pass never enlarges a file or corrupts
// transparency. It never fails the conversion for a single unreadable image; it
// logs and moves on, and returns the input unchanged when nothing qualifies.
func (engine *PdfCpu) OptimizeImages(ctx context.Context, logger *slog.Logger, imageQuality int, inputPath string) error {
	ctx, span := gotenberg.Tracer().Start(ctx, "pdfcpu.OptimizeImages",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(engine.spanAttrs()...),
	)
	defer span.End()

	fail := func(err error) error {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	images, err := engine.listImages(ctx, inputPath)
	if err != nil {
		return fail(fmt.Errorf("optimize PDF images with pdfcpu: %w", err))
	}

	var targets []pdfcpuImage
	for _, img := range images {
		if optimizableImage(img) {
			targets = append(targets, img)
		}
	}
	if len(targets) == 0 {
		logger.DebugContext(ctx, "no images to optimize")
		span.SetStatus(codes.Ok, "")
		return nil
	}

	workDir, err := os.MkdirTemp(filepath.Dir(inputPath), "optimize-images-*")
	if err != nil {
		return fail(fmt.Errorf("optimize PDF images with pdfcpu: create work directory: %w", err))
	}
	defer func() {
		if err := os.RemoveAll(workDir); err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("remove image optimization work directory: %v", err))
		}
	}()

	extracted, err := engine.extractImages(ctx, logger, inputPath, workDir)
	if err != nil {
		return fail(fmt.Errorf("optimize PDF images with pdfcpu: %w", err))
	}

	// Chain one update per image. Each update writes a fresh file; current holds
	// the latest successful output, so a single failed image is skipped without
	// discarding the ones already done. The input is only replaced on success.
	current := inputPath
	optimized := 0
	for _, img := range targets {
		src, ok := extracted[img.id]
		if !ok {
			logger.WarnContext(ctx, fmt.Sprintf("optimize images: image %s was not extracted, leaving it as is", img.id))
			continue
		}

		reencoded := filepath.Join(workDir, img.id+".jpg")
		err = reencodeToJpeg(src, reencoded, imageQuality)
		if err != nil {
			logger.WarnContext(ctx, fmt.Sprintf("optimize images: re-encode %s: %v; leaving it as is", img.id, err))
			continue
		}

		next := filepath.Join(workDir, fmt.Sprintf("optimized-%d.pdf", optimized))
		err = engine.updateImage(ctx, logger, current, reencoded, next, img.obj)
		if err != nil {
			logger.WarnContext(ctx, fmt.Sprintf("optimize images: update %s: %v; leaving it as is", img.id, err))
			continue
		}

		current = next
		optimized++
	}

	if optimized == 0 {
		span.SetStatus(codes.Ok, "")
		return nil
	}

	err = os.Rename(current, inputPath)
	if err != nil {
		return fail(fmt.Errorf("optimize PDF images with pdfcpu: replace input: %w", err))
	}

	logger.DebugContext(ctx, fmt.Sprintf("optimized %d image(s) at quality %d", optimized, imageQuality))
	span.SetStatus(codes.Ok, "")
	return nil
}

// optimizableImage reports whether an image is a safe, worthwhile target: a
// lossless (FlateDecode), non-CMYK, non-masked image at or above the size
// threshold. Everything else is left untouched.
func optimizableImage(img pdfcpuImage) bool {
	switch {
	case img.filter != "FlateDecode":
		return false // Already compressed (JPEG/JPX); re-encoding would only add loss.
	case img.comp == 4:
		return false // CMYK; a JPEG round-trip is unsafe.
	case img.masked:
		return false // Soft mask, image mask or alpha; JPEG has no transparency.
	case img.bytes < minOptimizeImageSize:
		return false
	default:
		return true
	}
}

// listImages runs `pdfcpu images list` and parses its table. The command writes
// to stdout, so it is run directly to capture the output.
func (engine *PdfCpu) listImages(ctx context.Context, inputPath string) ([]pdfcpuImage, error) {
	cmd := exec.CommandContext(ctx, engine.binPath, "images", "list", inputPath) //nolint:gosec // binPath is validated at Provision; inputPath is a Gotenberg working file.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("run pdfcpu images list: %w", err)
	}

	return parseImagesList(stdout.String()), nil
}

// parseImagesList parses the fixed-column table of `pdfcpu images list`. Columns
// are separated by U+2502; the header and separator rows are skipped because
// their second column is not a numeric object number.
func parseImagesList(output string) []pdfcpuImage {
	var images []pdfcpuImage
	for _, line := range strings.Split(output, "\n") {
		cols := strings.Split(line, "│")
		if len(cols) < 9 {
			continue
		}

		obj, err := strconv.Atoi(strings.TrimSpace(cols[1]))
		if err != nil {
			continue
		}

		comp := 0
		if fields := strings.Fields(cols[6]); len(fields) >= 2 {
			comp, _ = strconv.Atoi(fields[1])
		}

		images = append(images, pdfcpuImage{
			obj:    obj,
			id:     strings.TrimSpace(cols[2]),
			masked: strings.TrimSpace(cols[3]) != "image",
			comp:   comp,
			bytes:  parseHumanSize(cols[7]),
			filter: strings.TrimSpace(cols[8]),
		})
	}

	return images
}

// parseHumanSize converts a pdfcpu size cell such as "4.4 MB" or "194 KB" into
// a byte count.
func parseHumanSize(cell string) int64 {
	fields := strings.Fields(cell)
	if len(fields) == 0 {
		return 0
	}

	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}

	multiplier := float64(1)
	if len(fields) > 1 {
		switch strings.ToUpper(fields[1]) {
		case "KB":
			multiplier = 1 << 10
		case "MB":
			multiplier = 1 << 20
		case "GB":
			multiplier = 1 << 30
		}
	}

	return int64(value * multiplier)
}

// extractImages extracts every image of inputPath into dir and returns a map of
// image Id (e.g. "X6") to the extracted file path.
func (engine *PdfCpu) extractImages(ctx context.Context, logger *slog.Logger, inputPath, dir string) (map[string]string, error) {
	args := []string{"images", "extract", inputPath, dir}
	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return nil, fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return nil, fmt.Errorf("extract images: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read extracted images: %w", err)
	}

	extracted := make(map[string]string, len(entries))
	for _, entry := range entries {
		if match := pdfcpuListRowID.FindStringSubmatch(entry.Name()); match != nil {
			extracted[match[1]] = filepath.Join(dir, entry.Name())
		}
	}

	return extracted, nil
}

// updateImage replaces the image object objNr of inFile with the image at
// imagePath, writing the result to outFile. The replacement must share the
// original image dimensions, which reencodeToJpeg preserves.
func (engine *PdfCpu) updateImage(ctx context.Context, logger *slog.Logger, inFile, imagePath, outFile string, objNr int) error {
	args := []string{"images", "update", inFile, imagePath, outFile, strconv.Itoa(objNr)}
	cmd, err := gotenberg.CommandContext(ctx, logger, engine.binPath, args...)
	if err != nil {
		return fmt.Errorf("create command: %w", err)
	}

	_, err = cmd.Exec()
	if err != nil {
		return fmt.Errorf("update image: %w", err)
	}

	return nil
}

// reencodeToJpeg decodes the image at src and writes it to dst as JPEG at the
// given quality, keeping the original pixel dimensions (pdfcpu requires the
// replacement to match). quality is clamped to the valid 1 to 100 range.
func reencodeToJpeg(src, dst string, quality int) error {
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}

	in, err := os.Open(src) //nolint:gosec // src is a file this package extracted into its own temp dir.
	if err != nil {
		return fmt.Errorf("open image: %w", err)
	}
	defer in.Close()

	img, _, err := image.Decode(in)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	out, err := os.Create(dst) //nolint:gosec // dst is a file in this package's own temp dir.
	if err != nil {
		return fmt.Errorf("create re-encoded image: %w", err)
	}
	defer out.Close()

	err = jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return fmt.Errorf("encode JPEG: %w", err)
	}

	return nil
}
