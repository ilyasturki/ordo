package attachments

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))))
	return buf.Bytes()
}

func TestSniff_NamesTheFormatsOrdoAttaches(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var jpg, gf bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpg, img, nil))
	require.NoError(t, gif.Encode(&gf, img, nil))
	webp := []byte("RIFF\x24\x00\x00\x00WEBPVP8 \x18\x00\x00\x00")

	assert.Equal(t, "png", Sniff(pngBytes(t, 2, 2)))
	assert.Equal(t, "jpg", Sniff(jpg.Bytes()))
	assert.Equal(t, "gif", Sniff(gf.Bytes()))
	assert.Equal(t, "webp", Sniff(webp))
	assert.Empty(t, Sniff([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)), "no svg")
	assert.Empty(t, Sniff([]byte("BM\x00\x00\x00\x00")), "no bmp")
	assert.Empty(t, Sniff([]byte("plain text")))
}

func TestAdd_NamesTheFileByItsContent(t *testing.T) {
	projects := t.TempDir()
	s := NewStore(projects)
	data := pngBytes(t, 3, 3)
	sum := sha256.Sum256(data)

	target, err := s.Add(data)

	require.NoError(t, err)
	assert.Equal(t, "attachments/"+hex.EncodeToString(sum[:16])+".png", target)
	got, err := os.ReadFile(filepath.Join(projects, target))
	require.NoError(t, err)
	assert.Equal(t, data, got)
	assert.True(t, Internal(target))
}

func TestAdd_NeverRewritesAFileButMakesItLookNew(t *testing.T) {
	s := NewStore(t.TempDir())
	data := pngBytes(t, 3, 3)
	target, err := s.Add(data)
	require.NoError(t, err)
	path, _ := s.Path(target)
	old := time.Now().Add(-40 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(path, old, old))
	before, err := os.Stat(path)
	require.NoError(t, err)

	again, err := s.Add(data)

	require.NoError(t, err)
	assert.Equal(t, target, again)
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, os.SameFile(before, after), "a stored file is written once")
	assert.WithinDuration(t, time.Now(), after.ModTime(), time.Minute,
		"attached again, it must not look old enough to prune")
}

func TestAdd_LeavesNoTempFileBehind(t *testing.T) {
	s := NewStore(t.TempDir())
	_, err := s.Add(pngBytes(t, 3, 3))
	require.NoError(t, err)
	_, err = s.Add(pngBytes(t, 5, 5))
	require.NoError(t, err)

	entries, err := os.ReadDir(s.Dir())
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Regexp(t, `^[0-9a-f]{32}\.png$`, e.Name())
		info, err := e.Info()
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
	}
}

func TestAdd_RefusesWhatItCannotAttach(t *testing.T) {
	s := NewStore(t.TempDir())

	_, err := s.Add([]byte("not an image"))
	assert.ErrorIs(t, err, ErrNotImage)

	big := append(pngBytes(t, 2, 2), make([]byte, MaxSize)...)
	_, err = s.Add(big)
	assert.ErrorIs(t, err, ErrTooLarge)

	_, err = os.Stat(s.Dir())
	assert.True(t, os.IsNotExist(err), "a refused image creates nothing")
}

func TestReadImage(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "shot.png")
	require.NoError(t, os.WriteFile(img, pngBytes(t, 2, 2), 0o600))
	text := filepath.Join(dir, "notes.png")
	require.NoError(t, os.WriteFile(text, []byte("hello"), 0o600))
	big := filepath.Join(dir, "big.png")
	require.NoError(t, os.WriteFile(big, append(pngBytes(t, 2, 2), make([]byte, MaxSize)...), 0o600))

	data, err := ReadImage(img)
	require.NoError(t, err)
	assert.Equal(t, "png", Sniff(data))
	_, err = ReadImage(text)
	assert.ErrorIs(t, err, ErrNotImage, "the extension is not the format")
	_, err = ReadImage(big)
	assert.ErrorIs(t, err, ErrTooLarge)
	_, err = ReadImage(dir)
	assert.ErrorIs(t, err, ErrNotImage)
	_, err = ReadImage(filepath.Join(dir, "gone.png"))
	assert.Error(t, err)

	assert.True(t, IsImageFile(img))
	assert.False(t, IsImageFile(text))
	assert.False(t, IsImageFile(dir))
	assert.False(t, IsImageFile(filepath.Join(dir, "gone.png")))
}

func TestInternal_AcceptsOnlyTheStoresOwnNames(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef"
	for target, want := range map[string]bool{
		"attachments/" + hash + ".png":                     true,
		"attachments/" + hash + ".jpg":                     true,
		"attachments/" + hash + ".gif":                     true,
		"attachments/" + hash + ".webp":                    true,
		"attachments/" + hash + ".svg":                     false,
		"attachments/" + hash + ".jpeg":                    false,
		"attachments/0123456789ABCDEF0123456789abcdef.png": false,
		"attachments/" + hash[:31] + ".png":                false,
		"attachments/../" + hash + ".png":                  false,
		"attachments/../../etc/passwd":                     false,
		"attachments/x/" + hash + ".png":                   false,
		"./attachments/" + hash + ".png":                   false,
		"/abs/attachments/" + hash + ".png":                false,
	} {
		assert.Equal(t, want, Internal(target), target)
	}
}

