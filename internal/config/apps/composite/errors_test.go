package composite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	// Test error wrapping relationships
	require.ErrorIs(t, ErrNoScriptsSpecified, ErrAppCompositeScript)
	require.ErrorIs(t, ErrEmptyScriptID, ErrAppCompositeScript)
	require.ErrorIs(t, ErrInvalidStaticData, ErrAppCompositeScript)
	require.ErrorIs(t, ErrProtoConversion, ErrAppCompositeScript)
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "Base error",
			err:  ErrAppCompositeScript,
			want: "app composite script error",
		},
		{
			name: "No scripts specified",
			err:  ErrNoScriptsSpecified,
			want: "app composite script error: no scripts specified",
		},
		{
			name: "Empty script ID",
			err:  ErrEmptyScriptID,
			want: "app composite script error: empty script ID",
		},
		{
			name: "Invalid static data",
			err:  ErrInvalidStaticData,
			want: "app composite script error: invalid static data",
		},
		{
			name: "Proto conversion error",
			err:  ErrProtoConversion,
			want: "app composite script error: proto conversion error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}
