//go:generate go get -v github.com/mjibson/esc
//go:generate go get -v github.com/mailru/easyjson/...
//go:generate esc -o fixtures.go -pkg client -private _fixtures/m1.yml _fixtures/m2.yml _fixtures/m3.yml _fixtures/m4.yml _fixtures/final.yml _fixtures/eval.yml
//go:generate easyjson -snake_case -disallow_unknown_fields -pkg .

package client
