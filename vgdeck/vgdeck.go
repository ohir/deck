// vgdeck: slide decks for OpenVG
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/ajstarks/deck"
	"github.com/ajstarks/openvg"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
	"time"
)

// dodeck sets up the graphics environment and kicks off the interaction
func dodeck(filename, searchterm string, pausetime time.Duration, slidenum, cw, ch int, gp float64) {
	w, h := openvg.Init()
	if cw > 0 {
		w = cw
	}
	if ch > 0 {
		h = ch
	}
	openvg.Background(0, 0, 0)
	if pausetime == 0 {
		interact(filename, searchterm, w, h, slidenum, gp)
	} else {
		loop(filename, w, h, pausetime)
	}
	openvg.Finish()
}

// loadimage loads all the images of the deck into a map for later display
func loadimage(d deck.Deck, m map[string]image.Image) {
	for _, s := range d.Slide {
		for _, i := range s.Image {
			f, err := os.Open(i.Name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			img, _, err := image.Decode(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				continue
			}
			m[i.Name] = img
			f.Close()
		}
	}
}

// interact controls the display of the deck
func interact(filename, searchterm string, w, h, slidenum int, gp float64) {
	openvg.SaveTerm()
	defer openvg.RestoreTerm()
	var d deck.Deck
	var err error
	d, err = deck.Read(filename, w, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	openvg.RawTerm()
	r := bufio.NewReader(os.Stdin)
	lastslide := len(d.Slide) - 1
	if slidenum > lastslide {
		slidenum = lastslide
	}
	if slidenum < 0 {
		slidenum = 0
	}
	if len(searchterm) > 0 {
		sr := deck.Search(d, searchterm)
		if sr >= 0 {
			slidenum = sr
		}
	}
	n := slidenum
	xray := 1
	initial := 0
	imap := make(map[string]image.Image)
	// respond to keyboard commands, 'q' to exit
	for cmd := byte('0'); cmd != 'q'; cmd = readcmd(r) {
		switch cmd {
		// read/reload
		case 'r', 18: // r, Ctrl-R
			d, err = deck.Read(filename, w, h)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return
			}
			loadimage(d, imap)
			openvg.Background(0, 0, 0)
			xray = 1
			showslide(d, imap, n)

		// save slide
		case 's', 19: // s, Ctrl-S
			openvg.SaveEnd(fmt.Sprintf("%s-slide-%04d", filename, n))

		// first slide
		case '0', '1', 1, '^': // 0,1,Ctrl-A,^
			initial++
			if initial == 1 {
				loadimage(d, imap)
				n = slidenum
			} else {
				n = 0
			}
			showslide(d, imap, n)

		// last slide
		case '*', 5, '$': // *, Crtl-E, $
			n = lastslide
			showslide(d, imap, n)

		// next slide
		case '+', 'n', '\n', ' ', '\t', 14, 27: // +,n,newline,space,tab,Crtl-N
			n++
			if n > lastslide {
				n = 0
			}
			showslide(d, imap, n)

		// previous slide
		case '-', 'p', 8, 16, 127: // -,p,Backspace,Ctrl-P,Del
			n--
			if n < 0 {
				n = lastslide
			}
			showslide(d, imap, n)

		// x-ray
		case 'x', 24: // x, Ctrl-X
			xray++
			showslide(d, imap, n)
			if xray%2 == 0 {
				showgrid(d, n, gp)
			}

		// search
		case '/', 6: // slash, Ctrl-F
			openvg.RestoreTerm()
			searchterm, serr := r.ReadString('\n')
			openvg.RawTerm()
			if serr != nil {
				continue
			}
			if len(searchterm) > 2 {
				ns := deck.Search(d, searchterm[0:len(searchterm)-1])
				if ns >= 0 {
					showslide(d, imap, ns)
					n = ns
				}
			}
		}
	}
}

// loop through slides with a pause
func loop(filename string, w, h int, n time.Duration) {
	openvg.SaveTerm()
	defer openvg.RestoreTerm()
	var d deck.Deck
	var err error
	d, err = deck.Read(filename, w, h)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	openvg.RawTerm()
	r := bufio.NewReader(os.Stdin)
	imap := make(map[string]image.Image)
	loadimage(d, imap)
	// respond to keyboard commands, 'q' to exit
	for {
		for i := 0; i < len(d.Slide); i++ {
			cmd := readcmd(r)
			if cmd == 'q' {
				return
			}
			showslide(d, imap, i)
			time.Sleep(n)
		}
	}
}

// pct computes percentages
func pct(p float64, m openvg.VGfloat) openvg.VGfloat {
	return openvg.VGfloat((p / 100.0)) * m
}

func pctwidth(p float64, p1, p2 openvg.VGfloat) openvg.VGfloat {
	pw := deck.Pwidth(p, float64(p1), float64(p2))
	return openvg.VGfloat(pw)

}

// showgrid xrays a slide
func showgrid(d deck.Deck, n int, p float64) {
	w := openvg.VGfloat(d.Canvas.Width)
	h := openvg.VGfloat(d.Canvas.Height)
	percent := openvg.VGfloat(p)
	fs := (w / 100) // labels are 1% of the width
	xpct := (percent / 100.0) * w
	ypct := (percent / 100.0) * h

	openvg.StrokeColor("lightgray", 0.5)
	openvg.StrokeWidth(3)

	// horizontal gridlines
	xl := percent
	for x := xpct; x <= w; x += xpct {
		openvg.Line(x, 0, x, h)
		openvg.Text(x, percent, fmt.Sprintf("%.0f%%", xl), "sans", int(fs))
		xl += percent
	}

	// vertical gridlines
	yl := percent
	for y := ypct; y <= h; y += ypct {
		openvg.Line(0, y, w, y)
		openvg.Text(percent, y, fmt.Sprintf("%.0f%%", yl), "sans", int(fs))
		yl += percent
	}

	// show boundary and location of images
	if n < 0 || n > len(d.Slide) {
		return
	}
	for _, im := range d.Slide[n].Image {
		x := pct(im.Xp, w)
		y := pct(im.Yp, h)
		iw := openvg.VGfloat(im.Width)
		ih := openvg.VGfloat(im.Height)
		openvg.FillRGB(127, 0, 0, 0.3)
		openvg.Circle(x, y, fs)
		openvg.FillRGB(255, 0, 0, 0.1)
		openvg.Rect(x-iw/2, y-ih/2, iw, ih)
	}
	openvg.End()
}

//showtext displays text
func showtext(x, y openvg.VGfloat, s, align, font string, fs openvg.VGfloat) {
	fontsize := int(fs)
	switch align {
	case "center", "middle", "mid":
		openvg.TextMid(x, y, s, font, fontsize)
	case "right", "end":
		openvg.TextEnd(x, y, s, font, fontsize)
	default:
		openvg.Text(x, y, s, font, fontsize)
	}
}

// dimen returns device dimemsion from percentages
func dimen(d deck.Deck, x, y, s float64) (xo, yo, so openvg.VGfloat) {
	xf, yf, sf := deck.Dimen(d.Canvas, x, y, s)
	xo, yo, so = openvg.VGfloat(xf), openvg.VGfloat(yf), openvg.VGfloat(sf) * 0.8
	return
}

// showlide displays slides
func showslide(d deck.Deck, imap map[string]image.Image, n int) {
	if n < 0 || n > len(d.Slide)-1 {
		return
	}
	slide := d.Slide[n]
	if slide.Bg == "" {
		slide.Bg = "white"
	}
	if slide.Fg == "" {
		slide.Fg = "black"
	}
	openvg.Start(d.Canvas.Width, d.Canvas.Height)
	cw := openvg.VGfloat(d.Canvas.Width)
	ch := openvg.VGfloat(d.Canvas.Height)
	openvg.FillColor(slide.Bg)
	openvg.Rect(0, 0, cw, ch)
	var x, y, fs openvg.VGfloat

	// every image in the slide
	for _, im := range slide.Image {
		x = pct(im.Xp, cw)
		y = pct(im.Yp, ch)
		midx := openvg.VGfloat(im.Width / 2)
		midy := openvg.VGfloat(im.Height / 2)
		img, ok := imap[im.Name]
		if ok {
			openvg.Img(x-midx, y-midy, img)
		}
		// openvg.Image(x-midx, y-midy, im.Width, im.Height, im.Name)
		if len(im.Caption) > 0 {
			capfs := pctwidth(im.Sp, cw, cw/100)
			if im.Font == "" {
				im.Font = "sans"
			}
			if im.Color == "" {
				openvg.FillColor(slide.Fg)
			} else {
				openvg.FillColor(im.Color)
			}
			if im.Align == "" {
				im.Align = "center"
			}
			switch im.Align {
			case "left", "start":
				x -= midx
			case "right", "end":
				x += midx
			}
			showtext(x, y-((midy)+(capfs*2.0)), im.Caption, im.Align, im.Font, capfs)
		}
	}

	// every graphic on the slide
	const defaultColor = "rgb(127,127,127)"
	const defaultSw = 1.5
	var strokeopacity float64
	// line
	for _, line := range slide.Line {
		if line.Color == "" {
			line.Color = defaultColor
		}
		if line.Opacity == 0 {
			strokeopacity = 1
		} else {
			strokeopacity = line.Opacity/100.0
		}
		x1, y1, sw := dimen(d, line.Xp1, line.Yp1, line.Sp)
		x2, y2, _ := dimen(d, line.Xp2, line.Yp2, 0)
		openvg.StrokeColor(line.Color, openvg.VGfloat(strokeopacity))
		if sw == 0 {
			sw = defaultSw
		}
		openvg.StrokeWidth(openvg.VGfloat(sw))
		openvg.StrokeColor(line.Color)
		openvg.Line(x1, y1, x2, y2)
		openvg.StrokeWidth(0)
	}
	// ellipse
	for _, ellipse := range slide.Ellipse {
		x, y, _ = dimen(d, ellipse.Xp, ellipse.Yp, 0)
		var w, h openvg.VGfloat
		w = pct(ellipse.Wp, cw)
		if ellipse.Hr == 0 { // if relative height not specified, base height on overall height
			h = pct(ellipse.Hp, ch)
		} else {
			h = pct(ellipse.Hr, w)
		}
		if ellipse.Color == "" {
			ellipse.Color = defaultColor
		}
		if ellipse.Opacity == 0 {
			ellipse.Opacity = 1
		} else {
			ellipse.Opacity /= 100
		}
		openvg.FillColor(ellipse.Color, openvg.VGfloat(ellipse.Opacity))
		openvg.Ellipse(x, y, w, h)
	}
	// rect
	for _, rect := range slide.Rect {
		x, y, _ = dimen(d, rect.Xp, rect.Yp, 0)
		var w, h openvg.VGfloat
		w = pct(rect.Wp, cw)
		if rect.Hr == 0 { // if relative height not specified, base height on overall height 
			h = pct(rect.Hp, ch) 
		} else {
			h = pct(rect.Hr, w)
		}
		if rect.Color == "" {
			rect.Color = defaultColor
		}
		if rect.Opacity == 0 {
			rect.Opacity = 1
		} else {
			rect.Opacity /= 100
		}
		openvg.FillColor(rect.Color, openvg.VGfloat(rect.Opacity))
		openvg.Rect(x-(w/2), y-(h/2), w, h)
	}
	// curve
	for _, curve := range slide.Curve {
		if curve.Color == "" {
			curve.Color = defaultColor
		}
		if curve.Opacity == 0 {
			strokeopacity = 1 
		} else {
			strokeopacity = curve.Opacity/100.0
		}
		x1, y1, sw := dimen(d, curve.Xp1, curve.Yp1, curve.Sp)
		x2, y2, _ := dimen(d, curve.Xp2, curve.Yp2, 0)
		x3, y3, _ := dimen(d, curve.Xp3, curve.Yp3, 0)
		openvg.StrokeColor(curve.Color, openvg.VGfloat(strokeopacity))
		openvg.FillColor(slide.Bg, openvg.VGfloat(curve.Opacity))
		if sw == 0 {
			sw = defaultSw
		}
		openvg.StrokeWidth(sw)
		openvg.Qbezier(x1, y1, x2, y2, x3, y3)
		openvg.StrokeWidth(0)
	}

	// arc
	for _, arc := range slide.Arc {
		if arc.Color == "" {
			arc.Color = defaultColor
		}
		if arc.Opacity == 0 {
			strokeopacity = 1
		} else {
			strokeopacity = arc.Opacity/100.0
		}
		ax, ay, sw := dimen(d, arc.Xp, arc.Yp, arc.Sp)
		w := pct(arc.Wp, cw)
		h := pct(arc.Hp, cw)
		openvg.StrokeColor(arc.Color, openvg.VGfloat(strokeopacity))
		openvg.FillColor(slide.Bg, openvg.VGfloat(arc.Opacity))
		if sw == 0 {
			sw = defaultSw
		}
		openvg.StrokeWidth(sw)
		openvg.Arc(ax, ay, w, h, openvg.VGfloat(arc.A1), openvg.VGfloat(arc.A2))
		openvg.StrokeWidth(0)
	}
	openvg.FillColor(slide.Fg)

	// every list in the slide
	var offset, textopacity openvg.VGfloat
	const blinespacing = 2.4
	for _, l := range slide.List {
		if l.Font == "" {
			l.Font = "sans"
		}
		x, y, fs = dimen(d, l.Xp, l.Yp, l.Sp)
		if l.Type == "bullet" {
			offset = 1.2 * fs
		} else {
			offset = 0
		}
		if l.Opacity == 0 {
			textopacity = 1
		} else {
			textopacity = openvg.VGfloat(l.Opacity/100)
		}
		// every list item
		var li, lifont string
		for ln, tl := range l.Li {
			if len(l.Color) > 0  {
				openvg.FillColor(l.Color, textopacity)
			} else {
				openvg.FillColor(slide.Fg)
			}
			if l.Type == "bullet" {
				boffset := fs / 2
				openvg.Ellipse(x, y+boffset, boffset, boffset)
				//openvg.Rect(x, y+boffset/2, boffset, boffset)
			}
			if l.Type == "number" {
				li = fmt.Sprintf("%d. ", ln+1) + tl.ListText 
			} else {
				li = tl.ListText
			}
			if len(tl.Color) > 0 {
				openvg.FillColor(tl.Color, textopacity)
			}
			if len(tl.Font) > 0 {
				lifont = tl.Font
			} else {
				lifont = l.Font
			}
			showtext(x+offset, y, li, l.Align, lifont, fs)
			y -= fs * blinespacing
		}
	}
	openvg.FillColor(slide.Fg)

	// every text in the slide
	const linespacing =  1.8 

	for _, t := range slide.Text {
		if t.Font == "" {
			t.Font = "sans"
		}
		if t.Opacity == 0 {
			textopacity = 1
		} else {
			textopacity = openvg.VGfloat(t.Opacity/100)
		}
		x, y, fs = dimen(d, t.Xp, t.Yp, t.Sp)
		td := strings.Split(t.Tdata, "\n")
		if t.Type == "code" {
			t.Font = "mono"
			tdepth := ((fs * linespacing) * openvg.VGfloat(len(td))) + fs
			openvg.FillColor("rgb(240,240,240)")
			openvg.Rect(x-20, y-tdepth+(fs*linespacing), pctwidth(t.Wp, cw, cw-x-20), tdepth)
		}
		if t.Color == "" {
			openvg.FillColor(slide.Fg, textopacity)
		} else {
			openvg.FillColor(t.Color, textopacity)
		}
		if t.Type == "block" {
			textwrap(x, y, pctwidth(t.Wp, cw, cw/2), t.Tdata, t.Font, fs, fs*linespacing, 0.3)
		} else {
			// every text line
			for _, txt := range td {
				showtext(x, y, txt, t.Align, t.Font, fs)
				y -= (fs * linespacing)
			}
		}
	}
	openvg.FillColor(slide.Fg)
	openvg.End()
}

// whitespace determines if a rune is whitespace
func whitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '-'
}

