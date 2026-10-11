// Package attachments keeps the images tasks carry: files in the projects
// directory's attachments/ named by their content, and the ![name](target)
// references descriptions point at them with.
package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/ilyasturki/ordo/internal/fsutil"
)

const DirName = "attachments"

const (
	WarnSize = 5 << 20
	MaxSize  = 25 << 20
)

const Glyph = "▣"

var (
	ErrNotImage = errors.New("only png, jpeg, gif and webp images can be attached")
	ErrTooLarge = fmt.Errorf("images over %d MB can't be attached", MaxSize>>20)
)

var extensions = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/gif":  "gif",
	"image/webp": "webp",
}

// fileName is a stored file: 32 hex digits of its sha256, then its format.
var fileName = regexp.MustCompile(`^[0-9a-f]{32}\.(?:png|jpg|gif|webp)$`)

func Sniff(data []byte) string {
	return extensions[http.DetectContentType(data)]
}

// Store holds the images, written once and never modified.
type Store struct {
	dir string
}

func NewStore(projectsDir string) *Store {
	dir := filepath.Join(projectsDir, DirName)
	// Absolute so a path copied or opened from it still resolves elsewhere.
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	return &Store{dir: dir}
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) Add(data []byte) (string, error) {
	if len(data) > MaxSize {
		return "", ErrTooLarge
	}
	ext := Sniff(data)
	if ext == "" {
		return "", ErrNotImage
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:16]) + "." + ext
	target := DirName + "/" + name
	path := filepath.Join(s.dir, name)
	if _, err := os.Lstat(path); err == nil {
		// Prune spares files by age: one referenced again must look new.
		now := time.Now()
		_ = os.Chtimes(path, now, now)
		return target, nil
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}
	if err := fsutil.WriteAtomicUnique(path, data, 0o644); err != nil {
		return "", err
	}
	return target, nil
}

// Internal reports whether target names a file in the store. Nothing else
// resolves into it, so a reference can't reach outside the directory.
func Internal(target string) bool {
	name, ok := strings.CutPrefix(target, DirName+"/")
	return ok && fileName.MatchString(name)
}

// Path is the file target points at: in the store for an internal target, the
// target itself for an absolute path; ok is false for any other target.
func (s *Store) Path(target string) (path string, ok bool) {
	if Internal(target) {
		return filepath.Join(s.dir, strings.TrimPrefix(target, DirName+"/")), true
	}
	if filepath.IsAbs(target) {
		return target, true
	}
	return "", false
}

func (s *Store) Missing(target string) bool {
	path, ok := s.Path(target)
	if !ok {
		return true
	}
	info, err := os.Stat(path)
	return err != nil || !info.Mode().IsRegular()
}

func ReadImage(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, ErrNotImage
	}
	if info.Size() > MaxSize {
		return nil, ErrTooLarge
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if Sniff(data) == "" {
		return nil, ErrNotImage
	}
	return data, nil
}

func IsImageFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false
	}
	return Sniff(head[:n]) != ""
}

func FormatSize(n int64) string {
	switch {
	case n < 1<<10:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%d KB", n>>10)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
}

type Info struct {
	Size int64
	// Zero when the header can't be read.
	Width, Height int
}

func Describe(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Info{}, err
	}
	info := Info{Size: st.Size()}
	head := make([]byte, 30)
	n, _ := io.ReadFull(f, head)
	if w, h, ok := webpSize(head[:n]); ok {
		info.Width, info.Height = w, h
		return info, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return info, nil
	}
	if cfg, _, err := image.DecodeConfig(f); err == nil {
		info.Width, info.Height = cfg.Width, cfg.Height
	}
	return info, nil
}

// webpSize reads the canvas size from the first chunk's header: VP8X, VP8L
// or VP8, each laid out differently.
func webpSize(b []byte) (int, int, bool) {
	if len(b) < 30 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return 0, 0, false
	}
	le24 := func(p []byte) int { return int(p[0]) | int(p[1])<<8 | int(p[2])<<16 }
	switch string(b[12:16]) {
	case "VP8X":
		return le24(b[24:27]) + 1, le24(b[27:30]) + 1, true
	case "VP8L":
		bits := uint32(b[21]) | uint32(b[22])<<8 | uint32(b[23])<<16 | uint32(b[24])<<24
		return int(bits&0x3fff) + 1, int(bits>>14&0x3fff) + 1, true
	case "VP8 ":
		return (int(b[26]) | int(b[27])<<8) & 0x3fff, (int(b[28]) | int(b[29])<<8) & 0x3fff, true
	}
	return 0, 0, false
}

type Ref struct {
	Name   string
	Target string
}

func (r Ref) Internal() bool {
	return Internal(r.Target)
}

func (r Ref) Label() string {
	name := r.Name
	if name == "" {
		name = "image"
	}
	return Glyph + " " + name
}

var refPattern = regexp.MustCompile(`!\[([^\[\]\n]*)\]\(([^()\n]+)\)`)

func parseRef(match []string) (Ref, bool) {
	ref := Ref{Name: strings.TrimSpace(match[1]), Target: strings.TrimSpace(match[2])}
	return ref, ref.Internal() || filepath.IsAbs(ref.Target)
}

func Refs(text string) []Ref {
	if !strings.Contains(text, "![") {
		return nil
	}
	var refs []Ref
	for _, m := range refPattern.FindAllStringSubmatch(text, -1) {
		ref, ok := parseRef(m)
		if ok && !slices.ContainsFunc(refs, func(r Ref) bool { return r.Target == ref.Target }) {
			refs = append(refs, ref)
		}
	}
	return refs
}

type Piece struct {
	Text string
	Ref  *Ref
}

func Split(line string) []Piece {
	if !strings.Contains(line, "![") {
		return []Piece{{Text: line}}
	}
	var pieces []Piece
	last := 0
	for _, loc := range refPattern.FindAllStringSubmatchIndex(line, -1) {
		ref, ok := parseRef([]string{
			line[loc[0]:loc[1]], line[loc[2]:loc[3]], line[loc[4]:loc[5]],
		})
		if !ok {
			continue
		}
		if loc[0] > last {
			pieces = append(pieces, Piece{Text: line[last:loc[0]]})
		}
		pieces = append(pieces, Piece{Ref: &ref})
		last = loc[1]
	}
	if last < len(line) || len(pieces) == 0 {
		pieces = append(pieces, Piece{Text: line[last:]})
	}
	return pieces
}

// Display is text as it reads on screen, each reference turned into its label.
func Display(text string) string {
	return replaceRefs(text, Ref.Label)
}

// Strip blanks out every reference, hashes included.
func Strip(text string) string {
	return replaceRefs(text, func(Ref) string { return " " })
}

func replaceRefs(text string, with func(Ref) string) string {
	if !strings.Contains(text, "![") {
		return text
	}
	return refPattern.ReplaceAllStringFunc(text, func(match string) string {
		ref, ok := parseRef(refPattern.FindStringSubmatch(match))
		if !ok {
			return match
		}
		return with(ref)
	})
}

func Line(name, target string) string {
	name = strings.Map(func(r rune) rune {
		if r == '[' || r == ']' || unicode.IsControl(r) {
			return ' '
		}
		return r
	}, name)
	return "![" + strings.Join(strings.Fields(name), " ") + "](" + target + ")"
}
