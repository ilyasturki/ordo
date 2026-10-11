// Package attachments keeps the images tasks carry: files in the projects
// directory's attachments/ named by their content, and the ![name](target)
// references descriptions point at them with.
package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	return &Store{dir: filepath.Join(projectsDir, DirName)}
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

// Ref is an image a description references as ![Name](Target).
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

// parseRef reads one refPattern match; ok is false for a target that is
// neither internal nor an absolute path.
func parseRef(match []string) (Ref, bool) {
	ref := Ref{Name: strings.TrimSpace(match[1]), Target: strings.TrimSpace(match[2])}
	return ref, ref.Internal() || filepath.IsAbs(ref.Target)
}

func Refs(text string) []Ref {
	var refs []Ref
	for _, m := range refPattern.FindAllStringSubmatch(text, -1) {
		if ref, ok := parseRef(m); ok {
			refs = append(refs, ref)
		}
	}
	return refs
}

// Piece is a run of a line: text, or an image reference when Ref is set.
type Piece struct {
	Text string
	Ref  *Ref
}

func Split(line string) []Piece {
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
	return refPattern.ReplaceAllStringFunc(text, func(match string) string {
		ref, ok := parseRef(refPattern.FindStringSubmatch(match))
		if !ok {
			return match
		}
		return ref.Label()
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
