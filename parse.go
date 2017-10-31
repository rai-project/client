package client

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rai-project/model"
)

var (
	colorRe         = regexp.MustCompile(`\[(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]`)
	timeResultRe    = regexp.MustCompile(`Time:\s*([-+]?[0-9]*\.?[0-9]+)`)
	programOutputRe = regexp.MustCompile(`Done with ([-+]?[0-9]*\.?[0-9]+)\s.*elapsed = ([-+]?[0-9]*\.?[0-9]+)\smilliseconds.\sCorrectness:\s([-+]?[0-9]*\.?[0-9]+)`)
	projectURLRe    = regexp.MustCompile(`✱ The build folder has been uploaded to (\s*\[+?\s*(\!?)\s*([a-z]*)\s*\|?\s*([a-z0-9\.\-_]*)\s*\]+?)?\s*([^\s]+)\s*\..*`)
)

type ranking struct {
	OpFullRuntime time.Duration
}

func parseProgramOutputLine(ranking *model.Ranking, s string) {
	// if !programOutputRe.MatchString(s) {
	// 	return
	// }
	// matches := programOutputRe.FindAllStringSubmatch(s, 1)[0]
	// if len(matches) < 4 {
	// 	log.WithField("match_count", len(matches)).
	// 		Debug("Unexpected number of matches while parsing program output")
	// 	return
	// }
	// batchSize, err := strconv.Atoi(matches[1])
	// if err == nil {
	// 	ranking.BatchSize = batchSize
	// }
	// elapsedTime, err := time.ParseDuration(matches[2] + "ms")
	// if err == nil {
	// 	ranking.Runtime = elapsedTime
	// }
	// correctness, err := strconv.ParseFloat(matches[3], 64)
	// if err == nil {
	// 	ranking.Correctness = correctness
	// }
	return
}

func parseTimeResult(ranking *model.Ranking, s string) {
	if !timeResultRe.MatchString(s) {
		return
	}
	matches := timeResultRe.FindAllStringSubmatch(s, 1)[0]
	if len(matches) < 1 {
		log.WithField("match_count", len(matches)).
			Debug("Unexpected number of matches while parsing time result")
		return
	}
	op, err := time.ParseDuration(matches[1] + "s")
	if err == nil {
		ranking.OpFullRuntime = op
	}
}

func parseProjectURL(ranking *model.Ranking, s string) {
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
		ranking.S3Key = u.String()
		ranking.ProjectURL = u.String()
	}
	return
}

func removeColor(s string) string {
	s = strings.Replace(s, "\x1b", "", -1)
	return colorRe.ReplaceAllString(s, "")
}

func parseLine(ranking *model.Ranking, s string) {
	s = removeColor(s)
	parseProgramOutputLine(ranking, s)
	parseTimeResult(ranking, s)
	parseProjectURL(ranking, s)
}
