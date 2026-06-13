package handlers

import (
	"net/http"
	"provision-server/internal/services"

	"github.com/gin-gonic/gin"
)

func ScanProvision(c *gin.Context) {
	var req services.ScanProvisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	svc := services.NewProvisionService()
	zipData, err := svc.ScanProvision(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\"provision-"+req.DeviceSN+".zip\"")
	c.Data(http.StatusOK, "application/zip", zipData)
}

func DownloadProvision(c *gin.Context) {
	sn := c.Param("sn")
	svc := services.NewProvisionService()
	zipData, err := svc.GetProvisionBundle(sn)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\"provision-"+sn+".zip\"")
	c.Data(http.StatusOK, "application/zip", zipData)
}
