package client

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rai-project/model"
)

// func TestOutputInformation(t *testing.T) {

// 	config.Init()

// 	require.NotNil(t, 1)

// 	s := "Done with 10 queries in elapsed = 1231.73 milliseconds. Correctness: 1\n"

// 	require.True(t, programOutputRe.MatchString(s))

// 	matches := programOutputRe.FindAllStringSubmatch(s, 1)[0]
// 	assert.NotEmpty(t, matches)

// }

func TestParse(t *testing.T) {
	s := "✱ The build folder has been uploaded to http://s3.amazonaws.com/rai-server/uploads%2F629mfvXRR.tar.bz2. The data will be present for only a short duration of time."

	ranking := &model.Fa2017Ece408Ranking{}
	require.True(t, projectURLRe.MatchString(s))
	parseLine(ranking, s)
	assert.NotEmpty(t, ranking)
}

func TestParseTimeResult(t *testing.T) {
	s := "4.85user 2.97system 5.25elapsed 148%CPU (0avgtext+0avgdata 1319488maxresident)"

	ranking := &model.Fa2017Ece408Ranking{}
	require.True(t, timeResultRe.MatchString(s))
	parseTimeResult(ranking, s)
	fmt.Println(ranking)
	assert.NotEmpty(t, ranking)
}

func TestParseProgramOutput(t *testing.T) {
	s := "Correctness: 0.1 Batch Size: 10000 Model: blah-wh@tever$/ugh"

	ranking := &model.Fa2017Ece408Ranking{}
	require.True(t, programOutputRe.MatchString(s))
	parseProgramOutput(ranking, s)
	fmt.Println(ranking)
	assert.Equal(t, 0.1, ranking.Correctness)
	assert.Equal(t, 10000, ranking.BatchSize)
	assert.Equal(t, "blah-wh@tever$/ugh", ranking.Model)
}
