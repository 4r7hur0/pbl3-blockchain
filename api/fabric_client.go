package main

import (
	"crypto/x509"
	"fmt"
	"os"
	"path"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	fabricChannelName   = "mychannel"
	fabricChaincodeName = "pbl3"
)

func newGateway() (*client.Gateway, error) {
	// Pega todas as configurações necessárias das variáveis de ambiente
	peerEndpoint := os.Getenv("FABRIC_PEER_ENDPOINT")
	peerHostname := os.Getenv("FABRIC_PEER_HOSTNAME")
	tlsCertPath := os.Getenv("FABRIC_TLS_CERT_PATH")
	mspID := os.Getenv("FABRIC_MSP_ID")
	certPath := os.Getenv("FABRIC_CERT_PATH")
	keyPath := os.Getenv("FABRIC_KEY_PATH")

	if peerEndpoint == "" || peerHostname == "" || tlsCertPath == "" || mspID == "" || certPath == "" || keyPath == "" {
		return nil, fmt.Errorf("uma ou mais variáveis de ambiente da Fabric não foram definidas")
	}

	transportCredentials, err := loadCertificate(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar certificado TLS: %w", err)
	}

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials), grpc.WithAuthority(peerHostname))
	if err != nil {
		return nil, fmt.Errorf("falha ao discar para o peer: %w", err)
	}

	id, err := newIdentity(certPath, mspID)
	if err != nil {
		return nil, err
	}

	sign, err := newSign(keyPath)
	if err != nil {
		return nil, err
	}

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(connection),
	)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao Gateway: %w", err)
	}

	return gw, nil
}

func loadCertificate(certPath string) (credentials.TransportCredentials, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo de certificado TLS (%s): %w", certPath, err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		return nil, fmt.Errorf("falha ao adicionar certificado ao pool")
	}
	return credentials.NewClientTLSFromCert(certPool, ""), nil
}

func newIdentity(certPath string, mspID string) (*identity.X509Identity, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo de certificado de identidade (%s): %w", certPath, err)
	}

	cert, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, err
	}

	return identity.NewX509Identity(mspID, cert)
}

func newSign(keyDir string) (identity.Sign, error) {
	files, err := os.ReadDir(keyDir)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler diretório da chave privada (%s): %w", keyDir, err)
	}
	keyPEM, err := os.ReadFile(path.Join(keyDir, files[0].Name()))
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo de chave privada: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, err
	}
	return identity.NewPrivateKeySign(privateKey)
}
