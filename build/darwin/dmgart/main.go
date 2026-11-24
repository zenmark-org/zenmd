// Command dmgart generates the DMG art from build/appicon.svg and
// build/appicon.png, so it can be regenerated whenever the icon changes
// (`task icon` runs it):
//
//	go run ./build/darwin/dmgart [-name Zenmark] [-svg build/appicon.svg] [-icon build/appicon.png] [-out build/darwin]
//
// It writes dmg-background.png (540x380), dmg-background@2x.png (1080x760) and
// dmg-file-icon.icns (the .dmg file) and md-document.icns (the Finder icon of
// Markdown files: a white page with a folded corner, the app mark and "MD").
// The background is black type and an arrow on white; the
// SVG is rasterised with macOS QuickLook (qlmanage), like `task icon` does.
// The icon is composed and scaled with the standard library only.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	winW, winH = 540, 380
	// Where `wails3 tool package --format dmg` puts the two icons: x at 28% and
	// 72% of the width, y at height/2-26 (icon centres, 96 px icons).
	iconY       = winH/2 - 26
	appX, appsX = winW * 28 / 100, winW * 72 / 100
)

type options struct{ svg, icon, out, name string }

func main() {
	var o options
	flag.StringVar(&o.name, "name", "", "product name in the title (default: info.productName of build/config.yml)")
	flag.StringVar(&o.svg, "svg", "build/appicon.svg", "app icon (SVG), for the check that it exists")
	flag.StringVar(&o.icon, "icon", "build/appicon.png", "app icon (1024 px PNG with transparent margin)")
	flag.StringVar(&o.out, "out", "build/darwin", "output directory")
	flag.Parse()
	if o.name == "" {
		o.name = productName("build/config.yml")
	}
	if err := run(o); err != nil {
		fmt.Fprintf(os.Stderr, "dmgart: %v\n", err)
		os.Exit(1)
	}
}

// productName reads info.productName from the build config (one variable
// names the product everywhere).
func productName(cfg string) string {
	b, err := os.ReadFile(cfg)
	if err != nil {
		return "Zenmark"
	}
	for _, l := range bytes.Split(b, []byte("\n")) {
		t := bytes.TrimSpace(l)
		if bytes.HasPrefix(t, []byte("productName:")) {
			v := bytes.TrimSpace(bytes.TrimPrefix(t, []byte("productName:")))
			if i := bytes.Index(v, []byte(" #")); i >= 0 {
				v = bytes.TrimSpace(v[:i])
			}
			return string(bytes.Trim(v, `"`))
		}
	}
	return "Zenmark"
}

