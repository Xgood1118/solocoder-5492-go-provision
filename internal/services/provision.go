package services

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"provision-server/internal/certgen"
	"provision-server/internal/config"
	"provision-server/internal/db"
	"provision-server/internal/models"
	"sync"
)

var (
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
	caOnce sync.Once
	caErr  error
)

func InitCA() error {
	caOnce.Do(func() {
		caCert, caKey, caErr = loadOrCreateCA()
	})
	return caErr
}

func loadOrCreateCA() (*x509.Certificate, *rsa.PrivateKey, error) {
	caCertPath := config.App.FirmwareDir + "/../ca.crt"
	caKeyPath := config.App.FirmwareDir + "/../ca.key"

	if _, err := os.Stat(caCertPath); err == nil {
		certPEM, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, nil, err
		}
		keyPEM, err := os.ReadFile(caKeyPath)
		if err != nil {
			return nil, nil, err
		}
		certBlock, _ := pem.Decode(certPEM)
		if certBlock == nil {
			return nil, nil, errors.New("invalid CA cert")
		}
		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, nil, err
		}
		keyBlock, _ := pem.Decode(keyPEM)
		if keyBlock == nil {
			return nil, nil, errors.New("invalid CA key")
		}
		key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, nil, err
		}
		return cert, key, nil
	}

	_ = os.MkdirAll(config.App.FirmwareDir+"/..", 0755)
	cert, key, certPEM, keyPEM, err := certgen.GenerateCA()
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(caCertPath, certPEM, 0644); err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(caKeyPath, keyPEM, 0600); err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

type ProvisionService struct {
	devSvc *DeviceService
}

func NewProvisionService() *ProvisionService {
	return &ProvisionService{
		devSvc: NewDeviceService(),
	}
}

type ScanProvisionRequest struct {
	DeviceSN   string `json:"device_sn" binding:"required"`
	DeviceMAC  string `json:"device_mac"`
	ModelID    uint   `json:"model_id"`
	BatchID    uint   `json:"batch_id"`
	TemplateID uint   `json:"template_id"`
	Customer   string `json:"customer"`
	Region     string `json:"region"`
	Remark     string `json:"remark"`
}

func (s *ProvisionService) ScanProvision(req *ScanProvisionRequest) ([]byte, error) {
	if caCert == nil || caKey == nil {
		return nil, errors.New("CA not initialized")
	}

	d, err := s.devSvc.GetBySN(req.DeviceSN)
	if err != nil {
		d = &models.Device{
			DeviceSN:   req.DeviceSN,
			DeviceMAC:  req.DeviceMAC,
			ModelID:    req.ModelID,
			BatchID:    req.BatchID,
			TemplateID: req.TemplateID,
			Customer:   req.Customer,
			Region:     req.Region,
			Remark:     req.Remark,
		}
		if err := s.devSvc.Create(d); err != nil {
			return nil, err
		}
	}

	if d.Status != models.StatusPending && d.Status != models.StatusProvisioned {
		return nil, fmt.Errorf("device status %s cannot be provisioned", d.Status)
	}

	bundle, err := certgen.GenerateDeviceCert(caCert, caKey, req.DeviceSN)
	if err != nil {
		return nil, err
	}

	var cert models.DeviceCert
	db.DB.Where("device_id = ?", d.ID).First(&cert)
	cert.DeviceID = d.ID
	cert.CertSN = bundle.CertSN
	cert.CertPEM = string(bundle.CertPEM)
	cert.KeyPEM = string(bundle.KeyPEM)
	cert.ExpiresAt = caCert.NotAfter
	if err := db.DB.Save(&cert).Error; err != nil {
		return nil, err
	}

	tpl, err := s.getTemplate(d.TemplateID, d.BatchID, d.ModelID)
	if err != nil {
		return nil, err
	}

	zipData, err := certgen.BundleZIP(bundle, tpl.WifiSSID, tpl.WifiPSK, tpl.MqttHost, tpl.MqttPort, d.DeviceSN)
	if err != nil {
		return nil, err
	}

	if _, err := s.devSvc.Transition(d.DeviceSN, models.StatusProvisioned, "scan provision"); err != nil {
		return nil, err
	}

	return zipData, nil
}

func (s *ProvisionService) getTemplate(tplID, batchID, modelID uint) (*models.Template, error) {
	if tplID > 0 {
		var tpl models.Template
		if err := db.DB.First(&tpl, tplID).Error; err == nil {
			return &tpl, nil
		}
	}
	if batchID > 0 {
		var batch models.Batch
		if err := db.DB.First(&batch, batchID).Error; err == nil && batch.TemplateID > 0 {
			var tpl models.Template
			if err := db.DB.First(&tpl, batch.TemplateID).Error; err == nil {
				return &tpl, nil
			}
		}
	}
	var tpl models.Template
	if err := db.DB.Where("type = ?", "default").First(&tpl).Error; err != nil {
		return &models.Template{
			WifiSSID: "",
			WifiPSK:  "",
			MqttHost: "127.0.0.1",
			MqttPort: 1883,
		}, nil
	}
	return &tpl, nil
}

func (s *ProvisionService) GetProvisionBundle(deviceSN string) ([]byte, error) {
	d, err := s.devSvc.GetBySN(deviceSN)
	if err != nil {
		return nil, err
	}

	if d.Status != models.StatusProvisioned && d.Status != models.StatusActivated {
		return nil, errors.New("device not provisioned")
	}

	var cert models.DeviceCert
	if err := db.DB.Where("device_id = ?", d.ID).First(&cert).Error; err != nil {
		return nil, errors.New("cert not found")
	}

	tpl, err := s.getTemplate(d.TemplateID, d.BatchID, d.ModelID)
	if err != nil {
		return nil, err
	}

	bundle := &certgen.CertBundle{
		CertPEM: []byte(cert.CertPEM),
		KeyPEM:  []byte(cert.KeyPEM),
		CertSN:  cert.CertSN,
	}

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
	bundle.CAPEM = caPEM

	return certgen.BundleZIP(bundle, tpl.WifiSSID, tpl.WifiPSK, tpl.MqttHost, tpl.MqttPort, d.DeviceSN)
}
