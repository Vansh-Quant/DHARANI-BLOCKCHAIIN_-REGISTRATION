package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

type BlockchainAnchorResult struct {
	PropertyID           string    `json:"property_id" bson:"property_id"`
	BlockchainPropertyID uint64    `json:"blockchain_property_id" bson:"blockchain_property_id"`
	ReportHash           string    `json:"report_hash" bson:"report_hash"`
	TransactionHash      string    `json:"transaction_hash" bson:"transaction_hash"`
	ContractAddress      string    `json:"contract_address" bson:"contract_address"`
	Network              string    `json:"network" bson:"network"`
	AnchoredAt           time.Time `json:"anchored_at" bson:"anchored_at"`
}

func registerBlockchainAnchorRoutes(protected *gin.RouterGroup) {
	protected.POST("/blockchain/anchor-result", recordBlockchainAnchor)
	protected.GET("/blockchain/:propertyId/proof", getBlockchainProof)
}

func recordBlockchainAnchor(c *gin.Context) {
	if !sourceAuthorityRequired(c) { return }
	var req BlockchainAnchorResult
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error":"Invalid blockchain anchor result","details":err.Error()}); return }
	req.PropertyID=strings.TrimSpace(req.PropertyID); req.ReportHash=strings.ToLower(strings.TrimSpace(req.ReportHash)); req.TransactionHash=strings.TrimSpace(req.TransactionHash); req.ContractAddress=strings.TrimSpace(req.ContractAddress); req.Network=strings.TrimSpace(req.Network)
	if req.PropertyID==""||req.ReportHash==""||req.TransactionHash==""||req.ContractAddress==""||req.Network=="" { c.JSON(http.StatusBadRequest,gin.H{"error":"property_id, report_hash, transaction_hash, contract_address and network are required"}); return }
	if len(req.ReportHash)!=64||!isHex(req.ReportHash){c.JSON(http.StatusBadRequest,gin.H{"error":"report_hash must be a 64-character SHA-256 hex string"});return}
	if len(req.TransactionHash)<8{c.JSON(http.StatusBadRequest,gin.H{"error":"transaction_hash is invalid"});return}
	var report VerificationReport
	if err:=db.Collection("verification_reports").FindOne(context.Background(),bson.M{"property_id":req.PropertyID,"sha256":req.ReportHash}).Decode(&report);err!=nil{c.JSON(http.StatusConflict,gin.H{"error":"Report hash does not match a stored verification report"});return}
	var property bson.M
	if err:=db.Collection("properties").FindOne(context.Background(),bson.M{"property_id":req.PropertyID}).Decode(&property);err!=nil{c.JSON(http.StatusNotFound,gin.H{"error":"Property not found"});return}
	if status,_:=property["verification_status"].(string);status!="VERIFIED"{c.JSON(http.StatusConflict,gin.H{"error":"Blockchain anchoring requires authority verification first","verification_status":status});return}
	req.AnchoredAt=time.Now().UTC()	res,err:=db.Collection("properties").UpdateOne(context.Background(),bson.M{"property_id":req.PropertyID,"latest_verification_report_hash":req.ReportHash,"verification_status":"VERIFIED"},bson.M{"$set":bson.M{"blockchain_proof":req,"blockchain_status":"ANCHORED","updated_at":req.AnchoredAt}})
	if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"Could not store blockchain proof"});return};if res.MatchedCount==0{c.JSON(http.StatusConflict,gin.H{"error":"Property report hash does not match its latest verified artifact"});return}
	_,_=db.Collection("audit_events").InsertOne(context.Background(),bson.M{"property_id":req.PropertyID,"event_type":"BLOCKCHAIN_ANCHORED","actor":currentPhone(c),"report_hash":req.ReportHash,"transaction_hash":req.TransactionHash,"contract_address":req.ContractAddress,"network":req.Network,"created_at":req.AnchoredAt})
	c.JSON(http.StatusOK,gin.H{"message":"Blockchain proof recorded","proof":req})
}

func getBlockchainProof(c *gin.Context){propertyID:=strings.TrimSpace(c.Param("propertyId"));var property bson.M;if err:=db.Collection("properties").FindOne(context.Background(),bson.M{"property_id":propertyID}).Decode(&property);err!=nil{c.JSON(http.StatusNotFound,gin.H{"error":"Property not found"});return};proof,ok:=property["blockchain_proof"];if !ok{c.JSON(http.StatusNotFound,gin.H{"error":"Blockchain proof not recorded"});return};c.JSON(http.StatusOK,gin.H{"property_id":propertyID,"blockchain_status":property["blockchain_status"],"proof":proof})}
func isHex(value string)bool{for _,r:=range value{if !(r>='0'&&r<='9')&&!(r>='a'&&r<='f')&&!(r>='A'&&r<='F'){return false}};return true}
