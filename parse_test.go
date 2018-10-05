package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rai-project/model"
)

func TestParse(t *testing.T) {
	s := "✱ The build folder has been uploaded to http://s3.amazonaws.com/rai-server/uploads%2F629mfvXRR.tar.bz2. The data will be present for only a short duration of time."

	ranking := &model.Ece408Job{}
	require.True(t, projectURLRe.MatchString(s))
	parseLine(ranking, s)
	assert.NotEmpty(t, ranking)
}

func TestParseTimeResult(t *testing.T) {
	s := "4.85user 2.97system 5.25elapsed 148%CPU (0avgtext+0avgdata 1319488maxresident)"

	ranking := &model.Ece408Job{}
	ranking.StartNewInference()
	require.True(t, timeOutputRe.MatchString(s))
	parseTimeOutput(ranking, s)
	fmt.Println(ranking)
	assert.NotEmpty(t, ranking)
}

func TestParseProgramOutput(t *testing.T) {
	s := "Correctness: 0.1 Model: blah-wh@tever$/ugh"

	ranking := &model.Ece408Job{}
	ranking.StartNewInference()
	require.True(t, programOutputRe.MatchString(s))
	parseProgramOutput(ranking, s)
	fmt.Println(ranking)
	assert.Equal(t, 0.1, ranking.CurrentInference().Correctness)
	assert.Equal(t, "blah-wh@tever$/ugh", ranking.CurrentInference().Model)
}

func TestParseNewRanking(t *testing.T) {
	s := "[New Inference]"

	ranking := &model.Ece408Job{}
	require.True(t, newInferenceRe.MatchString(s))
	parseNewInference(ranking, s)
	parseNewInference(ranking, s)
	fmt.Println(ranking)
	assert.Equal(t, 2, len(ranking.Inferences))
}
