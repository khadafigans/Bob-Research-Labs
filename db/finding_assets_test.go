package db

import (
	"strconv"
	"strings"
	"testing"
)

// cleanupTreeFixtures 。 defer (
// t.Cleanup):t.Cleanup , defer d.Close() ,
func cleanupTreeFixtures(d *DB, taskID int64, rootDomains ...string) {
	d.Exec(`DELETE FROM assets WHERE root_domain = ANY($1::text[])`, rootDomains) //nolint:errcheck
	d.DeleteFindingsByTask(taskID)                                                //nolint:errcheck
}

// seedTreeAsset inserts one asset row.
func seedTreeAsset(t *testing.T, d *DB, kind string, cols map[string]any) int64 {
	t.Helper()
	names := []string{"type"}
	values := []any{kind}
	placeholders := []string{"$1"}
	for k, v := range cols {
		values = append(values, v)
		names = append(names, k)
		placeholders = append(placeholders, "$"+strconv.Itoa(len(values)))
	}
	q := "INSERT INTO assets(" + strings.Join(names, ",") + ") VALUES (" +
		strings.Join(placeholders, ",") + ") RETURNING id"
	var id int64
	if err := d.QueryRow(q, values...).Scan(&id); err != nil {
		t.Fatalf("seed %s asset: %v", kind, err)
	}
	return id
}

func nodeByKey(tree *FindingAssetTree, key string) *FindingAssetNode {
	for i := range tree.Nodes {
		if tree.Nodes[i].Key == key {
			return &tree.Nodes[i]
		}
	}
	return nil
}

