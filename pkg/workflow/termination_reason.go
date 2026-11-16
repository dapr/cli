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
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/dapr/durabletask-go/api/protos"
)

const executionTerminatedReasonFieldNumber protowire.Number = 3

// executionTerminationReason reads the dedicated ExecutionTerminatedEvent.reason
// field. It supports both generated types that know about field 3 and older
// durabletask-go versions that preserve it as an unknown protobuf field.
func executionTerminationReason(event *protos.ExecutionTerminatedEvent) (string, bool) {
	if event == nil {
		return "", false
	}

	message := event.ProtoReflect()
	if field := message.Descriptor().Fields().ByName("reason"); field != nil && message.Has(field) {
		if value, ok := message.Get(field).Message().Interface().(*wrapperspb.StringValue); ok {
			return value.GetValue(), true
		}
	}

	unknown := message.GetUnknown()
	for len(unknown) > 0 {
		number, wireType, tagLength := protowire.ConsumeTag(unknown)
		if tagLength < 0 {
			return "", false
		}
		unknown = unknown[tagLength:]

		if number == executionTerminatedReasonFieldNumber && wireType == protowire.BytesType {
			payload, payloadLength := protowire.ConsumeBytes(unknown)
			if payloadLength < 0 {
				return "", false
			}
			value := new(wrapperspb.StringValue)
			if err := proto.Unmarshal(payload, value); err != nil {
				return "", false
			}
			return value.GetValue(), true
		}

		fieldLength := protowire.ConsumeFieldValue(number, wireType, unknown)
		if fieldLength < 0 {
			return "", false
		}
		unknown = unknown[fieldLength:]
	}

	return "", false
}
