package main

// Visualizer for lem-in output (PNG frames)
// Reads echoed map + moves from STDIN, writes one PNG per turn to -out directory.
// Usage:
//   go run visualizer.go -out frames -w 1200 -h 800 < run_output.txt
// Or:
//   ./lem-in example00.txt | go run visualizer.go -out frames
//
// Only standard library packages are used.

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Room struct {
	X, Y int
}

type Input struct {
	Ants   int
	Rooms  map[string]Room
	Links  [][2]string
	Start  string
	End    string
	Turns  [][]Move // per turn
}

type Move struct {
	Ant  int
	Room string
}

var (
	flagOut    = flag.String("out", "frames", "output directory for PNG frames")
	flagW      = flag.Int("w", 1200, "image width")
	flagH      = flag.Int("h", 800, "image height")
	flagMargin = flag.Int("m", 30, "margin pixels")
	flagNodeR  = flag.Int("r", 10, "room node radius")
	flagAntR   = flag.Int("ar", 6, "ant dot radius")
	flagFooter = flag.Int("footer", 36, "footer height in pixels")
)

func main() {
	flag.Parse()
	inp, err := readInput(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "visualizer: parse error:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(*flagOut, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "visualizer: cannot create output dir:", err)
		os.Exit(1)
	}

	frames := len(inp.Turns)
	// start frame + one per turn
	total := frames + 1

	// Precompute layout scaling
	positions := layout(inp.Rooms, *flagW, *flagH, *flagMargin, *flagFooter)

	// Initial ant positions: all at Start
	antPos := map[int]string{}
	for i := 1; i <= inp.Ants; i++ {
		if inp.Start != "" {
			antPos[i] = inp.Start
		}
	}

	// Render frame 0
	if err := renderFrame(0, inp, positions, antPos); err != nil {
		fmt.Fprintln(os.Stderr, "visualizer: render error:", err)
		os.Exit(1)
	}

	// Apply turns and render subsequent frames
	for t := 0; t < frames; t++ {
		for _, m := range inp.Turns[t] {
			antPos[m.Ant] = m.Room
		}
		if err := renderFrame(t+1, inp, positions, antPos); err != nil {
			fmt.Fprintln(os.Stderr, "visualizer: render error:", err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "visualizer: wrote %d frames to %s\n", total, *flagOut)
}

// ---------------- Parsing ----------------

func readInput(r *os.File) (*Input, error) {
	sc := bufio.NewScanner(r)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	header, moves := splitHeaderMoves(lines)
	inp := &Input{Rooms: map[string]Room{}}
	if err := parseHeader(header, inp); err != nil {
		return nil, err
	}
	turns, err := parseMoves(moves)
	if err != nil {
		return nil, err
	}
	inp.Turns = turns
	return inp, nil
}

func splitHeaderMoves(lines []string) (header []string, moves []string) {
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			return lines[:i], lines[i+1:]
		}
	}
	return lines, nil
}

func parseHeader(header []string, inp *Input) error {
	pendingStart := false
	pendingEnd := false

	for idx, ln := range header {
		ln = strings.TrimRight(ln, "\r\n")
		if idx == 0 {
			// try ants on first non-empty, non-comment line;
			// but the header may have comments first; handle more generally below
		}

		if ln == "" {
			continue
		}
		if strings.HasPrefix(ln, "#") {
			switch ln {
			case "##start":
				pendingStart = true
				pendingEnd = false
			case "##end":
				pendingEnd = true
				pendingStart = false
			}
			continue
		}

		// Ants: first time we can parse a single integer
		if inp.Ants == 0 {
			if v, err := strconv.Atoi(strings.TrimSpace(ln)); err == nil {
				if v < 0 {
					return fmt.Errorf("negative ants count")
				}
				inp.Ants = v
				continue
			}
		}

		// Room: 3 fields
		fields := strings.Fields(ln)
		if len(fields) == 3 {
			x, err1 := strconv.Atoi(fields[1])
			y, err2 := strconv.Atoi(fields[2])
			if err1 != nil || err2 != nil {
				// not a valid room, fallthrough to link
			} else {
				name := fields[0]
				inp.Rooms[name] = Room{X: x, Y: y}
				if pendingStart {
					inp.Start = name
					pendingStart = false
				} else if pendingEnd {
					inp.End = name
					pendingEnd = false
				}
				continue
			}
		}

		// Link: "a-b" with no spaces
		if strings.Count(ln, "-") == 1 && !strings.Contains(ln, " ") {
			parts := strings.SplitN(ln, "-", 2)
			inp.Links = append(inp.Links, [2]string{parts[0], parts[1]})
			continue
		}
	}
	return nil
}

var (
	reMoveFull   = regexp.MustCompile(`^L(\d+)-(.+)$`)
	reMoveSplitL = regexp.MustCompile(`^L(\d+)$`)
	reMoveSplitR = regexp.MustCompile(`^-(.+)$`)
)

func parseMoves(lines []string) ([][]Move, error) {
	var turns [][]Move
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		toks := strings.Fields(ln)
		var turn []Move
		for i := 0; i < len(toks); i++ {
			t := toks[i]
			if m := reMoveFull.FindStringSubmatch(t); m != nil {
				ant, _ := strconv.Atoi(m[1])
				turn = append(turn, Move{Ant: ant, Room: m[2]})
				continue
			}
			// be forgiving for accidental space like "L16 -1"
			if m := reMoveSplitL.FindStringSubmatch(t); m != nil && i+1 < len(toks) {
				if m2 := reMoveSplitR.FindStringSubmatch(toks[i+1]); m2 != nil {
					ant, _ := strconv.Atoi(m[1])
					turn = append(turn, Move{Ant: ant, Room: m2[1]})
					i++
					continue
				}
			}
		}
		if len(turn) > 0 {
			turns = append(turns, turn)
		}
	}
	return turns, nil
}

