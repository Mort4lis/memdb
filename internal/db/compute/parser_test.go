package compute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantErr    bool
		wantResult Query
	}{
		{
			name:  "Successful SET",
			input: "SET test-key test-val",
			wantResult: Query{
				cmdID: SetCommandID,
				args:  []string{"test-key", "test-val"},
			},
		},
		{
			name:  "Successful GET",
			input: "GET test-key",
			wantResult: Query{
				cmdID: GetCommandID,
				args:  []string{"test-key"},
			},
		},
		{
			name:  "Successful DEL",
			input: "DEL test-key",
			wantResult: Query{
				cmdID: DelCommandID,
				args:  []string{"test-key"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gotResult, err := ParseQuery(testCase.input)
			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.wantResult, gotResult)
			}
		})
	}
}