// TestBuildFindingAssetTree covers the whole shape of the「」tree: the
// root→subdomain→service→endpoint chain gets rebuilt from a finding that only
// points at the leaf, ancestors aggregate their subtree, assets without any
// finding stay out, and a finding whose asset row is gone lands in the
// unassigned bucket.
func TestBuildFindingAssetTree(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	tk, err := d.CreateTask("", "", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(tk.ID)

	const root = "tree-test.example"
	const sub = "api.tree-test.example"
	defer cleanupTreeFixtures(d, tk.ID, root)
	rootID := seedTreeAsset(t, d, "root_domain", map[string]any{"domain": root, "root_domain": root})
	subID := seedTreeAsset(t, d, "subdomain", map[string]any{"domain": sub, "root_domain": root})
	svcID := seedTreeAsset(t, d, "service", map[string]any{
		"domain": sub, "root_domain": root, "url": "https://" + sub, "port": 443, "service_type": "http",
	})
	epID := seedTreeAsset(t, d, "endpoint", map[string]any{
		"domain": sub, "root_domain": root, "url": "https://" + sub + "/admin", "port": 443, "method": "GET",
	})
	seedTreeAsset(t, d, "service", map[string]any{
		"domain": sub, "root_domain": root, "url": "http://" + sub + ":8080", "port": 8080, "service_type": "http",
	})

	if _, err := d.AddFinding(tk.ID, 0, "XSS", " XSS", "high", "s", "e", "w", []int64{epID}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddFinding(tk.ID, 0, "Info", "", "low", "s", "e", "w", []int64{svcID}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddFinding(tk.ID, 0, "Misc", "", "medium", "s", "e", "w", []int64{999000111}); err != nil {
		t.Fatal(err)
	}

	tree, err := d.BuildFindingAssetTree(FindingFilter{TaskID: strconv.FormatInt(tk.ID, 10)})
	if err != nil {
		t.Fatal(err)
	}
	if tree.FindingTotal != 3 {
		t.Fatalf("finding_total: want 3, got %d", tree.FindingTotal)
	}

	rootNode := nodeByKey(tree, assetKey(rootID))
	subNode := nodeByKey(tree, assetKey(subID))
	svcNode := nodeByKey(tree, assetKey(svcID))
	epNode := nodeByKey(tree, assetKey(epID))
	for name, n := range map[string]*FindingAssetNode{
		"root": rootNode, "subdomain": subNode, "service": svcNode, "endpoint": epNode,
	} {
		if n == nil {
			t.Fatalf("%s node missing from tree", name)
		}
	}

	// :endpoint → service → subdomain → root_domain。
	if epNode.Parent != svcNode.Key {
		t.Errorf("endpoint parent: want %s, got %s", svcNode.Key, epNode.Parent)
	}
	if svcNode.Parent != subNode.Key {
		t.Errorf("service parent: want %s, got %s", subNode.Key, svcNode.Parent)
	}
	if subNode.Parent != rootNode.Key {
		t.Errorf("subdomain parent: want %s, got %s", rootNode.Key, subNode.Parent)
	}
	if rootNode.Parent != "" {
		t.Errorf("root parent: want top level, got %s", rootNode.Parent)
	}

	// :(endpoint  high + service  low),service 、。
	if rootNode.Total != 2 || rootNode.High != 1 || rootNode.Low != 1 {
		t.Errorf("root totals: want 2/high1/low1, got %d/high%d/low%d", rootNode.Total, rootNode.High, rootNode.Low)
	}
	if rootNode.Self != 0 {
		t.Errorf("root self: want 0 (), got %d", rootNode.Self)
	}
	if svcNode.Total != 2 || svcNode.Self != 1 {
		t.Errorf("service total/self: want 2/1, got %d/%d", svcNode.Total, svcNode.Self)
	}
	if epNode.Total != 1 || epNode.Self != 1 {
		t.Errorf("endpoint total/self: want 1/1, got %d/%d", epNode.Total, epNode.Self)
	}

	for _, n := range tree.Nodes {
		if n.Label == "http://"+sub+":8080" {
			t.Errorf("asset without findings should be hidden: %+v", n)
		}
	}

	none := nodeByKey(tree, FindingUnassignedAsset)
	if none == nil || none.Total != 1 || none.Medium != 1 {
		t.Fatalf("unassigned bucket: want 1 medium, got %+v", none)
	}
}

// TestFindingAssetScopeFilter verifies narrows the findings list to
// that node's whole subtree, and that the unassigned sentinel works too.
func TestFindingAssetScopeFilter(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	tk, err := d.CreateTask("", "", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(tk.ID)

	const root = "scope-test.example"
	const sub = "api.scope-test.example"
	const other = "other-scope-test.example"
	defer cleanupTreeFixtures(d, tk.ID, root, other)
	rootID := seedTreeAsset(t, d, "root_domain", map[string]any{"domain": root, "root_domain": root})
	subID := seedTreeAsset(t, d, "subdomain", map[string]any{"domain": sub, "root_domain": root})
	otherID := seedTreeAsset(t, d, "root_domain", map[string]any{"domain": other, "root_domain": other})

	if _, err := d.AddFinding(tk.ID, 0, "A", "", "high", "s", "e", "w", []int64{subID}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddFinding(tk.ID, 0, "B", "", "high", "s", "e", "w", []int64{otherID}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddFinding(tk.ID, 0, "C", "", "high", "s", "e", "w", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := d.AddFinding(tk.ID, 0, "D", "", "high", "s", "e", "w", []int64{999000333}); err != nil {
		t.Fatal(err)
	}

	base := FindingFilter{TaskID: strconv.FormatInt(tk.ID, 10)}
	cases := []struct {
		name  string
		scope string
		want  int
	}{
		{"", assetKey(rootID), 1},      // 
		{"", assetKey(subID), 1},       // 
		{"", assetKey(otherID), 1},     // 
		{"", FindingUnassignedAsset, 2}, // asset_ids  + 
		{"", "a:999000222", 0},       //  → ,
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := base
			f.AssetScope = tc.scope
			items, total, err := d.ListFindingsPage(f, 1, 50)
			if err != nil {
				t.Fatal(err)
			}
			if total != tc.want || len(items) != tc.want {
				t.Fatalf("scope %s: want %d findings, got total=%d items=%d", tc.scope, tc.want, total, len(items))
			}
		})
	}

	if _, total, err := d.ListFindingsPage(base, 1, 50); err != nil || total != 4 {
		t.Fatalf("unscoped: want 4, got %d (%v)", total, err)
	}

	tree, err := d.BuildFindingAssetTree(base)
	if err != nil {
		t.Fatal(err)
	}
	none := nodeByKey(tree, FindingUnassignedAsset)
	if none == nil || none.Total != 2 {
		t.Fatalf("unassigned bucket count: want 2, got %+v", none)
	}
}
