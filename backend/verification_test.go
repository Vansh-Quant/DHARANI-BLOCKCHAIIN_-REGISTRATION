package main

import "testing"

func TestEvaluateVerificationRulesDetectsFiveMVPSignals(t *testing.T) {
	records := []SourceRecord{
		{SourceType:"ROR", OwnerName:"Alice Sharma", Area:100, SurveyNumber:"12/3", OverlapFlag:true, LineageStatus:"VALID", EncumbranceStatus:"CLEAR", LitigationStatus:"CLEAR"},
		{SourceType:"REGISTRATION", OwnerName:"Bob Sharma", Area:120, SurveyNumber:"12/4", LineageStatus:"BROKEN", EncumbranceStatus:"", LitigationStatus:"UNKNOWN"},
	}
	conflicts := evaluateVerificationRules(records)
	seen:=map[string]bool{}
	for _,c:=range conflicts {seen[c.RuleCode]=true}
	want:=[]string{"AREA_MISMATCH","OWNERSHIP_MISMATCH","DUPLICATE_OVERLAP","BROKEN_SUBDIVISION_LINEAGE","MISSING_RISK_DATA"}
	for _,code:=range want {if !seen[code]{t.Fatalf("missing %s in %#v",code,conflicts)}}
}

func TestEvaluateVerificationRulesClearForConsistentSources(t *testing.T) {
	records:=[]SourceRecord{
		{SourceType:"GIS",OwnerName:"Alice Sharma",Area:100,SurveyNumber:"12/3",LineageStatus:"VALID",EncumbranceStatus:"CLEAR",LitigationStatus:"CLEAR"},
		{SourceType:"ROR",OwnerName:" Alice   Sharma ",Area:100,SurveyNumber:"12/3",LineageStatus:"VALID",EncumbranceStatus:"CLEAR",LitigationStatus:"CLEAR"},
		{SourceType:"REGISTRATION",OwnerName:"Alice Sharma",Area:100,SurveyNumber:"12/3",LineageStatus:"VALID",EncumbranceStatus:"CLEAR",LitigationStatus:"CLEAR"},
	}
	if got:=evaluateVerificationRules(records);len(got)!=0{t.Fatalf("expected no conflicts, got %#v",got)}
}

func TestVerificationScoreBounds(t *testing.T) {
	if got:=verificationScore(3,0);got!=100{t.Fatalf("expected 100, got %v",got)}
	if got:=verificationScore(2,10);got<0||got>100{t.Fatalf("score out of bounds: %v",got)}
	if got:=verificationScore(0,0);got!=0{t.Fatalf("expected zero for no sources, got %v",got)}
}

func TestNormalizeText(t *testing.T) {
	if got:=normalizeText("  Alice   SHARMA  ");got!="alice sharma"{t.Fatalf("unexpected normalization: %q",got)}
}