// backgroundSVG is the art on a square canvas (QuickLook renders SVG onto a
// square): the DMG window is its top 540x380 part.
func backgroundSVG(name string) string {
	const f = `font-family="Georgia, 'Times New Roman', serif"`
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="540" height="540" viewBox="0 0 540 540">
  <rect width="540" height="540" fill="#fff"/>
  <text x="270" y="72" %[1]s font-size="26" text-anchor="middle" fill="#000">Drag %[2]s to Applications</text>
  <g data-arrow="1" fill="none" stroke="#000" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
    <path d="M%[3]d %[5]d H%[4]d"/>
    <path d="M%[6]d %[7]d L%[4]d %[5]d L%[6]d %[8]d"/>
  </g>
  <text x="270" y="330" %[1]s font-size="13" font-style="italic" text-anchor="middle" fill="#000">Markdown reader and editor</text>
</svg>
`, f, html.EscapeString(name), appX+70, appsX-70, iconY, appsX-84, iconY-12, iconY+12)
}

func run(o options) error {
	if _, err := os.Stat(o.svg); err != nil {
		return err
	}
	if err := os.MkdirAll(o.out, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "dmgart")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	svg := filepath.Join(tmp, "background.svg")
	if err := os.WriteFile(svg, []byte(backgroundSVG(o.name)), 0o644); err != nil {
		return err
	}
	for _, s := range []struct {
		name string
		k    int
	}{{"dmg-background.png", 1}, {"dmg-background@2x.png", 2}} {
		img, err := rasterise(svg, 540*s.k, tmp)
		if err != nil {
			return err
		}
		if err := writePNG(filepath.Join(o.out, s.name), crop(img, winW*s.k, winH*s.k)); err != nil {
			return err
		}
	}
	f, err := os.Open(o.icon)
	if err != nil {
		return err
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		return err
	}
	data, err := icns(fileIcon(src))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(o.out, "dmg-file-icon.icns"), data, 0o644); err != nil {
		return err
	}
	md, err := mdIcns(src)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(o.out, "md-document.icns"), md, 0o644)
}

// rasterise renders svg with QuickLook at size x size.
func rasterise(svg string, size int, tmp string) (image.Image, error) {
	dir := filepath.Join(tmp, fmt.Sprint(size))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if out, err := exec.Command("qlmanage", "-t", "-s", fmt.Sprint(size), "-o", dir, svg).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("qlmanage: %v\n%s", err, out)
	}
	f, err := os.Open(filepath.Join(dir, filepath.Base(svg)+".png"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
		return nil, fmt.Errorf("QuickLook rendered %v, want %dx%d", b, size, size)
	}
	return img, nil
}

// crop returns the top-left w x h part as an opaque grayscale-safe NRGBA.
func crop(img image.Image, w, h int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Over)
	return out
}

func writePNG(path string, img image.Image) error {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return err
	}
	return os.WriteFile(path, b.Bytes(), 0o644)
}

// fileIcon is the icon of the .dmg file: the app tile standing on a disk
// slab, on a 1024 px transparent canvas.
func fileIcon(tile image.Image) *image.NRGBA {
	const n = 1024
	out := image.NewNRGBA(image.Rect(0, 0, n, n))
	// The slab: a black rounded bar.
	slab := image.Rect(150, 780, 874, 900)
	fillRound(out, slab, 40, color.NRGBA{0, 0, 0, 255})
	fillRound(out, image.Rect(190, 830, 520, 850), 10, color.NRGBA{255, 255, 255, 255})
	// The tile, scaled to 80% (the source has a 100 px transparent margin).
	small := scale(tile, 820)
	off := (n - 820) / 2
	draw.Draw(out, image.Rect(off, off-110, off+820, off-110+820), small, image.Point{}, draw.Over)
	return out
}

func fillRound(dst *image.NRGBA, r image.Rectangle, rad int, c color.NRGBA) {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			cx := min(max(x, r.Min.X+rad), r.Max.X-1-rad)
			cy := min(max(y, r.Min.Y+rad), r.Max.Y-1-rad)
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= rad*rad {
				dst.SetNRGBA(x, y, c)
			}
		}
	}
}

// scale resamples img to size x size by averaging (premultiplied), which is
// exact for the downscales used here and deterministic.
func scale(img image.Image, size int) *image.NRGBA {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	sx, sy := float64(b.Dx())/float64(size), float64(b.Dy())/float64(size)
	for y := 0; y < size; y++ {
		y0, y1 := float64(y)*sy, float64(y+1)*sy
		for x := 0; x < size; x++ {
			x0, x1 := float64(x)*sx, float64(x+1)*sx
			var r, g, bl, a, wsum float64
			for yy := int(y0); float64(yy) < y1 && yy < b.Dy(); yy++ {
				wy := min(float64(yy+1), y1) - max(float64(yy), y0)
				for xx := int(x0); float64(xx) < x1 && xx < b.Dx(); xx++ {
					w := wy * (min(float64(xx+1), x1) - max(float64(xx), x0))
					pr, pg, pb, pa := img.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
					r += float64(pr) * w
					g += float64(pg) * w
					bl += float64(pb) * w
					a += float64(pa) * w
					wsum += w
				}
			}
			if wsum == 0 || a == 0 {
				continue
			}
			// r,g,bl,a are premultiplied 16-bit sums.
			out.SetNRGBA(x, y, color.NRGBA{
				uint8(min(r/a*255+0.5, 255)), uint8(min(g/a*255+0.5, 255)), uint8(min(bl/a*255+0.5, 255)),
				uint8(min(a/wsum/257+0.5, 255)),
			})
		}
	}
	return out
}

type icnsType struct {
	typ  string
	size int
}

// icns builds a macOS icon file with PNG images at the standard sizes from a
// 1024 px source.
func icns(src image.Image) ([]byte, error) {
	return buildICNS([]icnsType{{"ic11", 32}, {"ic12", 64}, {"ic07", 128}, {"ic08", 256}, {"ic13", 256}, {"ic09", 512}, {"ic14", 512}, {"ic10", 1024}},
		func(n int) image.Image {
			if src.Bounds().Dx() == n {
				return src
			}
			return scale(src, n)
		})
}

// mdIcns is the Markdown document icon at every size from 16 to 1024 px. The
// 16 and 32 px images come from the simplified page (no mark, thick outline).
func mdIcns(tile image.Image) ([]byte, error) {
	full, small := docIcon(tile, false), docIcon(tile, true)
	return buildICNS([]icnsType{{"icp4", 16}, {"ic11", 32}, {"icp5", 32}, {"ic12", 64}, {"ic07", 128}, {"ic08", 256}, {"ic13", 256}, {"ic09", 512}, {"ic14", 512}, {"ic10", 1024}},
		func(n int) image.Image {
			switch {
			case n == 1024:
				return full
			case n <= 32:
				return scale(small, n)
			}
			return scale(full, n)
		})
}

func buildICNS(types []icnsType, get func(size int) image.Image) ([]byte, error) {
	var body bytes.Buffer
	for _, t := range types {
		var p bytes.Buffer
		if err := png.Encode(&p, get(t.size)); err != nil {
			return nil, err
		}
		body.WriteString(t.typ)
		_ = binary.Write(&body, binary.BigEndian, uint32(p.Len()+8))
		body.Write(p.Bytes())
	}
	var out bytes.Buffer
	out.WriteString("icns")
	_ = binary.Write(&out, binary.BigEndian, uint32(body.Len()+8))
	out.Write(body.Bytes())
	return out.Bytes(), nil
}

// Page geometry on the 1024 px canvas.
const (
	pgL, pgR, pgT, pgB = 192, 832, 64, 960
	pgRad, pgFold      = 56, 210
)

var (
	black = color.NRGBA{0, 0, 0, 255}
	white = color.NRGBA{255, 255, 255, 255}
)

// inPage reports whether (x,y) is inside the page shape shrunk by inset:
// a rounded rectangle with the top-right corner cut off diagonally.
func inPage(x, y, inset int) bool {
	l, r, t, b, rad := pgL+inset, pgR-inset, pgT+inset, pgB-inset, max(pgRad-inset, 0)
	if x < l || x >= r || y < t || y >= b {
		return false
	}
	cx, cy := min(max(x, l+rad), r-1-rad), min(max(y, t+rad), b-1-rad)
	if (x-cx)*(x-cx)+(y-cy)*(y-cy) > rad*rad {
		return false
	}
	// The cut corner: dx, dy measured from the fold's inner corner.
	dx, dy := x-(pgR-pgFold), (pgT+pgFold)-y
	return !(dx > 0 && dy > 0 && float64(dx+dy) > float64(pgFold)-float64(inset)*1.4143)
}

// docIcon draws the Markdown document icon on a 1024 px canvas: a white page
// with a dark outline and a folded corner, the app mark and "MD". The small
// variant (for 16 and 32 px) drops the mark and thickens outline and letters.
func docIcon(tile image.Image, small bool) *image.NRGBA {
	const n = 1024
	out := image.NewNRGBA(image.Rect(0, 0, n, n))
	w := 18
	if small {
		w = 56
	}
	for y := pgT; y < pgB; y++ {
		for x := pgL; x < pgR; x++ {
			switch {
			case inPage(x, y, w):
				out.SetNRGBA(x, y, white)
			case inPage(x, y, 0):
				out.SetNRGBA(x, y, black)
			}
		}
	}
	// The fold: a white flap outlined in black.
	fx, fy := pgR-pgFold, pgT+pgFold
	for y := pgT; y <= fy; y++ {
		for x := fx; x < pgR; x++ {
			dx, dy := x-fx, fy-y
			if dx+dy > pgFold || !inPage(x, y, 0) {
				continue
			}
			c := white
			if dx < w || dy < w || pgFold-(dx+dy) < int(float64(w)*0.7) {
				c = black
			}
			out.SetNRGBA(x, y, c)
		}
	}
	if small {
		letters(out, 512, 410, 240)
	} else {
		// The mark: the tile without its transparent margin.
		const m, ms = 100, 340
		crop := image.NewNRGBA(image.Rect(0, 0, 824, 824))
		draw.Draw(crop, crop.Bounds(), tile, image.Pt(m, m), draw.Src)
		draw.Draw(out, image.Rect(512-ms/2, 190, 512+ms/2, 190+ms), scale(crop, ms), image.Point{}, draw.Over)
		letters(out, 512, 640, 200)
	}
	return out
}

// letters draws "MD" in black, centred on cx, top at y, h pixels tall, from
// strokes and rounded rectangles (no fonts).
func letters(dst *image.NRGBA, cx, y, h int) {
	st := h / 5
	wm, wd, gap := h*85/100, h*78/100, h*28/100
	x := cx - (wm+gap+wd)/2
	// M: two stems and a V between them.
	fillRound(dst, image.Rect(x, y, x+st, y+h), 0, black)
	fillRound(dst, image.Rect(x+wm-st, y, x+wm, y+h), 0, black)
	mid := x + wm/2
	line(dst, float64(x+st/2), float64(y+st/2), float64(mid), float64(y+h*62/100), float64(st)*0.55)
	line(dst, float64(x+wm-st/2), float64(y+st/2), float64(mid), float64(y+h*62/100), float64(st)*0.55)
	// D: a stem and a bowl.
	dx := x + wm + gap
	fillRound(dst, image.Rect(dx, y, dx+wd, y+h), h*38/100, black)
	fillRound(dst, image.Rect(dx, y, dx+h*30/100, y+h), 0, black)
	fillRound(dst, image.Rect(dx+st, y+st, dx+wd-st, y+h-st), max(h*38/100-st, 0), white)
	fillRound(dst, image.Rect(dx+st, y+st, dx+st+h*20/100, y+h-st), 0, white)
}

// line fills a thick segment (radius r) in black.
func line(dst *image.NRGBA, x0, y0, x1, y1, r float64) {
	dx, dy := x1-x0, y1-y0
	l2 := dx*dx + dy*dy
	for y := int(min(y0, y1) - r); y <= int(max(y0, y1)+r); y++ {
		for x := int(min(x0, x1) - r); x <= int(max(x0, x1)+r); x++ {
			t := min(max(((float64(x)-x0)*dx+(float64(y)-y0)*dy)/l2, 0), 1)
			px, py := x0+t*dx-float64(x), y0+t*dy-float64(y)
			if px*px+py*py <= r*r {
				dst.SetNRGBA(x, y, black)
			}
		}
	}
}
