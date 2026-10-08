package services

import "testing"

func TestManufacturingManifestUsesWorkflowNavigation(t *testing.T) {
	manifest := ManufacturingManifest()
	if len(manifest.MenuExtensions) != 1 {
		t.Fatalf("expected one manufacturing root menu, got %#v", manifest.MenuExtensions)
	}
	root := manifest.MenuExtensions[0]
	if root["key"] != "manufacturing" || root["group"] != "manufacturing" {
		t.Fatalf("expected standalone manufacturing menu, got %#v", root)
	}
	if root["navMode"] != "workflow" {
		t.Fatalf("expected workflow nav mode, got %#v", root["navMode"])
	}
	children, ok := root["children"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected children slice, got %#v", root["children"])
	}
	if len(children) != 4 {
		t.Fatalf("expected 4 beginner workflow menus, got %d: %#v", len(children), children)
	}

	expected := map[string]string{
		"/manufacturing/quote-to-order":  "quote_to_order",
		"/manufacturing/sample-to-mass":  "sample_to_mass",
		"/manufacturing/delivery-change": "delivery_change",
		"/manufacturing/settings":        "settings",
	}
	for _, child := range children {
		path, _ := child["path"].(string)
		group, _ := child["workflowGroup"].(string)
		if expected[path] != group {
			t.Fatalf("unexpected workflow child path/group: %#v", child)
		}
		if child["beginnerVisible"] != true {
			t.Fatalf("workflow child should be beginner visible: %#v", child)
		}
		if _, ok := child["legacyModules"].([]string); !ok {
			t.Fatalf("workflow child must keep legacy module mappings: %#v", child)
		}
	}
}