// textwrap draws text at location, wrapping at the specified width
func textwrap(x, y, w openvg.VGfloat, s string, font string, fs, leading, factor openvg.VGfloat) {
	size := int(fs)
	wordspacing := openvg.TextWidth("m", font, size)
	words := strings.FieldsFunc(s, whitespace)
	xp := x
	yp := y
	edge := x + w
	for _, s := range words {
		tw := openvg.TextWidth(s, font, size)
		openvg.Text(xp, yp, s, font, size)
		xp += tw + (wordspacing * factor)
		if xp > edge {
			xp = x
			yp -= leading
		}
	}
}

// readcmd reads interaction commands
func readcmd(r *bufio.Reader) byte {
	s, err := r.ReadByte()
	if err != nil {
		return 'e'
	}
	return s
}

// for every file, make a deck
func main() {
	var pause = flag.Duration("loop", 0, "loop, pausing the specified duration between slides")
	var search = flag.String("search", "", "search term")
	var gridpct = flag.Float64("g", 10, "Grid percentage")
	var slidenum = flag.Int("slide", 0, "initial slide")
	var cw = flag.Int("w", 0, "canvas width")
	var ch = flag.Int("h", 0, "canvas height")
	flag.Parse()
	for _, f := range flag.Args() {
		dodeck(f, *search, *pause, *slidenum, *cw, *ch, *gridpct)
	}
}
