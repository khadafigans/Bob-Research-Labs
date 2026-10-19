package notify

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
)

//
// 、 FuncMap  readFile，， diff 

// TestTemplateContextHasNoMethods 。
//
// text/template （{{.Foo}} ）。
//
// ： webhookTemplateData / webhookItem 。
func TestTemplateContextHasNoMethods(t *testing.T) {
	for _, v := range []any{webhookTemplateData{}, webhookItem{}} {
		typ := reflect.TypeOf(v)
		if n := typ.NumMethod(); n != 0 {
			var names []string
			for i := 0; i < n; i++ {
				names = append(names, typ.Method(i).Name)
			}
			t.Fatalf("%s  %d （%s）：text/template ，"+
				"", typ.Name(), n, strings.Join(names, ", "))
		}
	}
}

// TestTemplateFuncsAreMinimal 。
//
// FuncMap 。 json / jsons，
func TestTemplateFuncsAreMinimal(t *testing.T) {
	var got []string
	for name := range webhookTemplateFuncs {
		got = append(got, name)
	}
	sort.Strings(got)
	want := []string{"json", "jsons"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("： %v， %v。"+
			"（、、）", got, want)
	}
}

// TestTemplateCannotReachUnknownData ：
func TestTemplateCannotReachUnknownData(t *testing.T) {
	_, err := renderWebhookBody(`{"x": {{.Environment}}, "y": {{.Env}}}`, singleMsg())
	if err == nil {
		t.Fatal("")
	}
	for _, leak := range []string{"SQL", " id"} {
		if strings.Contains(err.Error(), leak) {
			t.Errorf(" %q: %v", leak, err)
		}
	}
}

// TestTemplateRenderFailsPermanently ，。
func TestTemplateRenderFailsPermanently(t *testing.T) {
	cfg := map[string]any{
		"url":           "https://example.com/hook",
		"body_template": `{{.Items.`,
	}
	if err := (webhookChannel{}).Validate(cfg); err == nil {
		t.Fatal("")
	}
	_, err := (webhookChannel{}).Send(context.Background(), cfg, singleMsg())
	if err == nil || !IsPermanent(err) {
		t.Fatalf("， %v", err)
	}
}

// TestTemplateCanOnlyProduceJSON 「 JSON」。
func TestTemplateCanOnlyProduceJSON(t *testing.T) {
	ok := map[string]any{"url": "https://example.com/hook", "body_template": `{"t":{{json .Title}}}`}
	if err := (webhookChannel{}).Validate(ok); err != nil {
		t.Fatalf(": %v", err)
	}
	bad := map[string]any{"url": "http://127.0.0.1:1/hook", "body_template": `not json {{.Count}}`}
	_, err := (webhookChannel{}).Send(context.Background(), bad, singleMsg())
	if err == nil || !IsPermanent(err) {
		t.Fatalf(" JSON ， %v", err)
	}
	if !strings.Contains(err.Error(), "valid JSON") {
		t.Errorf(" JSON ， %v", err)
	}
}
