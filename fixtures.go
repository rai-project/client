// Code generated by "esc -o fixtures.go -pkg client _fixtures/final.yml _fixtures/m2.yml _fixtures/m3.yml _fixtures/eval.yml"; DO NOT EDIT.

package client

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// Dir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func Dir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// FSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func FSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		_ = f.Close()
		return b, err
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// FSMustByte is the same as FSByte, but panics if name is not present.
func FSMustByte(useLocal bool, name string) []byte {
	b, err := FSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// FSString is the string version of FSByte.
func FSString(useLocal bool, name string) (string, error) {
	b, err := FSByte(useLocal, name)
	return string(b), err
}

// FSMustString is the string version of FSMustByte.
func FSMustString(useLocal bool, name string) string {
	return string(FSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/_fixtures/eval.yml": {
		local:   "_fixtures/eval.yml",
		size:    1252,
		modtime: 1513796737,
		compressed: `
H4sIAAAAAAAC/+ySQY/TMBCF7/kVTy29ILnuVhWswpGfsOJcOc4kserYlsdu6b9HdtIKoYUb4sKp6syb
b94bZ4u33M2G2XiHwVjC4CNIE06HVwzKWhwPL5+bqEzbAFeKRdnisD82gJnVSC30LZCK7J0s2kGdSdPp
8HqevztK597rC8VWzf2nkxhDFlYl4tREYp+jJi5gHXL5AVTUk0mkU47Uog41wPh+OyjWytaG9tmlFi8N
4CjdfLy0xT5To/08K9fXNV02tl9IArIzTnaKJwiNjQ4Q8Qq5eBcctfwIWSPI8scHiir5KHXm5OcNttA+
3DF629NyO0byqBNYoiFFoj9uW9dUX09k8msF7JEmetBUqHeGcbWag/WqfxdfHpGjLsr1LeqiL+g9yuor
PpT27+PJfdE6enpaLfwS86d8zmiCcMcDZnUhiK8rHdvl6svEKg4mwDhO5fsSIjNFCHrYCfc0eQdg+9Ss
pc643riRH5kzx5o7mZkgBmx23ypr98Z3TjRjR2RVYOo3D4Skq7KCdTQhsRyMU3Yf7uuVhPW3v8aezDj9
h/8D+I8AAAD//54xew/kBAAA
`,
	},

	"/_fixtures/final.yml": {
		local:   "_fixtures/final.yml",
		size:    880,
		modtime: 1511905976,
		compressed: `
H4sIAAAAAAAC/7RSzYrbMBC++yk+ku6loCgbQru4xz7C0nNQ5LEtoj80UtK8fZHshFK2vfVkPPPN9zOj
Ld7L2RlmEzxGYwljSCBNOO7fMCprcdi/fu2SMn0HXClVZI/97tABxqmJeuhbJJU4eFmxozqRpuP+7eR+
esqnIegLpV654ctRTLEIqzJx7hJxKEkTV2IdS/0AKunZZNK5JOrRhjpg+rgdFWtlW0OH4nOP1w7wlG8h
Xfpqn6nTwTnlhyZzLsYOC5OAPBsvz4pnCI2NjhDpCrl4F5y0/AzZIsj6EyIllUOSunAOboMtdIh3TMEO
tOyOkQPaBJZoyInon2qrTPP1pMxhrYAD8kwPNhXbnmF8q5Zogxo+pK9H5KQrcr1FE/qGIaBKX/Gptv8e
T+4q1tPT02rhj5i/5fNGE4Q/7OHUhSC+r+zYLltfJlZwNBHGc67vS4jClCDoYSfe8xw8gO0Ts5bOxg/G
T/zIXDi13Nk4ghixefnRuF7e+c6ZHF6IrIpMw+ZBIemqrGCdTMwsR+OV3cX7uiUxm2n+b+Q23LpfAQAA
//8eN0w8cAMAAA==
`,
	},

	"/_fixtures/m2.yml": {
		local:   "_fixtures/m2.yml",
		size:    890,
		modtime: 1511906017,
		compressed: `
H4sIAAAAAAAC/7RSTY/TMBC951c8tewFyXW2qmAVjqvlxglxrlxn2lj1lzx2S/89spNWCC3cOEWZefM+
ZrzGN2OJc/CELbgcnGE2weNoLOEYEt5e37DrX/BVWYtt//y5S8oMHXChVJED+s22A4xTJxqgr5FU4uBl
xR7VnjTt+pe9++kp78egz5QG5cZPO6FjEVZl4twl4lCSJq7EOpb6AVTSk8mkc0k0oA11wOn9dlSslW0N
HYrPA/oO8JSvIZ0HHJVl6nRwTvmxyRyKsePMJCAPxsuD4glCY6UjRLpAzt4FJy0/QrYIsv6ESEnlkKQu
nINbYQ0d4g2nYEead8fIAW0CczTkRPRPtUWm+XpQ5rBUwAF5ojubim3PML5VS7RBje/S1yNy0hW53KIJ
fcEYUKUv+FDbf48nNxXr6eFpsfBHzN/yeaMJwm97OHUmiNeFHet56/PEAo4mwnjO9X0JUZgSBN3txFue
ggewfmCW0sH40fgT3zMXTi13No4gjlg9/WhcT9/5xpkcnoisikzj6k4h6aKsYJ1MzCzddvO8ibdlSWIy
p+l/cdtw7X4FAAD//7D4ch96AwAA
`,
	},

	"/_fixtures/m3.yml": {
		local:   "_fixtures/m3.yml",
		size:    924,
		modtime: 1511905979,
		compressed: `
H4sIAAAAAAAC/7RSTY/TMBC951c8tewFyU3brWAVjqvlxglxrlxnkpg6tjVjt/TfIydpD2jhximK5/l9
jdf4Zh1JCp7wDMmn0YrY4NFZR+gC4+31DYftC75q57Df7j5XrG1TARfigmyw3ewrwI66pwbmGkmzBF8X
bKePZOiwfTmOvzylYxvMmbjRY/vpoPqYldOJJFVMEjIbkkJsYi4fQLMZbCKTMlOD6VIF9O+Poxaj3TQw
IfvUYFcBntI18LlBp51QZcI4at9OMqdsXTszKdQn6+uTlgHKYGUiFF9Qz96VsKk/op4i1OUnRGKdAtcm
SwrjCmuYEG/og2tp7k6QAqYbmKMhMdE/1RaZydeDMnL4SSahK9RcSGcAJCANdCfXcaod1k+nObqg23fV
yk6FTUEuq5l0v6ANKE78BR/K/O9x600Be3p4XDz8EbvkxdWmIeSEcCG+sk33Brw1BOX3W4z6TFCvix7W
815mjgUcbYT1ksoLVCoLMRTdDcZbGoIHsH5glqOT9a31vdxryMJTFcmOBNVh9fRj4nr6LjdJNOKJyOko
1K7uFDVdtFNi2MYk9fi82W3ibelNDbYf/he3C9fqdwAAAP//2MCaAJwDAAA=
`,
	},

	"/": {
		isDir: true,
		local: "",
	},

	"/_fixtures": {
		isDir: true,
		local: "_fixtures",
	},
}
