package main

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "sort"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type SourceRecord struct {
    PropertyID string `json:"property_id" bson:"property_id"`
    SourceType string `json:"source_type" bson:"source_type"`
    SourceRecordID string `json:"source_record_id" bson:"source_record_id"`
    ULPIN string `json:"ulpin,omitempty" bson:"ulpin,omitempty"`
    SurveyNumber string `json:"survey_number,omitempty" bson:"survey_number,omitempty"`
    OwnerName string `json:"owner_name,omitempty" bson:"owner_name,omitempty"`
    Area float64 `json:"area,omitempty" bson:"area,omitempty"`
    PolygonHash string `json:"polygon_hash,omitempty" bson:"polygon_hash,omitempty"`
    ParentSurvey string `json:"parent_survey,omitempty" bson:"parent_survey,omitempty"`
    LineageStatus string `json:"lineage_status,omitempty" bson:"lineage_status,omitempty"`
    EncumbranceStatus string `json:"encumbrance_status,omitempty" bson:"encumbrance_status,omitempty"`
    LitigationStatus string `json:"litigation_status,omitempty" bson:"litigation_status,omitempty"`
    OverlapFlag bool `json:"overlap_flag,omitempty" bson:"overlap_flag,omitempty"`
    DocumentHash string `json:"document_hash,omitempty" bson:"document_hash,omitempty"`
    CapturedAt time.Time `json:"captured_at" bson:"captured_at"`
}

type Conflict struct {
    RuleCode string `json:"rule_code" bson:"rule_code"`
    Severity string `json:"severity" bson:"severity"`
    Title string `json:"title" bson:"title"`
    Description string `json:"description" bson:"description"`
    Sources []string `json:"sources,omitempty" bson:"sources,omitempty"`
}

type VerificationReport struct {
    ReportVersion string `json:"report_version" bson:"report_version"`
    PropertyID string `json:"property_id" bson:"property_id"`
    GeneratedAt time.Time `json:"generated_at" bson:"generated_at"`
    SourceCount int `json:"source_count" bson:"source_count"`
    Sources []SourceRecord `json:"sources" bson:"sources"`
    Conflicts []Conflict `json:"conflicts" bson:"conflicts"`
    Status string `json:"status" bson:"status"`
    Summary string `json:"summary" bson:"summary"`
    SHA256 string `json:"sha256" bson:"sha256"`
}

func registerVerificationRoutes(protected *gin.RouterGroup) {
    protected.POST("/source-records", createSourceRecord)
    protected.GET("/source-records/:propertyId", getSourceRecords)
    protected.POST("/reconciliation/:propertyId/run", runVerification)
    protected.GET("/reconciliation/:propertyId/report", getVerificationReport)
    protected.POST("/blockchain/anchor-result", recordBlockchainAnchor)
    protected.GET("/blockchain/:propertyId/proof", getBlockchainProof)
}

func sourceAuthorityRequired(c *gin.Context) bool {
    role := getRole(currentPhone(c))
    if role != "AUTHORITY" && role != "ADMIN" {
        c.JSON(http.StatusForbidden, gin.H{"error": "Authority role required"})
        return false
    }
    return true
}

func createSourceRecord(c *gin.Context) {
    if !sourceAuthorityRequired(c) { return }
    var record SourceRecord
    if err := c.ShouldBindJSON(&record); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source record", "details": err.Error()})
        return
    }
    record.PropertyID = strings.TrimSpace(record.PropertyID)
    record.SourceType = strings.ToUpper(strings.TrimSpace(record.SourceType))
    record.SourceRecordID = strings.TrimSpace(record.SourceRecordID)
    record.ULPIN = strings.TrimSpace(record.ULPIN)
    record.SurveyNumber = strings.TrimSpace(record.SurveyNumber)
    record.OwnerName = strings.TrimSpace(record.OwnerName)
    allowed := map[string]bool{"REVENUE": true, "ROR": true, "REGISTRATION": true, "GIS": true, "ENCUMBRANCE": true, "COURT": true}
    if record.PropertyID == "" || record.SourceType == "" || record.SourceRecordID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "property_id, source_type and source_record_id are required"})
        return
    }
    if !allowed[record.SourceType] {
        c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported source_type"})
        return
    }
    if record.Area < 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "area cannot be negative"})
        return
    }
    if record.CapturedAt.IsZero() { record.CapturedAt = time.Now().UTC() }
    if err := db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": record.PropertyID}).Err(); err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
        return
    }
    _, err := db.Collection("source_records").UpdateOne(context.Background(), bson.M{"property_id": record.PropertyID, "source_type": record.SourceType}, bson.M{"$set": record}, options.Update().SetUpsert(true))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save source record"})
        return
    }
    _, _ = db.Collection("properties").UpdateOne(context.Background(), bson.M{"property_id": record.PropertyID}, bson.M{"$set": bson.M{"verification_status": "UNDER_REVIEW", "updated_at": time.Now()}})
    c.JSON(http.StatusOK, gin.H{"message": "Source record saved", "record": record})
}

