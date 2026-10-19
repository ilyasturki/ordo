package web

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ilyasturki/ordo/internal/fsutil"
)

const (
	PairTTL      = 10 * time.Minute
	DeviceIdle   = 365 * 24 * time.Hour
	renewEvery   = 24 * time.Hour
	pbkdf2Rounds = 600_000
)

var (
	ErrBadPairing    = errors.New("this pairing link is used or expired")
	ErrBadPassphrase = errors.New("wrong passphrase")
	ErrNoPassphrase  = errors.New("no passphrase is set")
)

type Device struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
	PairedBy  string    `json:"paired_by,omitempty"`
}

type pairing struct {
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	By        string    `json:"by,omitempty"`
}

type passphrase struct {
	Salt string `json:"salt"`
	Hash string `json:"hash"`
}

type authFile struct {
	BaseURL    string      `json:"base_url,omitempty"`
	Passphrase *passphrase `json:"passphrase,omitempty"`
	Devices    []Device    `json:"devices"`
	Pairings   []pairing   `json:"pairings,omitempty"`
}

// Auth keeps paired devices in a file beside state.json, outside the synced projects directory.
type Auth struct {
	path string
	now  func() time.Time
}

func NewAuth(stateDir string) *Auth {
	return &Auth{path: filepath.Join(stateDir, "web.json"), now: time.Now}
}

func (a *Auth) Path() string { return a.path }

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *Auth) read() (authFile, error) {
	var f authFile
	raw, err := os.ReadFile(a.path)
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return f, err
	}
	return f, json.Unmarshal(raw, &f)
}

// update runs fn on the file under its lock and writes it back when fn reports a change.
func (a *Auth) update(fn func(*authFile) (bool, error)) error {
	if err := os.MkdirAll(filepath.Dir(a.path), 0o700); err != nil {
		return err
	}
	lock, err := fsutil.Lock(a.path+".lock", 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	f, err := a.read()
	if err != nil {
		return err
	}
	now := a.now()
	before := len(f.Pairings) + len(f.Devices)
	f.Pairings = slices.DeleteFunc(f.Pairings, func(p pairing) bool { return !now.Before(p.ExpiresAt) })
	f.Devices = slices.DeleteFunc(f.Devices, func(d Device) bool { return now.Sub(d.LastSeen) > DeviceIdle })
	changed, err := fn(&f)
	if err != nil {
		return err
	}
	if !changed && before == len(f.Pairings)+len(f.Devices) {
		return nil
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteAtomic(a.path, raw, 0o600)
}

// NewPairing mints a one-time token that pairs one device within PairTTL.
func (a *Auth) NewPairing() (string, error) {
	return a.NewPairingBy("")
}

// NewPairingBy mints a token on behalf of device by, replacing the one it minted before.
func (a *Auth) NewPairingBy(by string) (string, error) {
	token := rand.Text()
	err := a.update(func(f *authFile) (bool, error) {
		if by != "" {
			f.Pairings = slices.DeleteFunc(f.Pairings, func(p pairing) bool { return p.By == by })
		}
		f.Pairings = append(f.Pairings, pairing{TokenHash: hashToken(token), ExpiresAt: a.now().Add(PairTTL), By: by})
		return true, nil
	})
	return token, err
}

// pairToken reduces a pasted pairing link or code to the token it carries.
func pairToken(s string) string {
	s = strings.Join(strings.Fields(s), "")
	if _, rest, ok := strings.Cut(s, "/pair/"); ok {
		s = rest
		if i := strings.IndexAny(s, "/?#"); i >= 0 {
			s = s[:i]
		}
	}
	return strings.ToUpper(strings.ReplaceAll(s, "-", ""))
}

// Pair spends a pairing token, or a link carrying one, on a new device and returns its cookie value.
func (a *Auth) Pair(token, name string) (string, error) {
	var cookie string
	err := a.update(func(f *authFile) (bool, error) {
		h := hashToken(pairToken(token))
		i := slices.IndexFunc(f.Pairings, func(p pairing) bool {
			return subtle.ConstantTimeCompare([]byte(p.TokenHash), []byte(h)) == 1
		})
		if i < 0 {
			return false, ErrBadPairing
		}
		by := f.Pairings[i].By
		f.Pairings = slices.Delete(f.Pairings, i, i+1)
		cookie = a.addDevice(f, name, by)
		return true, nil
	})
	return cookie, err
}

func checkPassphrase(f *authFile, phrase string) error {
	if f.Passphrase == nil {
		return ErrNoPassphrase
	}
	salt, _ := hex.DecodeString(f.Passphrase.Salt)
	key, err := pbkdf2.Key(sha256.New, phrase, salt, pbkdf2Rounds, 32)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(key)), []byte(f.Passphrase.Hash)) != 1 {
		return ErrBadPassphrase
	}
	return nil
}

