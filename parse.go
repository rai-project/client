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
	colorRe         = regexp.MustCompile(`\[(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]`)
	timeOutputRe    = regexp.MustCompile(`([0-9]*\.?[0-9]+)user\s+([0-9]*\.?[0-9]+)system\s+([0-9]*\.?[0-9]+)elapsed.*`)
	programOutputRe = regexp.MustCompile(`Correctness: ([-+]?[0-9]*\.?[0-9]+)\s+Model: (.*)`)
	opTimeOutputRe  = regexp.MustCompile(`Op Time: ([-+]?[0-9]*\.?[0-9]+)`)
	projectURLRe    = regexp.MustCompile(`✱ The build folder has been uploaded to (\s*\[+?\s*(\!?)\s*([a-z]*)\s*\|?\s*([a-z0-9\.\-_]*)\s*\]+?)?\s*([^\s]+)\s*\..*`)
	newInferenceRe  = regexp.MustCompile(`New Inference`)
)

func parseNewInference(ranking *model.Fa2017Ece408Job, s string) {
	if !newInferenceRe.MatchString(s) {
		return
	}
	ranking.StartNewInference()
	return
}

func parseProgramOutput(ranking *model.Fa2017Ece408Job, s string) {
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
		ranking.CurrentInference().Correctness = correctness
	}
	ranking.CurrentInference().Model = matches[2]

	return
}

func parseOpTimeOutput(ranking *model.Fa2017Ece408Job, s string) {
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
		ranking.CurrentInference().OpRuntime += elapsed
	}

	return
}

func parseTimeOutput(ranking *model.Fa2017Ece408Job, s string) {
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
		ranking.CurrentInference().UserFullRuntime = user
	}
	system, err := time.ParseDuration(matches[2] + "s")
	if err == nil {
		ranking.CurrentInference().SystemFullRuntime = system
	}
	elapsed, err := time.ParseDuration(matches[3] + "s")
	if err == nil {
		ranking.CurrentInference().ElapsedFullRuntime = elapsed
	}
}

func parseProjectURL(ranking *model.Fa2017Ece408Job, s string) {
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
		ranking.ProjectURL = u.String()
	}
	return
}

func removeColor(s string) string {
	s = strings.Replace(s, "\x1b", "", -1)
	return colorRe.ReplaceAllString(s, "")
}

func parseLine(ranking *model.Fa2017Ece408Job, s string) {
	s = removeColor(s)
	parseNewInference(ranking, s)
	parseOpTimeOutput(ranking, s)
	parseProgramOutput(ranking, s)
	parseTimeOutput(ranking, s)
	parseProjectURL(ranking, s)
}
