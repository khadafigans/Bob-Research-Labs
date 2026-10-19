package server

import "testing"

// asset_intercept.go·task_intercept.go       
// (validateAssetInterceptRuleReq·validateTaskInterceptRuleReq) DB    
//   F3a   assertKoreanError( ·  0) .
// ·(pattern·exact_ip·cidr·kind·action·block·allow)   ASCII 
func TestAssetInterceptErrorsLocalized(t *testing.T) {
	for _, c := range []struct{ label, msg string }{
		{"pattern_empty", errAssetInterceptPatternEmpty},
		{"bad_exact_ip", errAssetInterceptInvalidExactIPFmt},
		{"bad_cidr", errAssetInterceptInvalidCIDRFmt},
		{"bad_kind", errAssetInterceptInvalidKindFmt},
		{"bad_action", errTaskInterceptInvalidAction},
	} {
		assertKoreanError(t, c.label, c.msg)
	}

	//  : validateAssetInterceptRuleReq    
	for _, c := range []struct {
		label string
		req   assetInterceptRuleReq
	}{
		{"empty_pattern", assetInterceptRuleReq{Kind: "exact_ip", Pattern: "   "}},
		{"invalid_exact_ip", assetInterceptRuleReq{Kind: "exact_ip", Pattern: "not-an-ip"}},
		{"invalid_cidr", assetInterceptRuleReq{Kind: "cidr", Pattern: "nonsense"}},
		{"invalid_kind", assetInterceptRuleReq{Kind: "nope", Pattern: "example.com"}},
	} {
		req := c.req
		err := validateAssetInterceptRuleReq(&req)
		if err == nil {
			t.Fatalf("%s:  ( )", c.label)
		}
		assertKoreanError(t, c.label, err.Error())
	}

	okReq := assetInterceptRuleReq{Kind: "cidr", Pattern: "192.168.0.0/16"}
	if err := validateAssetInterceptRuleReq(&okReq); err != nil {
		t.Fatalf(" CIDR  : %v", err)
	}

	//  : validateTaskInterceptRuleReq  action  
	badAction := taskInterceptRuleReq{Action: "nope", Kind: "exact_domain", Pattern: "example.com"}
	if err := validateTaskInterceptRuleReq(&badAction); err == nil {
		t.Fatalf("bad_action_path:  ( )")
	} else {
		assertKoreanError(t, "bad_action_path", err.Error())
	}

	defaultAction := taskInterceptRuleReq{Action: "", Kind: "exact_domain", Pattern: "example.com"}
	if err := validateTaskInterceptRuleReq(&defaultAction); err != nil {
		t.Fatalf(" action  : %v", err)
	}
	if defaultAction.Action != "block" {
		t.Fatalf(" action  block   : %q", defaultAction.Action)
	}
}
