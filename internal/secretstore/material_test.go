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
	"software.sslmate.com/src/go-pkcs12"
)

func TestPreparePEMAndCredential(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	prep, err := secretstore.Prepare(secretstore.MaterialInput{
		Kind: secretstore.KindProducerKey, Encoding: secretstore.EncodingPEM, Bytes: pemKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	if prep.FormatNote != "pem_private_key" || len(prep.StorageBytes) == 0 {
		t.Fatalf("%+v", prep)
	}
	secretstore.ZeroBytes(prep.StorageBytes)

	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test-prep"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	prep, err = secretstore.Prepare(secretstore.MaterialInput{
		Kind: secretstore.KindCertificate, Encoding: secretstore.EncodingPEM, Bytes: pemCert,
	})
	if err != nil || prep.FormatNote != "pem_certificate" {
		t.Fatalf("cert: %v %+v", err, prep)
	}

	cred := []byte("synthetic-credential-not-real")
	prep, err = secretstore.Prepare(secretstore.MaterialInput{
		Kind: secretstore.KindProducerCredential, Encoding: secretstore.EncodingCredential, Bytes: cred,
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(prep.StorageBytes) != string(cred) {
		t.Fatal("credential mismatch")
	}

	_, err = secretstore.Prepare(secretstore.MaterialInput{
		Kind: secretstore.KindProducerKey, Encoding: secretstore.EncodingPEM,
		Bytes: []byte("-----BEGIN PRIVATE KEY-----\nbad\n-----END PRIVATE KEY-----"),
	})
	if err == nil {
		t.Fatal("want invalid PEM error")
	}
}

func TestPreparePKCS12PasswordEphemeral(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "pkcs12-test"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatal(err)
	}
	pass := "ephemeral-test-pass"
	pfx, err := pkcs12.Modern.Encode(key, cert, nil, pass)
	if err != nil {
		t.Fatal(err)
	}
	pw := []byte(pass)
	prep, err := secretstore.Prepare(secretstore.MaterialInput{
		Kind: secretstore.KindCertificate, Encoding: secretstore.EncodingPKCS12,
		Bytes: pfx, Password: pw,
	})
	if err != nil {
		t.Fatal(err)
	}
	secretstore.ZeroBytes(pw)
	if string(pw) == pass {
		t.Fatal("password not zeroed")
	}
	if prep.FormatNote != "pkcs12" {
		t.Fatalf("%+v", prep)
	}
	// Prepared must not contain password string
	if string(prep.StorageBytes) == pass || containsBytes(prep.StorageBytes, []byte(pass)) {
		t.Fatal("password leaked into storage bytes")
	}

	_, err = secretstore.Prepare(secretstore.MaterialInput{
		Kind: secretstore.KindCertificate, Encoding: secretstore.EncodingPKCS12,
		Bytes: pfx, Password: []byte("wrong"),
	})
	if err == nil {
		t.Fatal("want wrong password error")
	}
}

func containsBytes(hay, needle []byte) bool {
	return len(needle) > 0 && string(hay) != "" && (len(hay) >= len(needle)) &&
		(string(hay) == string(needle) || len(needle) < len(hay) && containsStr(string(hay), string(needle)))
}

func containsStr(hay, needle string) bool {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
