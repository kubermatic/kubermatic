// +build ignore

// mkstdlib generates the zstdlib.go file, containing the Go standard
// library API symbols. It's baked into the binary to avoid scanning
// GOPATH in the common case.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

func mustOpen(name string) io.Reader {
	f, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	return f
}

func api(base string) string {
	return filepath.Join(runtime.GOROOT(), "api", base)
}

var sym = regexp.MustCompile(`^pkg (\S+).*?, (?:var|func|type|const) ([A-Z]\w*)`)

func main() {
	var buf bytes.Buffer
	outf := func(format string, args ...interface{}) {
		fmt.Fprintf(&buf, format, args...)
	}
	outf("// Code generated by mkstdlib.go. DO NOT EDIT.\n\n")
	outf("package imports\n")
	outf("var stdlib = map[string]string{\n")
	f := io.MultiReader(
		mustOpen(api("go1.txt")),
		mustOpen(api("go1.1.txt")),
		mustOpen(api("go1.2.txt")),
		mustOpen(api("go1.3.txt")),
		mustOpen(api("go1.4.txt")),
		mustOpen(api("go1.5.txt")),
		mustOpen(api("go1.6.txt")),
		mustOpen(api("go1.7.txt")),
		mustOpen(api("go1.8.txt")),
		mustOpen(api("go1.9.txt")),
		mustOpen(api("go1.10.txt")),
	)
	sc := bufio.NewScanner(f)
	fullImport := map[string]string{} // "zip.NewReader" => "archive/zip"
	ambiguous := map[string]bool{}
	var keys []string
	for sc.Scan() {
		l := sc.Text()
		has := func(v string) bool { return strings.Contains(l, v) }
		if has("struct, ") || has("interface, ") || has(", method (") {
			continue
		}
		if m := sym.FindStringSubmatch(l); m != nil {
			full := m[1]
			key := path.Base(full) + "." + m[2]
			if exist, ok := fullImport[key]; ok {
				if exist != full {
					ambiguous[key] = true
				}
			} else {
				fullImport[key] = full
				keys = append(keys, key)
			}
		}
	}
	if err := sc.Err(); err != nil {
		log.Fatal(err)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if ambiguous[key] {
			outf("\t// %q is ambiguous\n", key)
		} else {
			outf("\t%q: %q,\n", key, fullImport[key])
		}
	}
	outf("\n")
	for _, sym := range [...]string{"Alignof", "ArbitraryType", "Offsetof", "Pointer", "Sizeof"} {
		outf("\t%q: %q,\n", "unsafe."+sym, "unsafe")
	}
	outf("}\n")
	fmtbuf, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("zstdlib.go", fmtbuf, 0666)
	if err != nil {
		log.Fatal(err)
	}
}
