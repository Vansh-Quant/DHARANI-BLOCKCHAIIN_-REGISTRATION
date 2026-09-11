package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client
var db *mongo.Database
var sessions = map[string]string{}
var sessionMu sync.RWMutex

func main() {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" { mongoURI = "mongodb://localhost:27017" }
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil { log.Fatal(err) }
	mongoClient = client
	defer mongoClient.Disconnect(context.Background())
	if err = client.Ping(context.Background(), nil); err != nil { log.Fatalf("MongoDB unavailable: %v", err) }
	db = client.Database("dharani")
	ensureIndexes()

	r := gin.Default()
	r.Use(cors())
	api := r.Group("/api/v1")
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "service": "dharani-api", "time": time.Now()}) })
	api.POST("/auth/register", registerUser)
	api.POST("/auth/verify-otp", verifyOTP)

	protected := api.Group("")
	protected.Use(authMiddleware())
	protected.POST("/property/register", registerProperty)
	protected.GET("/property/user/list", getUserProperties)
	protected.GET("/property/all", getAllProperties)
	protected.GET("/property/search/query", searchProperties)
	protected.GET("/property/:id", getPropertyDetail)
	protected.GET("/property/:id/passport", getPropertyPassport)
	protected.POST("/property/:id/documents", addDocument)
	registerVerificationRoutes(protected)
	protected.GET("/verification/queue", getVerificationQueue)
	protected.POST("/verification/:id/review", reviewVerification)
	protected.POST("/transfer/initiate", initiateTransfer)
	protected.GET("/transfer/incoming", getIncomingTransfers)
	protected.GET("/transfer/outgoing", getOutgoingTransfers)
	protected.POST("/transfer/:id/accept", acceptTransfer)
	protected.POST("/transfer/:id/reject", rejectTransfer)

	port := os.Getenv("API_PORT")
	if port == "" { port = "8080" }
	log.Printf("Dharani API running on http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil { log.Fatal(err) }
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == "OPTIONS" { c.AbortWithStatus(204); return }
		c.Next()
	}
}

func ensureIndexes() {
	ctx := context.Background()
	_, _ = db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.M{"phone_number": 1}, Options: options.Index().SetUnique(true)})
	_, _ = db.Collection("properties").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "property_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "address", Value: "text"}, {Key: "property_name", Value: "text"}, {Key: "location", Value: "text"}}},
		{Keys: bson.D{{Key: "owner_phone", Value: 1}}},
		{Keys: bson.D{{Key: "verification_status", Value: 1}}},
	})
	_, _ = db.Collection("transfers").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.M{"transfer_id": 1}, Options: options.Index().SetUnique(true)})
	_, _ = db.Collection("source_records").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "property_id", Value: 1}, {Key: "source_type", Value: 1}}, Options: options.Index().SetUnique(true)})
	_, _ = db.Collection("verification_reports").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "property_id", Value: 1}, {Key: "generated_at", Value: -1}}})
	_, _ = db.Collection("audit_events").Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "property_id", Value: 1}, {Key: "created_at", Value: -1}}})
}

func randomToken() string { b := make([]byte, 24); _, _ = rand.Read(b); return hex.EncodeToString(b) }

func registerUser(c *gin.Context) {
	var req struct { PhoneNumber string `json:"phone_number"`; FirstName string `json:"first_name"`; LastName string `json:"last_name"`; Email string `json:"email"` }
	if c.BindJSON(&req) != nil || strings.TrimSpace(req.PhoneNumber) == "" { c.JSON(400, gin.H{"error": "Valid phone number required"}); return }
	role := "CITIZEN"
	if strings.EqualFold(strings.TrimSpace(req.Email), "authority@dharani.local") || strings.EqualFold(strings.TrimSpace(req.Email), "admin@dharani.local") { role = "AUTHORITY" }
	_, _ = db.Collection("users").UpdateOne(context.Background(), bson.M{"phone_number": req.PhoneNumber}, bson.M{"$set": bson.M{"phone_number": req.PhoneNumber, "first_name": req.FirstName, "last_name": req.LastName, "email": req.Email, "role": role, "updated_at": time.Now()}, "$setOnInsert": bson.M{"created_at": time.Now()}}, options.Update().SetUpsert(true))
	c.JSON(200, gin.H{"message": "Demo OTP generated. Connect an SMS provider before production.", "demo_otp": "1234"})
}