func getSourceRecords(c *gin.Context) {
    propertyID := strings.TrimSpace(c.Param("propertyId"))
    cur, err := db.Collection("source_records").Find(context.Background(), bson.M{"property_id": propertyID}, options.Find().SetSort(bson.D{{Key: "source_type", Value: 1}}))
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"}); return }
    defer cur.Close(context.Background())
    var records []SourceRecord
    if err := cur.All(context.Background(), &records); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read source records"}); return }
    c.JSON(http.StatusOK, gin.H{"property_id": propertyID, "records": records, "count": len(records)})
}

func runVerification(c *gin.Context) {
    if !sourceAuthorityRequired(c) { return }
    propertyID := strings.TrimSpace(c.Param("propertyId"))
    if err := db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": propertyID}).Err(); err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"}); return }
    cur, err := db.Collection("source_records").Find(context.Background(), bson.M{"property_id": propertyID})
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load source records"}); return }
    defer cur.Close(context.Background())
    var records []SourceRecord
    if err := cur.All(context.Background(), &records); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not decode source records"}); return }
    sort.Slice(records, func(i, j int) bool { return records[i].SourceType < records[j].SourceType })
    conflicts := evaluateVerificationRules(records)
    status, summary := "CLEAR", "All supplied source records are consistent under the configured MVP rules."
    if len(records) < 2 { status, summary = "INCONCLUSIVE", "At least two independent source records are required for cross-source reconciliation." } else if len(conflicts) > 0 { status, summary = "CONFLICT", fmt.Sprintf("%d verification conflict(s) require authority review.", len(conflicts)) }
    report := VerificationReport{ReportVersion: "1.1", PropertyID: propertyID, GeneratedAt: time.Now().UTC(), SourceCount: len(records), Sources: records, Conflicts: conflicts, Status: status, Summary: summary}
    canonical := report; canonical.SHA256 = ""
    payload, err := json.Marshal(canonical)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not canonicalize verification report"}); return }
    sum := sha256.Sum256(payload); report.SHA256 = hex.EncodeToString(sum[:])
    if _, err = db.Collection("verification_reports").InsertOne(context.Background(), report); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save verification report"}); return }
    score := verificationScore(len(records), len(conflicts))
    update := bson.M{"verification_status": "UNDER_REVIEW", "verification_score": score, "conflicts": conflicts, "updated_at": time.Now()}
    if status == "CLEAR" {
        update["latest_verification_report_hash"] = report.SHA256
        update["latest_verification_status"] = "CLEAR"
        update["latest_verification_at"] = report.GeneratedAt
    } else {
        update["verification_status"] = status
        update["latest_verification_status"] = status
        update["$unset"] = bson.M{"latest_verification_report_hash": "", "latest_verification_at": ""}
    }
    if _, err = db.Collection("properties").UpdateOne(context.Background(), bson.M{"property_id": propertyID}, bson.M{"$set": update, "$unset": update["$unset"]}); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update property verification state"})
        return
    }
    _, _ = db.Collection("audit_events").InsertOne(context.Background(), bson.M{"property_id": propertyID, "event_type": "VERIFICATION_RUN", "actor": currentPhone(c), "report_hash": report.SHA256, "status": status, "created_at": report.GeneratedAt})
    c.JSON(http.StatusOK, gin.H{"report": report, "property_status": status})
}

func getVerificationReport(c *gin.Context) {
    propertyID := strings.TrimSpace(c.Param("propertyId"))
    var report VerificationReport
    if err := db.Collection("verification_reports").FindOne(context.Background(), bson.M{"property_id": propertyID}, options.FindOne().SetSort(bson.D{{Key: "generated_at", Value: -1}})).Decode(&report); err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "Verification report not found"}); return }
    c.JSON(http.StatusOK, gin.H{"report": report})
}

