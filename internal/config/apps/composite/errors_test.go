package composite

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	// Test error wrapping relationships
	assert.True(t, errors.Is(ErrNoScriptsSpecified, ErrAppCompositeScript))
	assert.True(t, errors.Is(ErrEmptyScriptID, ErrAppCompositeScript))
	assert.True(t, errors.Is(ErrInvalidStaticData, ErrAppCompositeScript))
	assert.True(t, errors.Is(ErrProtoConversion, ErrAppCompositeScript))
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
