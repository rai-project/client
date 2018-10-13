// +build ece408ProjectMode

package client

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rai-project/model"
)

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

	validSubmissions = []string{
		"m1",
		"m2",
		"m3",
		"m4",
		"final",
		"eval",
	}

	colorRe         = regexp.MustCompile(`\[(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]`)
	timeOutputRe    = regexp.MustCompile(`([0-9]*\.?[0-9]+)user\s+([0-9]*\.?[0-9]+)system\s+([0-9]*\.?[0-9]+)elapsed.*`)
	programOutputRe = regexp.MustCompile(`Correctness: ([-+]?[0-9]*\.?[0-9]+)\s+Model: (.*)`)
	opTimeOutputRe  = regexp.MustCompile(`Op Time: ([-+]?[0-9]*\.?[0-9]+)`)
	projectURLRe    = regexp.MustCompile(`✱ The build folder has been uploaded to (\s*\[+?\s*(\!?)\s*([a-z]*)\s*\|?\s*([a-z0-9\.\-_]*)\s*\]+?)?\s*([^\s]+)\s*\..*`)
	newInferenceRe  = regexp.MustCompile(`New Inference`)
)

func parseNewInference(job *model.Ece408JobResponseBody, s string) {
	if !newInferenceRe.MatchString(s) {
		return
	}
	job.StartNewInference()
	return
}

func parseProgramOutput(job *model.Ece408JobResponseBody, s string) {
	if !programOutputRe.MatchString(s) {
		return
	}
	matches := programOutputRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 3 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing program output")
		return
	}

	correctness, err := strconv.ParseFloat(matches[1], 64)
	if err == nil {
		job.CurrentInference().Correctness = correctness
	}
	job.CurrentInference().Model = matches[2]

	return
}

func parseOpTimeOutput(job *model.Ece408JobResponseBody, s string) {
	if !opTimeOutputRe.MatchString(s) {
		return
	}
	matches := opTimeOutputRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 2 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing op time")
		return
	}
	elapsed, err := time.ParseDuration(matches[1] + "s")
	if err == nil {
		job.RecordOpRuntime(elapsed)
	}

	return
}

func parseTimeOutput(job *model.Ece408JobResponseBody, s string) {
	if !timeOutputRe.MatchString(s) {
		return
	}
	matches := timeOutputRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 4 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing time result")
		return
	}
	user, err := time.ParseDuration(matches[1] + "s")
	if err == nil {
		job.CurrentInference().UserFullRuntime = user
	}
	system, err := time.ParseDuration(matches[2] + "s")
	if err == nil {
		job.CurrentInference().SystemFullRuntime = system
	}
	elapsed, err := time.ParseDuration(matches[3] + "s")
	if err == nil {
		job.CurrentInference().ElapsedFullRuntime = elapsed
	}
}

func parseProjectURL(job *model.Ece408JobResponseBody, s string) {
	if !projectURLRe.MatchString(s) {
		return
	}
	matches := projectURLRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 2 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing project url")
		return
	}
	u, err := url.Parse(matches[len(matches)-1])
	if err == nil {
		job.ProjectURL = u.String()
	}
	return
}

func removeColor(s string) string {
	s = strings.Replace(s, "\x1b", "", -1)
	return colorRe.ReplaceAllString(s, "")
}

func (c *Client) parseLine(s string)  {
  if c.job == nil {
  c.job := &Ece408JobResponseBody{}
  }
  job, ok := c.job(*Ece408JobResponseBody)
  if !ok {
    panic("invalid job type")
  }
	s = removeColor(s)
	parseNewInference(job, s)
	parseOpTimeOutput(job, s)
	parseProgramOutput(job, s)
	parseTimeOutput(job, s)
	parseProjectURL(job, s)
}