// CheckPassphrase verifies the passphrase without pairing anything.
func (a *Auth) CheckPassphrase(phrase string) error {
	f, err := a.read()
	if err != nil {
		return err
	}
	return checkPassphrase(&f, phrase)
}

// Login checks the passphrase and pairs a new device on a match.
func (a *Auth) Login(phrase, name string) (string, error) {
	var cookie string
	err := a.update(func(f *authFile) (bool, error) {
		if err := checkPassphrase(f, phrase); err != nil {
			return false, err
		}
		cookie = a.addDevice(f, name, "")
		return true, nil
	})
	return cookie, err
}

func (a *Auth) addDevice(f *authFile, name, pairedBy string) string {
	id, token := rand.Text()[:10], rand.Text()
	now := a.now()
	f.Devices = append(f.Devices, Device{ID: id, Name: name, TokenHash: hashToken(token), CreatedAt: now, LastSeen: now, PairedBy: pairedBy})
	return id + "." + token
}

// Check resolves a cookie to its device; renew reports that the cookie is due to be sent again.
func (a *Auth) Check(cookie string) (dev Device, renew bool, ok bool) {
	id, token, found := strings.Cut(cookie, ".")
	if !found {
		return Device{}, false, false
	}
	f, err := a.read()
	if err != nil {
		return Device{}, false, false
	}
	i := slices.IndexFunc(f.Devices, func(d Device) bool { return d.ID == id })
	if i < 0 || subtle.ConstantTimeCompare([]byte(f.Devices[i].TokenHash), []byte(hashToken(token))) != 1 {
		return Device{}, false, false
	}
	dev = f.Devices[i]
	now := a.now()
	if now.Sub(dev.LastSeen) > DeviceIdle {
		return Device{}, false, false
	}
	if now.Sub(dev.LastSeen) < renewEvery {
		return dev, false, true
	}
	err = a.update(func(f *authFile) (bool, error) {
		for i := range f.Devices {
			if f.Devices[i].ID == id {
				f.Devices[i].LastSeen = now
				return true, nil
			}
		}
		return false, nil
	})
	return dev, err == nil, true
}

func (a *Auth) Devices() ([]Device, error) {
	f, err := a.read()
	return f.Devices, err
}

// Revoke signs out a device and every device it paired, theirs included, and returns them all.
func (a *Auth) Revoke(id string) ([]Device, error) {
	var gone []Device
	err := a.update(func(f *authFile) (bool, error) {
		gone = lineage(f.Devices, id)
		if len(gone) == 0 {
			return false, nil
		}
		ids := make(map[string]bool, len(gone))
		for _, d := range gone {
			ids[d.ID] = true
		}
		f.Devices = slices.DeleteFunc(f.Devices, func(d Device) bool { return ids[d.ID] })
		f.Pairings = slices.DeleteFunc(f.Pairings, func(p pairing) bool { return ids[p.By] })
		return true, nil
	})
	return gone, err
}

// lineage lists device id first, then the devices it paired, transitively; nil when id is unknown.
func lineage(devs []Device, id string) []Device {
	i := slices.IndexFunc(devs, func(d Device) bool { return d.ID == id })
	if i < 0 {
		return nil
	}
	out := []Device{devs[i]}
	seen := map[string]bool{id: true}
	for j := 0; j < len(out); j++ {
		for _, d := range devs {
			if d.PairedBy == out[j].ID && !seen[d.ID] {
				seen[d.ID] = true
				out = append(out, d)
			}
		}
	}
	return out
}

func (a *Auth) SetPassphrase(phrase string) error {
	salt := make([]byte, 16)
	rand.Read(salt)
	key, err := pbkdf2.Key(sha256.New, phrase, salt, pbkdf2Rounds, 32)
	if err != nil {
		return err
	}
	return a.update(func(f *authFile) (bool, error) {
		f.Passphrase = &passphrase{Salt: hex.EncodeToString(salt), Hash: hex.EncodeToString(key)}
		return true, nil
	})
}

func (a *Auth) HasPassphrase() bool {
	f, err := a.read()
	return err == nil && f.Passphrase != nil
}

func (a *Auth) BaseURL() string {
	f, _ := a.read()
	return f.BaseURL
}

func (a *Auth) SetBaseURL(url string) error {
	return a.update(func(f *authFile) (bool, error) {
		if f.BaseURL == url {
			return false, nil
		}
		f.BaseURL = url
		return true, nil
	})
}