func evaluateVerificationRules(records []SourceRecord) []Conflict {
    conflicts := make([]Conflict, 0)
    if len(records) < 2 { return conflicts }
    minArea, maxArea := records[0].Area, records[0].Area
    areaSources := make([]string, 0, len(records))
    for _, r := range records { if r.Area > 0 { if r.Area < minArea { minArea = r.Area }; if r.Area > maxArea { maxArea = r.Area }; areaSources = append(areaSources, r.SourceType) } }
    if minArea > 0 && maxArea > 0 && (maxArea-minArea)/minArea > 0.01 { conflicts = append(conflicts, Conflict{RuleCode: "AREA_MISMATCH", Severity: "HIGH", Title: "Extent mismatch", Description: fmt.Sprintf("Source areas range from %.2f to %.2f.", minArea, maxArea), Sources: areaSources}) }

    compareField := func(rule, title, description string, values map[string]string) { if len(uniqueValues(values)) > 1 { conflicts = append(conflicts, Conflict{RuleCode: rule, Severity: "HIGH", Title: title, Description: description, Sources: mapKeys(values)}) } }
    owners := map[string]string{}; ulpins := map[string]string{}; surveys := map[string]string{}
    for _, r := range records {
        if r.OwnerName != "" { owners[r.SourceType] = normalizeText(r.OwnerName) }
        if r.ULPIN != "" { ulpins[r.SourceType] = normalizeText(r.ULPIN) }
        if r.SurveyNumber != "" { surveys[r.SourceType] = normalizeText(r.SurveyNumber) }
    }
    compareField("OWNERSHIP_MISMATCH", "Ownership mismatch", "Normalized owner names differ across supplied sources.", owners)
    compareField("ULPIN_MISMATCH", "ULPIN mismatch", "Supplied ULPIN identifiers differ across sources.", ulpins)
    compareField("SURVEY_NUMBER_MISMATCH", "Survey number mismatch", "Supplied survey numbers differ across sources.", surveys)

    var overlapSources []string
    for _, r := range records { if r.OverlapFlag { overlapSources = append(overlapSources, r.SourceType) } }
    if len(overlapSources) > 0 { conflicts = append(conflicts, Conflict{RuleCode: "DUPLICATE_OVERLAP", Severity: "HIGH", Title: "Parcel overlap detected", Description: "At least one source explicitly flags a duplicate or spatial overlap risk.", Sources: overlapSources}) }
    var lineageSources []string
    for _, r := range records { s := strings.ToUpper(strings.TrimSpace(r.LineageStatus)); if s == "BROKEN" || s == "MISSING" || s == "INVALID" { lineageSources = append(lineageSources, r.SourceType) } }
    if len(lineageSources) > 0 { conflicts = append(conflicts, Conflict{RuleCode: "BROKEN_SUBDIVISION_LINEAGE", Severity: "MEDIUM", Title: "Subdivision lineage issue", Description: "A source reports missing or broken parent/subdivision lineage.", Sources: lineageSources}) }
    var missingSources []string
    for _, r := range records { enc, lit := strings.ToUpper(strings.TrimSpace(r.EncumbranceStatus)), strings.ToUpper(strings.TrimSpace(r.LitigationStatus)); if enc == "" || lit == "" || enc == "UNKNOWN" || lit == "UNKNOWN" || enc == "MISSING" || lit == "MISSING" { missingSources = append(missingSources, r.SourceType) } }
    if len(missingSources) > 0 { conflicts = append(conflicts, Conflict{RuleCode: "MISSING_RISK_DATA", Severity: "MEDIUM", Title: "Encumbrance/litigation data incomplete", Description: "One or more sources do not provide a usable encumbrance or litigation status.", Sources: missingSources}) }
    return conflicts
}

func verificationScore(sourceCount, conflictCount int) float64 { if sourceCount == 0 { return 0 }; score := 100.0 - float64(conflictCount)*18; if sourceCount < 3 { score -= 10 }; if score < 0 { score = 0 }; return score }
func normalizeText(value string) string { return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ") }
func uniqueValues(values map[string]string) []string { set := make(map[string]struct{}); for _, value := range values { if value != "" { set[value] = struct{}{} } }; result := make([]string, 0, len(set)); for value := range set { result = append(result, value) }; sort.Strings(result); return result }
func mapKeys(values map[string]string) []string { keys := make([]string, 0, len(values)); for key := range values { keys = append(keys, key) }; sort.Strings(keys); return keys }
