package poster

import (
	"bytes"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github-badges-backend/internal/stats"
	"github-badges-backend/internal/user"
)

// Canvas dimensions match the standard Open Graph image ratio.
const (
	canvasW = 1200
	canvasH = 630
)

// GitHub dark mode colour palette.
var (
	colBG      = hexRGB("#0D1117")
	colSurface = hexRGB("#161B22")
	colBorder  = hexRGB("#30363D")
	colText    = hexRGB("#E6EDF3")
	colMuted   = hexRGB("#7D8590")
	colGreen   = hexRGB("#3FB950")
	colBlue    = hexRGB("#58A6FF")
	colAmber   = hexRGB("#D29922")
	colRed     = hexRGB("#F85149")
)

// Generator renders a PNG poster for a user's monthly GitHub activity.
type Generator struct {
	boldSM  font.Face // ~20px — card labels, footer
	boldMD  font.Face // ~32px — profile name, header label
	boldLG  font.Face // ~52px — stat numbers
	boldXL  font.Face // ~64px — month header
	regSM   font.Face // ~20px — profile username
	httpCli *http.Client
}

// NewGenerator initialises the Generator, loading embedded Go fonts.
func NewGenerator() (*Generator, error) {
	bf, err := truetype.Parse(gobold.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing bold font: %w", err)
	}
	rf, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing regular font: %w", err)
	}

	face := func(f *truetype.Font, size float64) font.Face {
		return truetype.NewFace(f, &truetype.Options{
			Size:    size,
			DPI:     144,
			Hinting: font.HintingFull,
		})
	}

	return &Generator{
		boldSM:  face(bf, 7),
		boldMD:  face(bf, 11),
		boldLG:  face(bf, 22),
		boldXL:  face(bf, 28),
		regSM:   face(rf, 9),
		httpCli: &http.Client{Timeout: 5 * time.Second},
	}, nil
}

// Generate renders the PNG poster and returns the raw bytes.
func (g *Generator) Generate(u *user.User, st *stats.MonthlyStats) ([]byte, error) {
	dc := gg.NewContext(canvasW, canvasH)

	// ── Background ──────────────────────────────────────────────────────────
	dc.SetColor(colBG)
	dc.Clear()

	// Top accent bar
	dc.SetColor(colGreen)
	dc.DrawRectangle(0, 0, canvasW, 8)
	dc.Fill()

	// ── Header ──────────────────────────────────────────────────────────────
	dc.SetFontFace(g.boldSM)
	setColor(dc, colGreen)
	dc.DrawString("GITHUB ACTIVITY REPORT", 60, 50)

	dc.SetFontFace(g.boldXL)
	setColor(dc, colText)
	dc.DrawString(st.StatMonth.Format("January 2006"), 60, 110)

	// Right-side username
	dc.SetFontFace(g.boldMD)
	setColor(dc, colMuted)
	dc.DrawStringAnchored("@"+u.GithubLogin, canvasW-60, 50, 1, 0.5)

	dc.SetFontFace(g.regSM)
	setColor(dc, colMuted)
	dc.DrawStringAnchored("Powered by GitHub Badges", canvasW-60, 75, 1, 0.5)

	// Divider
	setColor(dc, colBorder)
	dc.SetLineWidth(1)
	dc.DrawLine(60, 140, canvasW-60, 140)
	dc.Stroke()

	// ── Profile ──────────────────────────────────────────────────────────────
	const (
		avatarCX = 105
		avatarCY = 205
		avatarR  = 48
	)

	// Green glow border
	dc.SetColor(colGreen)
	dc.DrawCircle(avatarCX, avatarCY, avatarR+4)
	dc.Fill()

	// Avatar image (or fallback initials)
	avatar, err := g.downloadAvatar(u.AvatarURL, avatarR*2)
	if err == nil {
		dc.Push()
		dc.DrawCircle(avatarCX, avatarCY, avatarR)
		dc.Clip()
		dc.DrawImageAnchored(avatar, avatarCX, avatarCY, 0.5, 0.5)
		dc.Pop()
	} else {
		// Fallback: filled circle + initials
		dc.SetColor(colSurface)
		dc.DrawCircle(avatarCX, avatarCY, avatarR)
		dc.Fill()
		dc.SetFontFace(g.boldLG)
		setColor(dc, colText)
		dc.DrawStringAnchored(initials(u.Name, u.GithubLogin), avatarCX, avatarCY, 0.5, 0.5)
	}

	// Name + login
	dc.SetFontFace(g.boldMD)
	setColor(dc, colText)
	displayName := u.Name
	if displayName == "" {
		displayName = u.GithubLogin
	}
	dc.DrawString(displayName, 185, 185)

	dc.SetFontFace(g.regSM)
	setColor(dc, colMuted)
	dc.DrawString("@"+u.GithubLogin, 185, 215)

	// ── Stat Cards ───────────────────────────────────────────────────────────
	const (
		cardY = 265
		cardH = 185
		cardW = 255
		gap   = 20
	)

	type cardData struct {
		label    string
		value    string
		subtitle string
		accent   color.Color
	}

	pctStr, pctPositive := formatPct(st.CommitPctChange)
	pctAccent := colGreen
	if !pctPositive {
		pctAccent = colRed
	}

	cards := []cardData{
		{
			label:    "COMMITS",
			value:    formatComma(st.TotalCommits),
			subtitle: "this month",
			accent:   colBlue,
		},
		{
			label:    "NEW REPOS",
			value:    formatComma(st.ReposCreated),
			subtitle: "created this month",
			accent:   colGreen,
		},
		{
			label:    "OSS CONTRIBUTIONS",
			value:    formatComma(st.OpenSourceContributions),
			subtitle: "PRs to other projects",
			accent:   colAmber,
		},
		{
			label:    "COMMIT GROWTH",
			value:    pctStr,
			subtitle: "vs. previous month",
			accent:   pctAccent,
		},
	}

	startX := 60.0
	for i, card := range cards {
		cx := startX + float64(i)*(cardW+gap)
		g.drawCard(dc, cx, cardY, cardW, cardH, card.label, card.value, card.subtitle, card.accent)
	}

	// ── Footer ───────────────────────────────────────────────────────────────
	setColor(dc, colBorder)
	dc.SetLineWidth(1)
	dc.DrawLine(60, 480, canvasW-60, 480)
	dc.Stroke()

	dc.SetFontFace(g.boldSM)
	setColor(dc, colMuted)
	dc.DrawString("github-badges · Monthly Activity Report", 60, 510)
	dc.DrawStringAnchored(
		fmt.Sprintf("Generated %s", time.Now().UTC().Format("2 Jan 2006")),
		canvasW-60, 510, 1, 0.5,
	)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("encoding poster PNG: %w", err)
	}
	return buf.Bytes(), nil
}

