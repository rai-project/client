//go:generate go get -v github.com/mjibson/esc
//go:generate esc -o fixtures.go -pkg client -private _fixtures/m1.yml _fixtures/m2.yml _fixtures/m3.yml _fixtures/m4.yml _fixtures/final.yml _fixtures/eval.yml

package client
