package main

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func registerTransferAuthorityRoutes(protected *gin.RouterGroup) {
	protected.GET("/transfer/authority-queue", getTransferAuthorityQueue)
	protected.POST("/transfer/:id/authority-approve", approveTransferByAuthority)
	protected.POST("/transfer/:id/authority-reject", rejectTransferByAuthority)
}

func getTransferAuthorityQueue(c *gin.Context) {
	if !sourceAuthorityRequired(c) { return }
	cur, err := db.Collection("transfers").Find(context.Background(), bson.M{"status": "PENDING"}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil { c.JSON(500, gin.H{"error": "Database error"}); return }
	defer cur.Close(context.Background())
	var transfers []bson.M
	if err := cur.All(context.Background(), &transfers); err != nil { c.JSON(500, gin.H{"error": "Could not read transfer queue"}); return }
	c.JSON(200, gin.H{"transfers": transfers, "count": len(transfers)})
}

func approveTransferByAuthority(c *gin.Context) {
	if !sourceAuthorityRequired(c) { return }
	id := strings.TrimSpace(c.Param("id"))
	var t bson.M
	if err := db.Collection("transfers").FindOne(context.Background(), bson.M{"transfer_id": id, "status": "PENDING"}).Decode(&t); err != nil { c.JSON(404, gin.H{"error": "Pending transfer not found"}); return }
	pid, _ := t["property_id"].(string)
	var p bson.M
	if err := db.Collection("properties").FindOne(context.Background(), bson.M{"property_id": pid, "verification_status": "VERIFIED"}).Decode(&p); err != nil { c.JSON(409, gin.H{"error": "Transfer requires a VERIFIED property"}); return }
	now := time.Now().UTC()
	res, err := db.Collection("transfers").UpdateOne(context.Background(), bson.M{"transfer_id": id, "status": "PENDING"}, bson.M{"$set": bson.M{"authority_approved": true, "authority_verified_by": currentPhone(c), "authority_verified_at": now}})
	if err != nil || res.MatchedCount == 0 { c.JSON(409, gin.H{"error": "Transfer changed before authority approval"}); return }
	_, _ = db.Collection("audit_events").InsertOne(context.Background(), bson.M{"property_id": pid, "event_type": "TRANSFER_AUTHORITY_APPROVED", "actor": currentPhone(c), "transfer_id": id, "created_at": now})
	c.JSON(200, gin.H{"status": "AUTHORITY_APPROVED", "transfer_id": id})
}

func rejectTransferByAuthority(c *gin.Context) {
	if !sourceAuthorityRequired(c) { return }
	id := strings.TrimSpace(c.Param("id"))
	now := time.Now().UTC()
	res, err := db.Collection("transfers").UpdateOne(context.Background(), bson.M{"transfer_id": id, "status": "PENDING"}, bson.M{"$set": bson.M{"status": "AUTHORITY_REJECTED", "authority_verified_by": currentPhone(c), "authority_verified_at": now}})
	if err != nil || res.MatchedCount == 0 { c.JSON(404, gin.H{"error": "Pending transfer not found"}); return }
	var t bson.M
	_ = db.Collection("transfers").FindOne(context.Background(), bson.M{"transfer_id": id}).Decode(&t)
	pid, _ := t["property_id"].(string)
	_, _ = db.Collection("audit_events").InsertOne(context.Background(), bson.M{"property_id": pid, "event_type": "TRANSFER_AUTHORITY_REJECTED", "actor": currentPhone(c), "transfer_id": id, "created_at": now})
	c.JSON(200, gin.H{"status": "AUTHORITY_REJECTED", "transfer_id": id})
}