func verifyOTP(c *gin.Context) {
	var req struct { PhoneNumber string `json:"phone_number"`; OTP string `json:"otp"` }
	if c.BindJSON(&req) != nil { c.JSON(400, gin.H{"error": "Invalid request"}); return }
	if req.OTP != "1234" { c.JSON(401, gin.H{"error": "Invalid demo OTP"}); return }
	var user bson.M
	if err := db.Collection("users").FindOne(context.Background(), bson.M{"phone_number": req.PhoneNumber}).Decode(&user); err != nil { c.JSON(404, gin.H{"error": "User not registered"}); return }
	token := randomToken()
	sessionMu.Lock(); sessions[token] = req.PhoneNumber; sessionMu.Unlock()
	c.JSON(200, gin.H{"user": user, "token": token})
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") { c.JSON(401, gin.H{"error": "Bearer token required"}); c.Abort(); return }
		token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer"))
		sessionMu.RLock(); phone, ok := sessions[token]; sessionMu.RUnlock()
		if !ok { c.JSON(401, gin.H{"error": "Session expired or invalid"}); c.Abort(); return }
		c.Set("phone", phone)
		c.Next()
	}
}

func currentPhone(c *gin.Context) string { v, _ := c.Get("phone"); return v.(string) }
func makePropertyID() string { return "DHR-" + strings.ToUpper(randomToken()[:10]) }
func makeQRPayload(id string) string { return "dharani://property/" + id }

func registerProperty(c *gin.Context) {
	var req struct { PropertyName string `json:"property_name"`; PropertyType string `json:"property_type"`; Location string `json:"location"`; Address string `json:"address"`; Area float64 `json:"area"`; Latitude float64 `json:"latitude"`; Longitude float64 `json:"longitude"` }
	if c.BindJSON(&req) != nil || req.PropertyName == "" || req.Address == "" || req.Area <= 0 { c.JSON(400, gin.H{"error": "property_name, address and positive area are required"}); return }
	id := makePropertyID()
	doc := bson.M{"property_id": id, "property_name": req.PropertyName, "property_type": req.PropertyType, "location": req.Location, "address": req.Address, "area": req.Area, "latitude": req.Latitude, "longitude": req.Longitude, "owner_phone": currentPhone(c), "verification_status": "SUBMITTED", "verification_score": 0, "documents": []interface{}{}, "conflicts": []interface{}{}, "created_at": time.Now(), "updated_at": time.Now()}
	if _, err := db.Collection("properties").InsertOne(context.Background(), doc); err != nil { c.JSON(500, gin.H{"error": "Could not save property"}); return }
	c.JSON(201, gin.H{"property_id": id, "qr_payload": makeQRPayload(id), "verification_status": "SUBMITTED", "message": "Property submitted for verification"})
}