func TestPath(t *testing.T) {
	projects := t.TempDir()
	s := NewStore(projects)
	hash := "0123456789abcdef0123456789abcdef"

	path, ok := s.Path("attachments/" + hash + ".png")
	assert.True(t, ok)
	assert.Equal(t, filepath.Join(projects, "attachments", hash+".png"), path)

	path, ok = s.Path("/home/me/Pictures/shot.png")
	assert.True(t, ok, "an absolute path is an external image")
	assert.Equal(t, "/home/me/Pictures/shot.png", path)

	_, ok = s.Path("attachments/../../../etc/passwd")
	assert.False(t, ok)
	_, ok = s.Path("pictures/shot.png")
	assert.False(t, ok)
}

func TestMissing(t *testing.T) {
	s := NewStore(t.TempDir())
	target, err := s.Add(pngBytes(t, 2, 2))
	require.NoError(t, err)

	assert.False(t, s.Missing(target))
	assert.True(t, s.Missing("attachments/0123456789abcdef0123456789abcdef.png"), "not synced yet")
	assert.True(t, s.Missing("/nowhere/shot.png"))
	assert.True(t, s.Missing("relative.png"))
	assert.True(t, s.Missing(filepath.Dir(s.Dir())), "a directory is no image")
}

func TestRefs(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef"
	text := "the overflow here:\n![Screenshot 10.10](attachments/" + hash + ".png)\n" +
		"inline ![ old shot ](/home/me/old.png) too\n" +
		"![web](https://example.com/a.png) ![rel](pics/a.png) ![bad](attachments/../x.png)\n" +
		"![](attachments/" + hash + ".gif) [not an image](attachments/" + hash + ".png)"

	assert.Equal(t, []Ref{
		{Name: "Screenshot 10.10", Target: "attachments/" + hash + ".png"},
		{Name: "old shot", Target: "/home/me/old.png"},
		{Name: "", Target: "attachments/" + hash + ".gif"},
	}, Refs(text))
	assert.Nil(t, Refs("no images here"))
	assert.Equal(t, []Ref{{Name: "first", Target: "/a.png"}}, Refs("![first](/a.png) ![second](/a.png)"),
		"each file once, under its first name")
}

func TestSplit(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef"
	ref := Ref{Name: "shot", Target: "attachments/" + hash + ".png"}
	line := "see ![shot](" + ref.Target + ") and ![web](https://x/y.png)"

	assert.Equal(t, []Piece{
		{Text: "see "},
		{Ref: &ref},
		{Text: " and ![web](https://x/y.png)"},
	}, Split(line))
	assert.Equal(t, []Piece{{Ref: &ref}}, Split("![shot]("+ref.Target+")"))
	assert.Equal(t, []Piece{{Text: ""}}, Split(""))
	assert.Equal(t, []Piece{{Text: "plain"}}, Split("plain"))
}

func TestDisplay_ShowsTheNameNotTheHash(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef"
	text := "see ![Screenshot](attachments/" + hash + ".png)\n![](/tmp/a.png) ![web](https://x/y.png)"

	assert.Equal(t, "see ▣ Screenshot\n▣ image ![web](https://x/y.png)", Display(text))
}

func TestDescribe(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, data, 0o600))
		return path
	}
	img := image.NewRGBA(image.Rect(0, 0, 7, 3))
	var jpg, gf bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpg, img, nil))
	require.NoError(t, gif.Encode(&gf, img, nil))
	riff := func(chunk string, body ...byte) []byte {
		head := []byte("RIFF\x00\x00\x00\x00WEBP" + chunk + "\x00\x00\x00\x00")
		return append(head, body...)
	}
	vp8l := uint32(1280-1) | uint32(640-1)<<14
	for name, tc := range map[string]struct {
		data []byte
		w, h int
	}{
		"png":  {pngBytes(t, 7, 3), 7, 3},
		"jpeg": {jpg.Bytes(), 7, 3},
		"gif":  {gf.Bytes(), 7, 3},
		"webp VP8X": {riff("VP8X", 0, 0, 0, 0,
			0xff, 0x04, 0x00, 0x7f, 0x02, 0x00), 1280, 640},
		"webp VP8L": {riff("VP8L", 0x2f,
			byte(vp8l), byte(vp8l>>8), byte(vp8l>>16), byte(vp8l>>24), 0, 0, 0, 0, 0), 1280, 640},
		"webp VP8": {riff("VP8 ", 0, 0, 0, 0x9d, 0x01, 0x2a,
			0x00, 0x05, 0x80, 0x02), 1280, 640},
		"not an image": {[]byte("hello"), 0, 0},
	} {
		t.Run(name, func(t *testing.T) {
			info, err := Describe(write(name, tc.data))
			require.NoError(t, err)
			assert.Equal(t, Info{Size: int64(len(tc.data)), Width: tc.w, Height: tc.h}, info)
		})
	}
	_, err := Describe(filepath.Join(dir, "gone.png"))
	assert.Error(t, err)
}

func TestStrip_LeavesNoHashBehind(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef"
	assert.Equal(t, "see   and ![web](https://x/y.png)", Strip("see ![Shot 10.10](attachments/"+hash+".png) and ![web](https://x/y.png)"))
}

func TestLine(t *testing.T) {
	assert.Equal(t, "![Screenshot 2026-10-11 10.10](attachments/x.png)", Line("Screenshot 2026-10-11 10.10", "attachments/x.png"))
	assert.Equal(t, "![a b c](t)", Line(" a [b]\nc ", "t"), "brackets and newlines would end the reference")
	ref := Refs(Line("weird ] name", "/tmp/a.png"))
	require.Len(t, ref, 1)
	assert.Equal(t, "weird name", ref[0].Name)
}
