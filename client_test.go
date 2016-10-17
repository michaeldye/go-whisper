package whisper

import (
	"testing"
)

// AND between terms
func Test_client_maps_msg_params_array(t *testing.T) {
	paramMap := make(map[string]interface{}, 1)
	paramMap["topics"] = []string{"micropayment", "handout"}

	if params, err := MapMsgParams(paramMap, ToHex); err != nil {
		t.Error(err)
	} else {
		out := NewWhisperRPCOutgoingMsg("POST", params)
		topics := out.Params[0].(map[string]interface{})["topics"]

		if len(topics.([]interface{})) != 2 {
			t.Errorf("Expected a two-member topic array, but got: %v", topics)
		}
	}
}

// OR between terms
func Test_client_maps_msg_params_nested_arrays(t *testing.T) {
	paramMap := make(map[string]interface{}, 1)
	paramMap["topics"] = [][]string{[]string{"micropayment", "handout"}}

	if params, err := MapMsgParams(paramMap, ToHex); err != nil {
		t.Error(err)
	} else {
		out := NewWhisperRPCOutgoingMsg("POST", params)
		topics := out.Params[0].(map[string]interface{})["topics"]

		if len(topics.([][]string)) != 1 {
			t.Errorf("Expected a one-member topic array, but got: %v", topics)
		} else if len(topics.([][]string)[0]) != 2 {
			t.Errorf("Expected two values in single-member topic array, but got: %v", topics)
		}
	}
}
