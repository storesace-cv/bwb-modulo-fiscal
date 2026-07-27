package secretstore_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestValidateOfflineKeyCertPair(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(7),
		Subject:               pkix.Name{CommonName: "offline-test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	rep, err := secretstore.ValidateOfflineKeyCert(keyPEM, certPEM, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.OK || !rep.PairMatch || !rep.WithinValidity || !rep.PurposeOK || rep.FingerprintSHA256 == "" {
		t.Fatalf("%+v", rep)
	}
	if rep.KeyBits != 2048 || rep.AlgorithmNote != "RSA" {
		t.Fatalf("alg %+v", rep)
	}

	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	otherPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(other)})
	rep, err = secretstore.ValidateOfflineKeyCert(otherPEM, certPEM, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK || rep.PairMatch {
		t.Fatalf("mismatched pair should fail: %+v", rep)
	}
}
