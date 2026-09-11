package main

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "math"
    "sort"
    "strings"
    "time"

    "go.mongodb.org/mongo-driver/bson"
)

type SourceRecord struct {
    Source        string                 `json:"source" bson:"source"`
    RecordID      string                 `json:"record_id" bson:"record_id"`
    OwnerName     string                 `json:"owner_name,omitempty" bson:"owner_name,omitempty"`
    SurveyNumber  string                 `json:"survey_number,omitempty" bson:"survey_number,omitempty"`
    ParentSurvey  string                 `json:"parent_survey,omitempty" bson:"parent_survey,omitempty"`
    Area          float64                `json:"area,omitempty" bson:"area,omitempty"`
    Latitude      float64                `json:"latitude,omitempty" bson:"latitude,omitempty"`
    Longitude     float64                `json:"longitude,omitempty" bson:"longitude,omitempty"`
    Encumbrance   *bool                  `json:"encumbrance,omitempty" bson:"encumbrance,omitempty"`
    Litigation    *bool                  `json:"litigation,omitempty" bson:"litigation,omitempty"`
    Status        string                 `json:"status,omitempty" bson:"status,omitempty"`
    Metadata      map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

type Conflict struct {
    Code       string   `json:"code" bson:"code"`
    Severity   string   `json:"severity" bson:"severity"`
    Title      string   `json:"title" bson:"title"`
    Message    string   `json:"message" bson:"message"`
    Sources    []string `json:"sources,omitempty" bson:"sources,omitempty"`
}

type ReconciliationReport struct {
    PropertyID       string       `json:"property_id" bson:"property_id"`
    GeneratedAt      time.Time    `json:"generated_at" bson:"generated_at"`
    AlgorithmVersion string       `json:"algorithm_version" bson:"algorithm_version"`
    SourcesChecked   int          `json:"sources_checked" bson:"sources_checked"`
    Conflicts        []Conflict   `json:"conflicts" bson:"conflicts"`
    Score            float64      `json:"score" bson:"score"`
    Outcome          string       `json:"outcome" bson:"outcome"`
    ReportHash       string       `json:"report_hash" bson:"report_hash"`
    BlockchainHash   string       `json:"blockchain_hash" bson:"blockchain_hash"`
}

func normalizeString(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func buildConflicts(prop bson.M, sources []SourceRecord) []Conflict {
    var conflicts []Conflict
    if len(sources) == 0 { return conflicts }

    const areaTolerance = 0.01
    var baseArea float64
    if v, ok := prop["area"].(float64); ok { baseArea = v }
    var areaSources []string
    ownerSet := map[string][]string{}
    surveySet := map[string][]string{}
    var encMissing, litMissing bool
    for _, s := range sources {
        if s.Area > 0 { areaSources = append(areaSources, s.Source) }
        if s.OwnerName != "" { ownerSet[normalizeString(s.OwnerName)] = append(ownerSet[normalizeString(s.OwnerName)], s.Source) }
        if s.SurveyNumber != "" { surveySet[normalizeString(s.SurveyNumber)] = append(surveySet[normalizeString(s.SurveyNumber)], s.Source) }
        if s.Encumbrance == nil { encMissing = true }
        if s.Litigation == nil { litMissing = true }
    }

    if baseArea > 0 {
        for _, s := range sources {
            if s.Area > 0 && math.Abs(s.Area-baseArea) > areaTolerance {
                conflicts = append(conflicts, Conflict{"AREA_MISMATCH", "HIGH", "Area mismatch", fmt.Sprintf("Registered area %.2f differs from %s value %.2f", baseArea, s.Source, s.Area), []string{s.Source}})
            }
        }
    }
    if len(ownerSet) > 1 {
        keys := sortedKeys(ownerSet)
        var src []string
        for _, k := range keys { src = append(src, ownerSet[k]...) }
        conflicts = append(conflicts, Conflict{"OWNERSHIP_MISMATCH", "CRITICAL", "Ownership mismatch", "Source records disagree on the recorded owner", uniqueStrings(src)})
    }
    if len(surveySet) > 1 {
        keys := sortedKeys(surveySet)
        var src []string
        for _, k := range keys { src = append(src, surveySet[k]...) }
        conflicts = append(conflicts, Conflict{"SUBDIVISION_LINEAGE", "HIGH", "Survey lineage mismatch", "Source records disagree on survey/subdivision identity", uniqueStrings(src)})
    }
    if len(sources) >= 2 {
        for i := 0; i < len(sources); i++ {
            for j := i + 1; j < len(sources); j++ {
                if sources[i].Latitude != 0 && sources[i].Longitude != 0 && sources[j].Latitude != 0 && sources[j].Longitude != 0 {
                    dLat := sources[i].Latitude - sources[j].Latitude
                    dLon := sources[i].Longitude - sources[j].Longitude
                    if math.Sqrt(dLat*dLat+dLon*dLon) < 0.00001 && sources[i].Area > 0 && sources[j].Area > 0 && math.Abs(sources[i].Area-sources[j].Area) > areaTolerance {
                        conflicts = append(conflicts, Conflict{"OVERLAP_SIGNAL", "HIGH", "Possible spatial overlap", "Source geometries share a location but report materially different extents; GIS polygon intersection should be used for final confirmation", []string{sources[i].Source, sources[j].Source}})
                    }
                }
            }
        }
    }
    if encMissing || litMissing {
        var missing []string
        if encMissing { missing = append(missing, "encumbrance") }
        if litMissing { missing = append(missing, "litigation") }
        conflicts = append(conflicts, Conflict{"MISSING_RISK_DATA", "MEDIUM", "Incomplete risk evidence", "Missing " + strings.Join(missing, " and ") + " status in at least one source", areaSources})
    }
    return conflicts
}

func sortedKeys(m map[string][]string) []string { r:=make([]string,0,len(m)); for k:=range m { r=append(r,k) }; sort.Strings(r); return r }
func uniqueStrings(in []string) []string { seen:=map[string]bool{}; out:=[]string{}; for _,v:=range in { if v!=""&&!seen[v] {seen[v]=true;out=append(out,v)} }; sort.Strings(out); return out }

func reportScore(conflicts []Conflict) float64 {
    score := 100.0
    for _, c := range conflicts {
        switch c.Severity { case "CRITICAL": score -= 35; case "HIGH": score -= 20; case "MEDIUM": score -= 8; case "LOW": score -= 3 }
    }
    if score < 0 { score = 0 }; return math.Round(score*100)/100
}

func canonicalReportHash(r ReconciliationReport) (string, error) {
    r.ReportHash = ""; r.BlockchainHash = ""
    payload, err := json.Marshal(r); if err != nil { return "", err }
    sum := sha256.Sum256(payload)
    return hex.EncodeToString(sum[:]), nil
}