// ---------------- Layout & Rendering ----------------

type Pt struct{ X, Y int }

func layout(rooms map[string]Room, W, H, margin, footer int) map[string]Pt {
	pos := make(map[string]Pt, len(rooms))
	if len(rooms) == 0 {
		return pos
	}
	minx, maxx := math.MaxInt, math.MinInt
	miny, maxy := math.MaxInt, math.MinInt
	for _, r := range rooms {
		if r.X < minx { minx = r.X }
		if r.X > maxx { maxx = r.X }
		if r.Y < miny { miny = r.Y }
		if r.Y > maxy { maxy = r.Y }
	}
	spanx := maxx - minx
	spany := maxy - miny
	if spanx == 0 { spanx = 1 }
	if spany == 0 { spany = 1 }

	innerW := W - 2*margin
	innerH := (H - footer) - 2*margin
	if innerW < 10 { innerW = 10 }
	if innerH < 10 { innerH = 10 }

	for name, r := range rooms {
		nx := float64(r.X - minx) / float64(spanx)
		ny := float64(r.Y - miny) / float64(spany)
		x := margin + int(nx*float64(innerW))
		y := margin + int(ny*float64(innerH))
		if x < 0 { x = 0 }
		if y < 0 { y = 0 }
		if x >= W { x = W-1 }
		limitY := H - footer - 1
		if y >= limitY { y = limitY }
		pos[name] = Pt{X: x, Y: y}
	}
	return pos
}

func renderFrame(idx int, inp *Input, pos map[string]Pt, antPos map[int]string) error {
	W, H := *flagW, *flagH
	img := image.NewRGBA(image.Rect(0, 0, W, H))

	// background
	bg := color.RGBA{245, 246, 248, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	// draw links
	linkC := color.RGBA{190, 196, 205, 255}
	for _, e := range inp.Links {
		a, aok := pos[e[0]]
		b, bok := pos[e[1]]
		if !aok || !bok { continue }
		drawLine(img, a.X, a.Y, b.X, b.Y, linkC)
	}

	// draw rooms
	for name, p := range pos {
		c := color.RGBA{80, 120, 200, 255}
		if name == inp.Start {
			c = color.RGBA{70, 170, 80, 255}
		} else if name == inp.End {
			c = color.RGBA{220, 80, 80, 255}
		}
		fillCircle(img, p.X, p.Y, *flagNodeR, c)
		// inner dot
		fillCircle(img, p.X, p.Y, 2, color.RGBA{255,255,255,255})
	}

	// draw ants
	// resolve multiplicity per room
	type bucket struct{ Name string; P Pt; Count int }
	roomCount := map[string]int{}
	for _, room := range antPos {
		roomCount[room]++
	}
	// stable ordering keys: room name
	var keys []string
	for k := range roomCount { keys = append(keys, k) }
	sort.Strings(keys)
	for _, room := range keys {
		count := roomCount[room]
		p, ok := pos[room]
		if !ok { continue }
		// single ant: draw filled circle
		if count == 1 {
			fillCircle(img, p.X, p.Y, *flagAntR, color.RGBA{40,40,40,200})
		} else {
			// multiple: draw ring to hint many ants
			drawCircle(img, p.X, p.Y, *flagAntR+1, color.RGBA{30,30,30,220})
			fillCircle(img, p.X, p.Y, *flagAntR-2, color.RGBA{40,40,40,180})
		}
	}

	// footer bar
	fh := *flagFooter
	if fh < 24 { fh = 24 }
	drawRect(img, 0, H-fh, W, fh, color.RGBA{230, 232, 236, 255})
	// separator line
	drawRect(img, 0, H-fh-1, W, 1, color.RGBA{200, 204, 210, 255})
	// legend placeholder (std lib has no fonts)

	// write file
	fn := filepath.Join(*flagOut, fmt.Sprintf("turn_%04d.png", idx))
	f, err := os.Create(fn)
	if err != nil { return err }
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return nil
}

// Basic drawing helpers (stdlib only)

func set(img *image.RGBA, x, y int, c color.Color) {
	if !(image.Pt(x, y).In(img.Bounds())) { return }
	img.Set(x, y, c)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.Color) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx := -1
	if x0 < x1 { sx = 1 }
	sy := -1
	if y0 < y1 { sy = 1 }
	err := dx + dy
	for {
		set(img, x0, y0, c)
		if x0 == x1 && y0 == y1 { break }
		e2 := 2*err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, r int, col color.Color) {
	if r <= 0 { return }
	r2 := r*r
	for y := cy - r; y <= cy + r; y++ {
		for x := cx - r; x <= cx + r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx + dy*dy <= r2 {
				set(img, x, y, col)
			}
		}
	}
}

func drawCircle(img *image.RGBA, cx, cy, r int, col color.Color) {
	if r <= 0 { return }
	r2 := r*r
	th := 2 // thickness
	for y := cy - r; y <= cy + r; y++ {
		for x := cx - r; x <= cx + r; x++ {
			dx := x - cx
			dy := y - cy
			d := dx*dx + dy*dy
			if d <= r2 && d >= (r-th)*(r-th) {
				set(img, x, y, col)
			}
		}
	}
}

func drawRect(img *image.RGBA, x, y, w, h int, col color.Color) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			set(img, xx, yy, col)
		}
	}
}
