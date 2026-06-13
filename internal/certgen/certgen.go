package certgen

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"provision-server/internal/config"
	"strconv"
	"time"
)

type CertBundle struct {
	CertPEM []byte
	KeyPEM  []byte
	CAPEM   []byte
	CertSN  string
}

func GenerateCA() (*x509.Certificate, *rsa.PrivateKey, []byte, []byte, error) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	caTpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Provision CA"},
			CommonName:   "Factory Root CA",
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})
	return caTpl, caKey, caPEM, caKeyPEM, nil
}

func GenerateDeviceCert(caCert *x509.Certificate, caKey *rsa.PrivateKey, deviceSN string) (*CertBundle, error) {
	devKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	sn, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tpl := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			Organization: []string{"Provision Device"},
			CommonName:   deviceSN,
		},
		NotBefore:   time.Now().Add(-24 * time.Hour),
		NotAfter:    time.Now().AddDate(0, 0, config.App.CertValidDays),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, &devKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(devKey)})
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
	return &CertBundle{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
		CAPEM:   caPEM,
		CertSN:  sn.String(),
	}, nil
}

func BundleZIP(cert *CertBundle, wifiSSID, wifiPSK, mqttHost string, mqttPort int, deviceSN string) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	addFile := func(name string, data []byte) error {
		f, err := w.Create(name)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	}

	if err := addFile("device.crt", cert.CertPEM); err != nil {
		return nil, err
	}
	if err := addFile("device.key", cert.KeyPEM); err != nil {
		return nil, err
	}
	if err := addFile("ca.crt", cert.CAPEM); err != nil {
		return nil, err
	}

	configText := "device_sn=" + deviceSN + "\n" +
		"wifi_ssid=" + wifiSSID + "\n" +
		"wifi_psk=" + wifiPSK + "\n" +
		"mqtt_host=" + mqttHost + "\n" +
		"mqtt_port=" + strconv.Itoa(mqttPort) + "\n" +
		"cert_sn=" + cert.CertSN + "\n"
	if err := addFile("device.cfg", []byte(configText)); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}


