package staticdata

import (
	"testing"

	pbData "github.com/atlanticdynamic/firelynx/gen/settings/v1alpha1/data/v1"
	"github.com/robbyt/protobaggins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestStaticDataToProto(t *testing.T) {
	t.Run("NilStaticData", func(t *testing.T) {
		var sd *StaticData
		pb := sd.ToProto()
		assert.Nil(t, pb)
	})

	t.Run("EmptyStaticData", func(t *testing.T) {
		sd := &StaticData{
			MergeMode: StaticDataMergeModeUnspecified,
		}
		pb := sd.ToProto()
		require.NotNil(t, pb.MergeMode)
		assert.Equal(
			t,
			pbData.StaticData_MERGE_MODE_UNSPECIFIED,
			*pb.MergeMode,
		)
		assert.Nil(t, pb.Data)
	})

	t.Run("FullStaticData", func(t *testing.T) {
		sd := &StaticData{
			Data: map[string]any{
				"string": "value",
				"number": 42.0,
				"bool":   true,
			},
			MergeMode: StaticDataMergeModeLast,
		}
		pb := sd.ToProto()
		require.NotNil(t, pb.MergeMode)
		assert.Equal(
			t,
			pbData.StaticData_MERGE_MODE_LAST,
			*pb.MergeMode,
		)
		assert.Len(t, pb.Data, 3)

		// Verify the converted values
		assert.Equal(t, "value", pb.Data["string"].GetStringValue())
		assert.Equal(t, 42.0, pb.Data["number"].GetNumberValue())
		assert.Equal(t, true, pb.Data["bool"].GetBoolValue())
	})
}

func TestFromProto(t *testing.T) {
	t.Run("NilProto", func(t *testing.T) {
		var pb *pbData.StaticData
		sd, err := FromProto(pb)
		require.NoError(t, err)
		assert.Nil(t, sd)
	})

	t.Run("EmptyProto", func(t *testing.T) {
		mergeMode := pbData.StaticData_MERGE_MODE_UNSPECIFIED
		pb := &pbData.StaticData{
			MergeMode: &mergeMode,
		}
		sd, err := FromProto(pb)
		require.NoError(t, err)
		assert.Equal(t, StaticDataMergeModeUnspecified, sd.MergeMode)
		assert.Nil(t, sd.Data)
	})

	t.Run("FullProto", func(t *testing.T) {
		// Create a proto StaticData with some values
		mergeMode := pbData.StaticData_MERGE_MODE_UNIQUE
		pb := &pbData.StaticData{
			Data:      map[string]*structpb.Value{},
			MergeMode: &mergeMode,
		}

		pb.Data["string"] = protobaggins.TryNewStructValue("value")
		pb.Data["number"] = protobaggins.TryNewStructValue(42.0)
		pb.Data["bool"] = protobaggins.TryNewStructValue(true)

		sd, err := FromProto(pb)
		require.NoError(t, err)
		assert.Equal(t, StaticDataMergeModeUnique, sd.MergeMode)
		assert.Len(t, sd.Data, 3)

		// Verify the converted values
		assert.Equal(t, "value", sd.Data["string"])
		assert.Equal(t, 42.0, sd.Data["number"])
		assert.Equal(t, true, sd.Data["bool"])
	})
}

func TestStaticDataMergeModeConversion(t *testing.T) {
	// Test conversion from domain to proto
	t.Run("DomainToProto", func(t *testing.T) {
		tests := []struct {
			domain   StaticDataMergeMode
			expected pbData.StaticData_MergeMode
		}{
			{
				StaticDataMergeModeUnspecified,
				pbData.StaticData_MERGE_MODE_UNSPECIFIED,
			},
			{
				StaticDataMergeModeLast,
				pbData.StaticData_MERGE_MODE_LAST,
			},
			{
				StaticDataMergeModeUnique,
				pbData.StaticData_MERGE_MODE_UNIQUE,
			},
			{
				StaticDataMergeMode(999),
				pbData.StaticData_MERGE_MODE_UNSPECIFIED,
			}, // Invalid defaults to unspecified
		}

		for _, tt := range tests {
			result := staticDataMergeModeToProto(tt.domain)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, *result)
		}
	})

	// Test conversion from proto to domain
	t.Run("ProtoToDomain", func(t *testing.T) {
		tests := []struct {
			proto    pbData.StaticData_MergeMode
			expected StaticDataMergeMode
		}{
			{
				pbData.StaticData_MERGE_MODE_UNSPECIFIED,
				StaticDataMergeModeUnspecified,
			},
			{
				pbData.StaticData_MERGE_MODE_LAST,
				StaticDataMergeModeLast,
			},
			{
				pbData.StaticData_MERGE_MODE_UNIQUE,
				StaticDataMergeModeUnique,
			},
			{
				pbData.StaticData_MergeMode(999),
				StaticDataMergeModeUnspecified,
			}, // Invalid defaults to unspecified
		}

		for _, tt := range tests {
			result := protoToStaticDataMergeMode(tt.proto)
			assert.Equal(t, tt.expected, result)
		}
	})

	// Test round-trip conversion
	t.Run("RoundTrip", func(t *testing.T) {
		modes := []StaticDataMergeMode{
			StaticDataMergeModeUnspecified,
			StaticDataMergeModeLast,
			StaticDataMergeModeUnique,
		}

		for _, mode := range modes {
			protoPtr := staticDataMergeModeToProto(mode)
			require.NotNil(t, protoPtr)
			result := protoToStaticDataMergeMode(*protoPtr)
			assert.Equal(t, mode, result)
		}
	})
}
