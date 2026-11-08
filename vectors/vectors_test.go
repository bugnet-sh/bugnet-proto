package vectors_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/bugnet-sh/bugnet-proto/vectors"
)

type pathsFile struct {
	V       *int        `json:"v"`
	Paths   []pathCase  `json:"paths"`
	Queries []queryCase `json:"queries"`
}

type pathCase struct {
	Path *string `json:"path"`
	OK   *bool   `json:"ok"`
}

type queryCase struct {
	Query *string `json:"query"`
	OK    *bool   `json:"ok"`
}

type serviceIDsFile struct {
	V     *int            `json:"v"`
	Cases []serviceIDCase `json:"cases"`
}

type serviceIDCase struct {
	Method string `json:"method"`
	Host   string `json:"host"`
	Prefix string `json:"prefix"`
	ID     string `json:"id"`
}

type suffixesFile struct {
	V        *int     `json:"v"`
	Suffixes []string `json:"suffixes"`
}

var (
	idShape     = regexp.MustCompile(`^xs_[a-z2-7]{16}$`)
	prefixShape = regexp.MustCompile(`^/.+/$`)
	hostShape   = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9-]+)+$`)
)

// decode reads one vector file and fails the test when strictDecode refuses it
func decode(t *testing.T, name string, into any) {
	t.Helper()
	raw, err := vectors.X402.ReadFile("x402/" + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if err := strictDecode(raw, into); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
}

// strictDecode refuses bad UTF-8, unknown fields, trailing data and keys not spelled as the tags
func strictDecode(raw []byte, into any) error {
	if !utf8.Valid(raw) {
		return errors.New("not valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("data after the object")
	}
	// encoding/json matches keys in any case, so a round trip must give back the same tree
	var asWritten, asTagged any
	if err := json.Unmarshal(raw, &asWritten); err != nil {
		return err
	}
	tagged, err := json.Marshal(into)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(tagged, &asTagged); err != nil {
		return err
	}
	if !reflect.DeepEqual(asWritten, asTagged) {
		return errors.New("a key is missing or not spelled exactly as the contract")
	}
	return nil
}

func TestStrictDecodeRefusesKeysSpelledOtherwise(t *testing.T) {
	good := `{"v":1,"paths":[{"path":"/search","ok":true}],"queries":[]}`
	if err := strictDecode([]byte(good), new(pathsFile)); err != nil {
		t.Fatalf("exact keys refused: %v", err)
	}
	for _, bad := range []string{
		`{"v":1,"paths":[{"Path":"/search","OK":true}],"queries":[]}`,
		`{"V":1,"paths":[],"queries":[]}`,
		`{"v":1,"Paths":[],"queries":[]}`,
		`{"v":1,"paths":[{"path":"/search"}],"queries":[]}`,
		`{"v":1,"paths":[],"queries":[],"extra":1}`,
		`{"v":1,"paths":[],"queries":[]} {}`,
	} {
		if strictDecode([]byte(bad), new(pathsFile)) == nil {
			t.Errorf("%s was accepted", bad)
		}
	}
	if strictDecode([]byte(`{"v":1,"cases":[{"Method":"GET","host":"a.b","prefix":"/a/",`+
		`"id":"xs_aaaaaaaaaaaaaaaa"}]}`), new(serviceIDsFile)) == nil {
		t.Error("a service case with Method was accepted")
	}
	if strictDecode([]byte(`{"v":1,"Suffixes":["a.b"]}`), new(suffixesFile)) == nil {
		t.Error("a suffix file with Suffixes was accepted")
	}
}

// vector is one path or query case and whether XS 6.7 passes it
type vector struct {
	in string
	ok bool
}

// readPaths decodes paths.json and fails the test on a case that lacks a field
func readPaths(t *testing.T) (paths, queries []vector) {
	t.Helper()
	var f pathsFile
	decode(t, "paths.json", &f)
	versionOne(t, "paths.json", f.V)
	for i, c := range f.Paths {
		if c.Path == nil || c.OK == nil {
			t.Fatalf("paths[%d] lacks path or ok", i)
		}
		paths = append(paths, vector{*c.Path, *c.OK})
	}
	for i, c := range f.Queries {
		if c.Query == nil || c.OK == nil {
			t.Fatalf("queries[%d] lacks query or ok", i)
		}
		queries = append(queries, vector{*c.Query, *c.OK})
	}
	return paths, queries
}

// versionOne fails the test unless v is present and 1
func versionOne(t *testing.T, name string, v *int) {
	t.Helper()
	if v == nil || *v != 1 {
		t.Fatalf("%s has no v of 1", name)
	}
}

// once fails the test when a key appears twice
func once(t *testing.T, name string, keys []string) {
	t.Helper()
	seen := make(map[string]bool, len(keys))
	for _, k := range keys {
		if seen[k] {
			t.Errorf("%s holds %q twice", name, k)
		}
		seen[k] = true
	}
}

// bothOutcomes fails the test unless the cases hold at least one pass and one refusal
func bothOutcomes(t *testing.T, name string, oks []bool) {
	t.Helper()
	var pass, refuse int
	for _, ok := range oks {
		if ok {
			pass++
		} else {
			refuse++
		}
	}
	if pass == 0 || refuse == 0 {
		t.Errorf("%s has %d passed and %d refused cases", name, pass, refuse)
	}
}

func TestVectorsParse(t *testing.T) {
	names, err := fs.Glob(vectors.X402, "x402/*")
	if err != nil {
		t.Fatal(err)
	}
	eq(t, names, []string{
		"x402/paths.json",
		"x402/service-ids.json",
		"x402/shared-hosting-suffixes.json",
	})

	t.Run("paths", func(t *testing.T) {
		paths, queries := readPaths(t)
		for name, cases := range map[string][]vector{"paths": paths, "queries": queries} {
			var ins []string
			var oks []bool
			for _, c := range cases {
				ins = append(ins, c.in)
				oks = append(oks, c.ok)
			}
			once(t, name, ins)
			bothOutcomes(t, name, oks)
		}
	})

	t.Run("service-ids", func(t *testing.T) {
		var f serviceIDsFile
		decode(t, "service-ids.json", &f)
		versionOne(t, "service-ids.json", f.V)
		if len(f.Cases) == 0 {
			t.Fatal("service-ids.json has no cases")
		}
		var inputs, ids []string
		for i, c := range f.Cases {
			if c.Method != "GET" && c.Method != "POST" {
				t.Errorf("cases[%d] method %q is not GET or POST", i, c.Method)
			}
			if !hostShape.MatchString(c.Host) || !prefixShape.MatchString(c.Prefix) {
				t.Errorf("cases[%d] host %q or prefix %q is malformed", i, c.Host, c.Prefix)
			}
			inputs = append(inputs, c.Method+" "+c.Host+c.Prefix)
			ids = append(ids, c.ID)
		}
		once(t, "service id inputs", inputs)
		once(t, "service ids", ids)
	})

	t.Run("shared-hosting-suffixes", func(t *testing.T) {
		var f suffixesFile
		decode(t, "shared-hosting-suffixes.json", &f)
		versionOne(t, "shared-hosting-suffixes.json", f.V)
		if len(f.Suffixes) == 0 {
			t.Fatal("shared-hosting-suffixes.json has no suffixes")
		}
		once(t, "suffixes", f.Suffixes)
	})
}

// serviceID is the reference id of XS 5.1
func serviceID(method, host, prefix string) string {
	sum := sha256.Sum256([]byte(method + "\x00" + host + "\x00" + prefix))
	return "xs_" + strings.ToLower(base32.StdEncoding.EncodeToString(sum[:]))[:16]
}

func TestServiceIDVectors(t *testing.T) {
	known := map[[3]string]string{
		{"POST", "agent402.tools", "/v1/auto/"}: "xs_w5a4htopmpzmmhos",
		{"GET", "api.example.com", "/search/"}:  "xs_vky43ltmauaj3w53",
	}
	for in, id := range known {
		eq(t, serviceID(in[0], in[1], in[2]), id)
	}

	var f serviceIDsFile
	decode(t, "service-ids.json", &f)
	found := 0
	for _, c := range f.Cases {
		if !idShape.MatchString(c.ID) {
			t.Errorf("id %q is not xs_ and 16 base32 characters", c.ID)
		}
		if got := serviceID(c.Method, c.Host, c.Prefix); got != c.ID {
			t.Errorf("%s %s %s gives %s, the file says %s", c.Method, c.Host, c.Prefix, got, c.ID)
		}
		if _, ok := known[[3]string{c.Method, c.Host, c.Prefix}]; ok {
			found++
		}
	}
	eq(t, found, len(known))
}

func TestSuffixesAreTheSixteen(t *testing.T) {
	var f suffixesFile
	decode(t, "shared-hosting-suffixes.json", &f)
	for _, s := range f.Suffixes {
		if !hostShape.MatchString(s) {
			t.Errorf("suffix %q is not a lower-case name", s)
		}
	}
	eq(t, f.Suffixes, []string{
		"workers.dev", "onrender.com", "vercel.app", "netlify.app", "pages.dev",
		"herokuapp.com", "fly.dev", "up.railway.app", "replit.app", "ngrok-free.app",
		"ngrok.io", "trycloudflare.com", "github.io", "glitch.me", "deno.dev", "web.app",
	})
}

// pathOK is the reference path rule of XS 6.7
func pathOK(p string) bool {
	if !strings.HasPrefix(p, "/") || strings.Contains(p, "//") || strings.ContainsAny(p, ";\\") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "." || seg == ".." {
			return false
		}
	}
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c < 0x21 || c > 0x7e {
			return false
		}
		if c != '%' {
			continue
		}
		if i+2 >= len(p) || !isHex(p[i+1]) || !isHex(p[i+2]) {
			return false
		}
		b := unhex(p[i+1])<<4 | unhex(p[i+2])
		if b < 0x20 || b >= 0x7f || strings.IndexByte(`/\.;%`, b) >= 0 {
			return false
		}
		i += 2
	}
	return true
}

// queryOK is the reference query rule of XS 6.7
func queryOK(q string) bool {
	for i := 0; i < len(q); i++ {
		if q[i] < 0x21 || q[i] > 0x7e {
			return false
		}
	}
	return true
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func unhex(c byte) byte {
	switch {
	case c <= '9':
		return c - '0'
	case c <= 'F':
		return c - 'A' + 10
	default:
		return c - 'a' + 10
	}
}

func TestPathVectorsFollowTheRule(t *testing.T) {
	paths, queries := readPaths(t)
	for _, c := range paths {
		if pathOK(c.in) != c.ok {
			t.Errorf("path %q says ok %v, XS 6.7 says %v", c.in, c.ok, !c.ok)
		}
	}
	for _, c := range queries {
		if queryOK(c.in) != c.ok {
			t.Errorf("query %q says ok %v, XS 6.7 says %v", c.in, c.ok, !c.ok)
		}
	}
}
