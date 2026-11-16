/*
Copyright 2026 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workflow

import (
	"testing"

	"github.com/dapr/durabletask-go/api/protos"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestWithTerminateReasonEncodesOptionalFieldFive(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{name: "non-empty", reason: "cancelled by customer"},
		{name: "explicit empty", reason: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := new(protos.TerminateRequest)
			require.NoError(t, withTerminateReason(tt.reason)(req))

			unknown := req.ProtoReflect().GetUnknown()
			number, wireType, tagLength := protowire.ConsumeTag(unknown)
			require.Equal(t, protowire.Number(5), number)
			require.Equal(t, protowire.BytesType, wireType)
			require.Positive(t, tagLength)

			payload, payloadLength := protowire.ConsumeBytes(unknown[tagLength:])
			require.Positive(t, payloadLength)

			value := new(wrapperspb.StringValue)
			require.NoError(t, proto.Unmarshal(payload, value))
			require.Equal(t, tt.reason, value.GetValue())
		})
	}
}

func TestWithTerminateReasonRejectsNilRequest(t *testing.T) {
	require.ErrorContains(t, withTerminateReason("reason")(nil), "terminate request is nil")
}

func TestWithTerminateReasonPreservesExistingUnknownFields(t *testing.T) {
	req := new(protos.TerminateRequest)
	original := protowire.AppendTag(nil, 99, protowire.VarintType)
	original = protowire.AppendVarint(original, 1)
	req.ProtoReflect().SetUnknown(original)

	require.NoError(t, withTerminateReason("reason")(req))

	unknown := req.ProtoReflect().GetUnknown()
	number, wireType, tagLength := protowire.ConsumeTag(unknown)
	require.Equal(t, protowire.Number(99), number)
	require.Equal(t, protowire.VarintType, wireType)
	_, valueLength := protowire.ConsumeVarint(unknown[tagLength:])
	require.Positive(t, valueLength)

	next := unknown[tagLength+valueLength:]
	number, wireType, tagLength = protowire.ConsumeTag(next)
	require.Equal(t, protowire.Number(5), number)
	require.Equal(t, protowire.BytesType, wireType)
	require.Positive(t, tagLength)
}