// drawCard renders a single stat card onto the drawing context.
func (g *Generator) drawCard(dc *gg.Context, x, y, w, h float64, label, value, subtitle string, accent color.Color) {
	// Border
	dc.SetColor(colBorder)
	dc.DrawRoundedRectangle(x-1, y-1, w+2, h+2, 13)
	dc.Fill()

	// Background
	dc.SetColor(colSurface)
	dc.DrawRoundedRectangle(x, y, w, h, 12)
	dc.Fill()

	// Top accent stripe (clip to card shape first)
	dc.Push()
	dc.DrawRoundedRectangle(x, y, w, h, 12)
	dc.Clip()
	dc.SetColor(accent)
	dc.DrawRectangle(x, y, w, 4)
	dc.Fill()
	dc.Pop()

	// Label
	dc.SetFontFace(g.boldSM)
	setColor(dc, colMuted)
	dc.DrawString(label, x+16, y+32)

	// Value (large, centered, accent-coloured)
	dc.SetFontFace(g.boldLG)
	dc.SetColor(accent)
	dc.DrawStringAnchored(value, x+w/2, y+105, 0.5, 0.5)

	// Subtitle
	dc.SetFontFace(g.regSM)
	setColor(dc, colMuted)
	dc.DrawStringAnchored(subtitle, x+w/2, y+152, 0.5, 0.5)
}

// downloadAvatar fetches the GitHub avatar URL and scales it to size×size pixels.
func (g *Generator) downloadAvatar(avatarURL string, size int) (image.Image, error) {
	resp, err := g.httpCli.Get(avatarURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	src, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func setColor(dc *gg.Context, c color.Color) {
	dc.SetColor(c)
}

func hexRGB(hex string) color.RGBA {
	hex = hex[1:] // strip '#'
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	gr, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return color.RGBA{R: uint8(r), G: uint8(gr), B: uint8(b), A: 255}
}

func formatComma(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.Itoa(n)
	var out []byte
	for i, ch := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, ch)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

func formatPct(v sql.NullFloat64) (string, bool) {
	if !v.Valid {
		return "First Month", true
	}
	val := v.Float64
	if val >= 0 {
		return fmt.Sprintf("+%.1f%%", val), true
	}
	return fmt.Sprintf("%.1f%%", val), false
}

func initials(name, login string) string {
	src := name
	if src == "" {
		src = login
	}
	parts := []rune(src)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		return string(parts[0:1])
	}
	// Take first char of each word (max 2)
	runes := []rune{}
	inWord := false
	for _, r := range parts {
		if r == ' ' {
			inWord = false
			continue
		}
		if !inWord {
			runes = append(runes, r)
			inWord = true
		}
		if len(runes) == 2 {
			break
		}
	}
	return string(runes)
}
