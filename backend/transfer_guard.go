package main

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

// transferApprovalGuard closes the server-side gap between the presentation
// workflow and ownership mutation: a buyer cannot accept a pending transfer
// until an authorized authority has explicitly approved it.
func transferApprovalGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" || !strings.HasSuffix(c.Request.URL.Path, "/accept") {
			c.Next()
			return
		}

		id := strings.TrimSpace(c.Param("id"))
		if id == "" {
			c.JSON(400, gin.H{"error": "Transfer ID required"})
			c.Abort()
			return
		}

		var transfer bson.M
		err := db.Collection("transfers").FindOne(context.Background(), bson.M{
			"transfer_id": id,
			"status":      "PENDING",
		}).Decode(&transfer)
		if err != nil {
			c.JSON(404, gin.H{"error": "Pending transfer not found"})
			c.Abort()
			return
		}

		approved, _ := transfer["authority_approved"].(bool)
		if !approved {
			c.JSON(409, gin.H{"error": "Transfer requires authority approval before buyer acceptance"})
			c.Abort()
			return
		}

		c.Next()
	}
}
