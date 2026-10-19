package server

import "testing"

//   F3a   assertKoreanError( ·  0) .

// TestTriggerErrorConstantsLocalized    3   .
func TestTriggerErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"no_condition", errTriggerNoCondition},
		{"tool_set_empty", errTriggerToolSetEmpty},
		{"custom_only", errTriggerCustomOnly},
	}
	for _, c := range cases {
		assertKoreanError(t, c.name, c.msg)
	}
}

// TestValidateTriggerLocalized  validateTrigger    ,  ·
// pgCreateTrigger·pgUpdateTrigger  writeErr(400, msg)    .
func TestValidateTriggerLocalized(t *testing.T) {
	if msg := validateTrigger(&triggerReq{}); msg != errTriggerNoCondition {
		t.Fatalf("   = %q,  = %q", msg, errTriggerNoCondition)
	}
	if msg := validateTrigger(&triggerReq{OnToolCall: true}); msg != errTriggerToolSetEmpty {
		t.Fatalf("   = %q,  = %q", msg, errTriggerToolSetEmpty)
	}
	if msg := validateTrigger(&triggerReq{OnFinding: true}); msg != "" {
		t.Fatalf("   = %q,  =  ", msg)
	}
	if msg := validateTrigger(&triggerReq{OnToolCall: true, ToolNames: []string{"nmap"}}); msg != "" {
		t.Fatalf("    = %q,  =  ", msg)
	}
}
