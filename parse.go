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
	timeResultRe    = regexp.MustCompile(`([0-9]*\.?[0-9]+)user\s+([0-9]*\.?[0-9]+)system\s+([0-9]*\.?[0-9]+)elapsed.*`)
	programOutputRe = regexp.MustCompile(`Correctness: ([-+]?[0-9]*\.?[0-9]+)\s+Batch Size: ([-+]?[0-9]*\.?[0-9]+) Model: (.*)`)
	programTimeRe   = regexp.MustCompile(`Time: ([-+]?[0-9]*\.?[0-9]+)`)
	projectURLRe    = regexp.MustCompile(`✱ The build folder has been uploaded to (\s*\[+?\s*(\!?)\s*([a-z]*)\s*\|?\s*([a-z0-9\.\-_]*)\s*\]+?)?\s*([^\s]+)\s*\..*`)
)

func parseProgramOutput(ranking *model.Fa2017Ece408Ranking, s string) {
	if !programOutputRe.MatchString(s) {
		return
	}
	matches := programOutputRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 4 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing program output")
		return
	}
	correctness, err := strconv.ParseFloat(matches[1], 64)
	if err == nil {
		ranking.Correctness = correctness
	}
	batchSize, err := strconv.Atoi(matches[2])
	if err == nil {
		ranking.BatchSize = batchSize
	}
	ranking.Model = matches[3]

	return
}

func parseProgramTime(ranking *model.Fa2017Ece408Ranking, s string) {
	if !programTimeRe.MatchString(s) {
		return
	}
	matches := programTimeRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 2 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing program time")
		return
	}
	elapsed, err := time.ParseDuration(matches[1] + "s")
	if err == nil {
		ranking.OpRuntime = elapsed
	}

	return
}

func parseTimeResult(ranking *model.Fa2017Ece408Ranking, s string) {
	if !timeResultRe.MatchString(s) {
		return
	}
	matches := timeResultRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 4 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing time result")
		return
	}
	user, err := time.ParseDuration(matches[1] + "s")
	if err == nil {
		ranking.UserFullRuntime = user
	}
	system, err := time.ParseDuration(matches[2] + "s")
	if err == nil {
		ranking.SystemFullRuntime = system
	}
	elapsed, err := time.ParseDuration(matches[3] + "s")
	if err == nil {
		ranking.ElapsedFullRuntime = elapsed
	}
}

func parseProjectURL(ranking *model.Fa2017Ece408Ranking, s string) {
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

func parseLine(ranking *model.Fa2017Ece408Ranking, s string) {
	s = removeColor(s)
	parseProgramTime(ranking, s)
	parseProgramOutput(ranking, s)
	parseTimeResult(ranking, s)
	parseProjectURL(ranking, s)
}
