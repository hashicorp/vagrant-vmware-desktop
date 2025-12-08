// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package utility

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path"
	"time"
)

// number of years until certificate expiry
const CERTIFICATE_EXPIRES_IN = 10

type CertificatePaths struct {
	Certificate       string
	PrivateKey        string
	ClientCertificate string
	ClientKey         string
}

func GenerateCertificate() error {
	paths, err := GetCertificatePaths()
	if err != nil {
		return errors.New(fmt.Sprintf(
			"path generation failed: %s", err))
	}
	serialMax := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"setup failure encountered: %s", err))
	}
	clientSerial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"setup failure encountered: %s", err))
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"private key generation failed: %s", err))
	}
	expires := time.Now().Add(((time.Hour * 24) * 365) * CERTIFICATE_EXPIRES_IN)
	cert := x509.Certificate{
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:                  true,
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign | x509.KeyUsageDataEncipherment,
		NotBefore:    time.Now(),
		NotAfter:     expires,
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"HashiCorp"},
		},
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert,
		&privateKey.PublicKey, privateKey)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"certificate generation failed: %s", err))
	}

	if err := os.RemoveAll(paths.Certificate); err != nil {
		return errors.New(fmt.Sprintf(
			"certificate path cleanup error: %s", err))
	}
	if err := os.RemoveAll(paths.PrivateKey); err != nil {
		return errors.New(fmt.Sprintf(
			"private key path cleanup error: %s", err))
	}
	if err := os.RemoveAll(paths.ClientCertificate); err != nil {
		return errors.New(fmt.Sprintf(
			"client certificate path cleanup error: %s", err))
	}
	if err := os.RemoveAll(paths.ClientKey); err != nil {
		return errors.New(fmt.Sprintf(
			"client key path cleanup error: %s", err))
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"client key generation failed: %s", err))
	}
	clientCert := x509.Certificate{
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		NotBefore:             time.Now(),
		NotAfter:              expires,
		SerialNumber:          clientSerial,
		Subject: pkix.Name{
			Organization: []string{"Vagrant"},
		},
	}
	parentCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"cert parse failure: %s", err))
	}
	clientCertBytes, err := x509.CreateCertificate(rand.Reader, &clientCert, parentCert,
		&clientKey.PublicKey, privateKey)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"client certificate generation failed: %s", err))
	}

	certFile, err := os.Create(paths.Certificate)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"certificate write failure: %s", err))
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return errors.New(fmt.Sprintf(
			"certificate encoding failure: %s", err))
	}
	keyFile, err := os.OpenFile(paths.PrivateKey, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"private key write failure: %s", err))
	}
	defer keyFile.Close()
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		return errors.New(fmt.Sprintf(
			"private key write failure: %s", err))
	}

	clientCertFile, err := os.Create(paths.ClientCertificate)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"client certificate write failure: %s", err))
	}
	defer clientCertFile.Close()
	if err := pem.Encode(clientCertFile, &pem.Block{Type: "CERTIFICATE", Bytes: clientCertBytes}); err != nil {
		return errors.New(fmt.Sprintf(
			"client certificate encoding failure: %s", err))
	}
	clientKeyFile, err := os.Create(paths.ClientKey)
	if err != nil {
		return errors.New(fmt.Sprintf(
			"client key write failure: %s", err))
	}
	defer clientKeyFile.Close()
	if err := pem.Encode(clientKeyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(clientKey)}); err != nil {
		return errors.New(fmt.Sprintf(
			"client key write failure: %s", err))
	}

	return nil
}

// Paths are based on platform. If the platform can't be detected
// then we just use the executable's directory as the base and create
// a certificate directory within.
func GetCertificatePaths() (*CertificatePaths, error) {
	basePath := DirectoryFor("certificates")
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}
	return &CertificatePaths{
		Certificate:       path.Join(basePath, "vagrant-utility.crt"),
		PrivateKey:        path.Join(basePath, "vagrant-utility.key"),
		ClientCertificate: path.Join(basePath, "vagrant-utility.client.crt"),
		ClientKey:         path.Join(basePath, "vagrant-utility.client.key"),
	}, nil
}