func getUserProperties(c *gin.Context) {
	cur, err := db.Collection("properties").Find(context.Background(), bson.M{"owner_phone": currentPhone(c)}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil { c.JSON(500, gin.H{"error": "Database error"}); return }
	defer cur.Close(context.Background())
	var docs []bson.M
	if cur.All(context.Background(), &docs) != nil { c.JSON(500, gin.H{"error": "Database error"}); return }
	c.JSON(200, gin.H{"properties": docs, "count": len(docs)})
}

func getAllProperties(c *gin.Context) {
	limit := int64(20); if v := c.Query("limit"); v != "" { fmt.Sscan(v, &limit) }; if limit < 1 || limit > 100 { limit = 20 }
	cur, err := db.Collection("properties").Find(context.Background(), bson.M{}, options.Find().SetLimit(limit).SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil { c.JSON(500, gin.H{"error": "Database error"}); return }
	defer cur.Close(context.Background())
	var docs []bson.M; _ = cur.All(context.Background(), &docs)
	total, _ := db.Collection("properties").CountDocuments(context.Background(), bson.M{})
	c.JSON(200, gin.H{"properties": docs, "total": total})
}

func searchProperties(c *gin.Context) {
	q := strings.TrimSpace(c.Query("query")); if q == "" { c.JSON(200, gin.H{"properties": []bson.M{}, "count": 0}); return }
	filter := bson.M{"$text": bson.M{"$search": q}}
	cur, err := db.Collection("properties").Find(context.Background(), filter, options.Find().SetLimit(30))
	if err != nil { rx := primitive.Regex{Pattern: q, Options: "i"}; cur, err = db.Collection("properties").Find(context.Background(), bson.M{"$or": []bson.M{{"property_name": rx}, {"address": rx}, {"location": rx}, {"property_id": rx}}}, options.Find().SetLimit(30)) }
	if err != nil { c.JSON(500, gin.H{"error": "Search failed"}); return }
	defer cur.Close(context.Background())
	var docs []bson.M; _ = cur.All(context.Background(), &docs)
	c.JSON(200, gin.H{"properties": docs, "count": len(docs), "query": q})
}

func getPropertyDetail(c *gin.Context) { var doc bson.M; if err := db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": c.Param("id")}).Decode(&doc); err != nil { c.JSON(404, gin.H{"error": "Property not found"}); return }; c.JSON(200, doc) }

func getPropertyPassport(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var doc bson.M
	if err := db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": id}).Decode(&doc); err != nil { c.JSON(404, gin.H{"error": "Property not found"}); return }
	doc["qr_payload"] = makeQRPayload(id)
	var report VerificationReport
	if err := db.Collection("verification_reports").FindOne(context.Background(), bson.M{"property_id": id}, options.FindOne().SetSort(bson.D{{Key: "generated_at", Value: -1}})).Decode(&report); err == nil { doc["verification_report"] = report }
	if proof, ok := doc["blockchain_proof"]; ok { doc["blockchain_proof"] = proof }
	cur, _ := db.Collection("audit_events").Find(context.Background(), bson.M{"property_id": id}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50))
	if cur != nil { defer cur.Close(context.Background()); var events []bson.M; _ = cur.All(context.Background(), &events); doc["audit_history"] = events }
	c.JSON(200, gin.H{"passport": doc})
}

func addDocument(c *gin.Context) {
	id := c.Param("id")
	var req struct { Name string `json:"name"`; Type string `json:"type"`; Hash string `json:"hash"`; Extracted map[string]interface{} `json:"extracted"`; Confidence float64 `json:"confidence"` }
	if c.BindJSON(&req) != nil || req.Name == "" { c.JSON(400, gin.H{"error": "document name required"}); return }
	r := bson.M{"name": req.Name, "type": req.Type, "hash": req.Hash, "extracted": req.Extracted, "confidence": req.Confidence, "submitted_at": time.Now()}
	res, err := db.Collection("properties").UpdateOne(context.Background(), bson.M{"property_id": id, "owner_phone": currentPhone(c)}, bson.M{"$push": bson.M{"documents": r}, "$set": bson.M{"verification_status": "UNDER_REVIEW", "updated_at": time.Now()}})
	if err != nil || res.MatchedCount == 0 { c.JSON(404, gin.H{"error": "Property not found or not owned by current user"}); return }
	c.JSON(200, gin.H{"message": "Document added and property moved to review", "document": r})
}

func getVerificationQueue(c *gin.Context) {
	role := getRole(currentPhone(c)); if role != "AUTHORITY" && role != "ADMIN" { c.JSON(403, gin.H{"error": "Authority role required"}); return }
	filter := bson.M{"verification_status": bson.M{"$in": []string{"SUBMITTED", "UNDER_REVIEW", "CONFLICT", "INCONCLUSIVE"}}}
	cur, err := db.Collection("properties").Find(context.Background(), filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil { c.JSON(500, gin.H{"error": "Database error"}); return }
	defer cur.Close(context.Background())
	var docs []bson.M; _ = cur.All(context.Background(), &docs)
	c.JSON(200, gin.H{"queue": docs, "count": len(docs)})
}

func reviewVerification(c *gin.Context) {
	role := getRole(currentPhone(c)); if role != "AUTHORITY" && role != "ADMIN" { c.JSON(403, gin.H{"error": "Authority role required"}); return }
	var req struct { Decision string `json:"decision"`; Notes string `json:"notes"` }
	if c.BindJSON(&req) != nil { c.JSON(400, gin.H{"error": "Invalid review"}); return }
	decision := strings.ToUpper(strings.TrimSpace(req.Decision))
	if decision != "VERIFIED" && decision != "CONFLICT" && decision != "REJECTED" { c.JSON(400, gin.H{"error": "decision must be VERIFIED, CONFLICT or REJECTED"}); return }
	id := strings.TrimSpace(c.Param("id"))
	var p bson.M
	if err := db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": id}).Decode(&p); err != nil { c.JSON(404, gin.H{"error": "Property not found"}); return }
	if decision == "VERIFIED" {
		reportHash, hashOK := p["latest_verification_report_hash"].(string)
		reportStatus, statusOK := p["latest_verification_status"].(string)
		if !hashOK || strings.TrimSpace(reportHash) == "" || !statusOK || reportStatus != "CLEAR" { c.JSON(409, gin.H{"error": "VERIFIED requires the latest reconciliation report to be CLEAR"}); return }
		var report VerificationReport
		if err := db.Collection("verification_reports").FindOne(context.Background(), bson.M{"property_id": id, "sha256": reportHash}).Decode(&report); err != nil || report.Status != "CLEAR" { c.JSON(409, gin.H{"error": "Latest verification artifact is not CLEAR"}); return }
	}
	set := bson.M{"verification_status": decision, "verification_notes": strings.TrimSpace(req.Notes), "verified_by": currentPhone(c), "verified_at": time.Now(), "updated_at": time.Now()}
	if decision != "VERIFIED" { set["verification_score"] = 0 }
	if decision == "VERIFIED" { if score, ok := p["verification_score"]; ok { set["verification_score"] = score } }
	res, err := db.Collection("properties").UpdateOne(context.Background(), bson.M{"property_id": id}, bson.M{"$set": set})
	if err != nil || res.MatchedCount == 0 { c.JSON(404, gin.H{"error": "Property not found"}); return }
	_, _ = db.Collection("audit_events").InsertOne(context.Background(), bson.M{"property_id": id, "event_type": "AUTHORITY_REVIEW", "actor": currentPhone(c), "decision": decision, "notes": strings.TrimSpace(req.Notes), "created_at": time.Now().UTC()})
	c.JSON(200, gin.H{"status": decision, "property_id": id})
}

func getRole(phone string) string { var u bson.M; if db.Collection("users").FindOne(context.Background(), bson.M{"phone_number": phone}).Decode(&u) != nil { return "CITIZEN" }; if v, ok := u["role"].(string); ok { return v }; return "CITIZEN" }

func initiateTransfer(c *gin.Context) {
	var req struct { PropertyID string `json:"property_id"`; BuyerPhone string `json:"buyer_phone"`; Amount float64 `json:"amount"`; Notes string `json:"notes"` }
	if c.BindJSON(&req) != nil || strings.TrimSpace(req.PropertyID) == "" || strings.TrimSpace(req.BuyerPhone) == "" { c.JSON(400, gin.H{"error": "property_id and buyer_phone required"}); return }
	var prop bson.M
	if db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": req.PropertyID}).Decode(&prop) != nil { c.JSON(404, gin.H{"error": "Property not found"}); return }
	if prop["owner_phone"] != currentPhone(c) { c.JSON(403, gin.H{"error": "Only current owner can initiate transfer"}); return }
	status, _ := prop["verification_status"].(string)
	if status != "VERIFIED" { c.JSON(409, gin.H{"error": "Transfer blocked: property is not VERIFIED", "verification_status": status}); return }
	transferID := "TRF-" + strings.ToUpper(randomToken()[:10])
	now := time.Now().UTC()
	t := bson.M{"transfer_id": transferID, "property_id": req.PropertyID, "seller_phone": currentPhone(c), "buyer_phone": req.BuyerPhone, "amount": req.Amount, "notes": req.Notes, "status": "PENDING", "created_at": now}
	if _, err := db.Collection("transfers").InsertOne(context.Background(), t); err != nil { c.JSON(500, gin.H{"error": "Could not create transfer"}); return }
	_, _ = db.Collection("audit_events").InsertOne(context.Background(), bson.M{"property_id": req.PropertyID, "event_type": "TRANSFER_INITIATED", "actor": currentPhone(c), "transfer_id": transferID, "buyer_phone": req.BuyerPhone, "created_at": now})
	c.JSON(201, t)
}

func listTransfers(c *gin.Context, field string) { cur, err := db.Collection("transfers").Find(context.Background(), bson.M{field: currentPhone(c)}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})); if err != nil { c.JSON(500, gin.H{"error": "Database error"}); return }; defer cur.Close(context.Background()); var docs []bson.M; _ = cur.All(context.Background(), &docs); c.JSON(200, gin.H{"transfers": docs, "count": len(docs)}) }
func getIncomingTransfers(c *gin.Context) { listTransfers(c, "buyer_phone") }
func getOutgoingTransfers(c *gin.Context) { listTransfers(c, "seller_phone") }

func acceptTransfer(c *gin.Context) {
	var t bson.M
	if db.Collection("transfers").FindOne(context.Background(), bson.M{"transfer_id": c.Param("id"), "buyer_phone": currentPhone(c), "status": "PENDING"}).Decode(&t) != nil { c.JSON(404, gin.H{"error": "Pending transfer not found"}); return }
	pid, ok := t["property_id"].(string); if !ok { c.JSON(500, gin.H{"error": "Invalid transfer record"}); return }
	var prop bson.M
	if db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": pid, "verification_status": "VERIFIED"}).Decode(&prop) != nil { c.JSON(409, gin.H{"error": "Property is no longer verified or could not be transferred"}); return }
	seller, _ := t["seller_phone"].(string)
	now := time.Now().UTC()
	res, err := db.Collection("transfers").UpdateOne(context.Background(), bson.M{"transfer_id": c.Param("id"), "buyer_phone": currentPhone(c), "seller_phone": seller, "status": "PENDING"}, bson.M{"$set": bson.M{"status": "COMPLETED", "completed_at": now}})
	if err != nil || res.MatchedCount == 0 { c.JSON(409, gin.H{"error": "Transfer is no longer pending"}); return }
	res, err = db.Collection("properties").UpdateOne(context.Background(), bson.M{"property_id": pid, "owner_phone": seller, "verification_status": "VERIFIED"}, bson.M{"$set": bson.M{"owner_phone": currentPhone(c), "updated_at": now}})
	if err != nil || res.MatchedCount == 0 {
		_, _ = db.Collection("transfers").UpdateOne(context.Background(), bson.M{"transfer_id": c.Param("id"), "status": "COMPLETED"}, bson.M{"$set": bson.M{"status": "PENDING"}, "$unset": bson.M{"completed_at": ""}})
		c.JSON(409, gin.H{"error": "Property ownership could not be updated; transfer remains pending"})
		return
	}
	_, _ = db.Collection("audit_events").InsertOne(context.Background(), bson.M{"property_id": pid, "event_type": "TRANSFER_ACCEPTED", "actor": currentPhone(c), "transfer_id": c.Param("id"), "previous_owner": seller, "new_owner": currentPhone(c), "created_at": now})
	c.JSON(200, gin.H{"status": "COMPLETED", "transfer_id": c.Param("id")})
}

func rejectTransfer(c *gin.Context) {
	var req struct { Reason string `json:"reason"` }
	_ = c.BindJSON(&req)
	now := time.Now().UTC()
	res, err := db.Collection("transfers").UpdateOne(context.Background(), bson.M{"transfer_id": c.Param("id"), "buyer_phone": currentPhone(c), "status": "PENDING"}, bson.M{"$set": bson.M{"status": "REJECTED", "rejection_reason": strings.TrimSpace(req.Reason), "rejected_at": now}})
	if err != nil || res.MatchedCount == 0 { c.JSON(404, gin.H{"error": "Pending transfer not found"}); return }
	var t bson.M
	_ = db.Collection("transfers").FindOne(context.Background(), bson.M{"transfer_id": c.Param("id")}).Decode(&t)
	pid, _ := t["property_id"].(string)
	_, _ = db.Collection("audit_events").InsertOne(context.Background(), bson.M{"property_id": pid, "event_type": "TRANSFER_REJECTED", "actor": currentPhone(c), "transfer_id": c.Param("id"), "reason": strings.TrimSpace(req.Reason), "created_at": now})
	c.JSON(200, gin.H{"status": "REJECTED", "transfer_id": c.Param("id")})
}
