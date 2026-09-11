package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func boolPtr(v bool) *bool { return &v }

func TestBuildConflictsDetectsCoreRules(t *testing.T) {
	prop := bson.M{"area": 100.0}
	records := []SourceRecord{
		{Source: "RoR", RecordID: "ROR-1", OwnerName: "Alice", SurveyNumber: "12/3", Area: 100, Latitude: 26.9, Longitude: 75.8, Encumbrance: boolPtr(false), Litigation: boolPtr(false)},
		{Source: "Registration", RecordID: "REG-1", OwnerName: "Bob", SurveyNumber: "12/4", Area: 120, Latitude: 26.9, Longitude: 75.8},
	}
	conflicts := buildConflicts(prop, records)
	seen := map[string]bool{}
	for _, c := range conflicts { seen[c.Code] = true }
	for _, code := range []string{"AREA_MISMATCH", "OWNERSHIP_MISMATCH", "SUBDIVISION_LINEAGE", "OVERLAP_SIGNAL", "MISSING_RISK_DATA"} {
		if !seen[code] { t.Fatalf("expected conflict %s, got %#v", code, conflicts) }
	}
}

func TestBuildConflictsClearWhenSourcesAgree(t *testing.T) {
	prop := bson.M{"area": 100.0}
	records := []SourceRecord{
		{Source: "RoR", RecordID: "ROR-1", OwnerName: "Alice", SurveyNumber: "12/3", Area: 100, Encumbrance: boolPtr(false), Litigation: boolPtr(false)},
		{Source: "Registration", RecordID: "REG-1", OwnerName: "Alice", SurveyNumber: "12/3", Area: 100, Encumbrance: boolPtr(false), Litigation: boolPtr(false)},
	}
	if got := buildConflicts(prop, records); len(got) != 0 { t.Fatalf("expected no conflicts, got %#v", got) }
}

func TestReportHashIsDeterministic(t *testing.T) {
	r := ReconciliationReport{PropertyID: "DHR-TEST", AlgorithmVersion: "reconciliation-v1", SourcesChecked: 2, Score: 80, Outcome: "REVIEW_REQUIRED"}
	a, err := canonicalReportHash(r)
	if err != nil { t.Fatal(err) }
	b, err := canonicalReportHash(r)
	if err != nil { t.Fatal(err) }
	if a != b { t.Fatalf("expected deterministic hash, got %s and %s", a, b) }
	if len(a) != 64 { t.Fatalf("expected SHA-256 hex length 64, got %d", len(a)) }
}

func TestReportScoreNeverNegative(t *testing.T) {
	conflicts := []Conflict{{Severity:"CRITICAL"},{Severity:"CRITICAL"},{Severity:"HIGH"},{Severity:"HIGH"},{Severity:"MEDIUM"}}
	if got := reportScore(conflicts); got < 0 { t.Fatalf("score must not be negative: %v", got) }
}
