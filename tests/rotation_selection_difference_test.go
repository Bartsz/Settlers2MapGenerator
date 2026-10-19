package tests

import (
	"bytes"
	"encoding/json"
	engine "settlers2-final-go/InternalMapManipulation"
	"testing"
)

// Old projects may contain the removed rotateResourceMasks property.
// Both true and false must use the same mandatory five-player rotation.
func TestFivePlayerLegacyRotationFlagIsIgnored(t *testing.T) {
	payload := buildRotationPayload(5, 109, 55, false)
	var p map[string]any
	if err := json.Unmarshal(payload, &p); err != nil {
		t.Fatal(err)
	}
	delete(p, "rotateResourceMasks")
	current, _ := json.Marshal(p)
	original, err := engine.Generate(current)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.ValidateSWD(original.SWD); err != nil {
		t.Fatal(err)
	}
	for _, legacy := range []bool{false, true} {
		p["rotateResourceMasks"] = legacy
		old, _ := json.Marshal(p)
		result, err := engine.Generate(old)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(original.SWD, result.SWD) {
			t.Fatalf("legacy rotateResourceMasks=%v changed the map", legacy)
		}
	}
}
