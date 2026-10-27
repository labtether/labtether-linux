package agentmgr

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// These Compose cases match protocol/testdata/conformance/agent-wire-v1.json.
// Exercise this repo's duplicated types: dropping or reordering the selection
// can make an otherwise valid request act on the wrong Compose project files.
func TestDockerComposeWireCompatibility(t *testing.T) {
	t.Run("endpoint field types", testComposeEndpointWireTypes)
	raw, err := os.ReadFile("testdata/compose-wire-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Schema string `json:"schema"`
		Cases  []struct {
			Name         string          `json:"name"`
			DataContract string          `json:"data_contract"`
			Message      json.RawMessage `json:"message"`
		} `json:"cases"`
	}
	strict := func(t *testing.T, raw []byte, destination any) {
		t.Helper()
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(destination); err != nil {
			t.Fatal(err)
		}
	}
	strict(t, raw, &fixture)
	if fixture.Schema != "labtether-agent-wire-conformance/v1" || len(fixture.Cases) != 6 {
		t.Fatal("expected the six legacy/v2 Compose cases")
	}
	for _, item := range fixture.Cases {
		t.Run(item.Name, func(t *testing.T) {
			var message Message
			strict(t, item.Message, &message)
			if !IsKnownMessageType(message.Type) || message.ID == "" {
				t.Fatal("unregistered or uncorrelated Compose message")
			}
			var contract any
			switch item.DataContract {
			case "DockerDiscoveryData":
				contract = &DockerDiscoveryData{}
			case "DockerDiscoveryDeltaData":
				contract = &DockerDiscoveryDeltaData{}
			case "DockerComposeActionData":
				contract = &DockerComposeActionData{}
			default:
				t.Fatalf("unsupported Compose contract %q", item.DataContract)
			}
			strict(t, message.Data, contract)
			roundTrip, err := json.Marshal(contract)
			if err != nil {
				t.Fatal(err)
			}
			var want, got any
			strict(t, message.Data, &want)
			strict(t, roundTrip, &got)
			wantJSON, _ := json.Marshal(want)
			gotJSON, _ := json.Marshal(got)
			if !bytes.Equal(wantJSON, gotJSON) {
				t.Fatalf("Compose wire shape or config order changed:\nwant %s\ngot  %s", wantJSON, gotJSON)
			}
		})
	}
}

// Legacy messages may omit endpoint identity. A non-string identity must not
// silently decode to an empty selector; v2 runtime owners validate string values.
func testComposeEndpointWireTypes(t *testing.T) {
	for _, item := range []struct {
		name       string
		field      string
		newPayload func() any
	}{
		{"full", "compose_endpoint_id", func() any { return &DockerDiscoveryData{} }},
		{"delta", "compose_endpoint_id", func() any { return &DockerDiscoveryDeltaData{} }},
		{"action", "endpoint_id", func() any { return &DockerComposeActionData{} }},
	} {
		t.Run(item.name, func(t *testing.T) {
			raw := []byte(`{"` + item.field + `":42}`)
			if err := json.Unmarshal(raw, item.newPayload()); err == nil {
				t.Fatal("non-string endpoint identity accepted")
			}
		})
	}
}
