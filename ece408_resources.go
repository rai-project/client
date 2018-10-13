// +build ece408ProjectMode

package client

var (
	m1Name     = "_fixtures/m1.yml"
	m2Name     = "_fixtures/m2.yml"
	m3Name     = "_fixtures/m3.yml"
	m4Name     = "_fixtures/m4.yml"
	finalName  = "_fixtures/final.yml"
	evalName   = "_fixtures/eval.yml"
	m1Build    = _escFSMustByte(false, "/"+m1Name)
	m2Build    = _escFSMustByte(false, "/"+m2Name)
	m3Build    = _escFSMustByte(false, "/"+m3Name)
	m4Build    = _escFSMustByte(false, "/"+m4Name)
	finalBuild = _escFSMustByte(false, "/"+finalName)
	evalBuild  = _escFSMustByte(false, "/"+evalName)
)
