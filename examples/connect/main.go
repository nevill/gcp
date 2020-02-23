package main

// This program will create a random SSH key pair, then do the same as:
// gcloud compute instances add-metadata instances_name \
// --metadata "ssh-keys=user_name:ssh-rsa public_key user@host.com"

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"strings"

	"github.com/nevill/gcp/compute"
	"golang.org/x/crypto/ssh"
)

// User is a default name for ssh connection.
const User = "bot"

// generate private / pub key to instance via setting metadata

func generateKeyPair() (*string, *string, error) {
	rsaBits := 2048
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, nil, err
	}
	privateKeyPemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyPem := string(pem.EncodeToMemory(privateKeyPemBlock))

	sshPubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	sshPubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
	openSSHPubKey := strings.TrimSpace(string(sshPubKeyBytes))

	return &privateKeyPem, &openSSHPubKey, nil

}

func connect(address, key string) (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:22", address), config)
}

func main() {
	privateKey, publicKey, theError := generateKeyPair()
	if theError != nil {
		panic(theError)
	}
	log.Println("public:", *publicKey, "private:", *privateKey)

	computeManager, err := compute.New()

	if err != nil {
		log.Fatal("Failed to connect: ", err)
	}

	instance, err := computeManager.GetInstance()
	if err != nil {
		log.Fatal("Error get instance info:", err)
	}

	metadata := make(map[string]string, 1)

	metadata["ssh-keys"] = fmt.Sprintf("%s:%s %s", User, *publicKey, User)
	if err := computeManager.SetMetadata(metadata); err != nil {
		log.Fatal(err)
	}

	externalIP := ""
	for _, networkInterface := range instance.NetworkInterfaces {
		if accessConfig := networkInterface.AccessConfigs[0]; len(accessConfig.NatIP) > 0 {
			externalIP = accessConfig.NatIP
			break
		}
	}

	log.Println("External IP:", externalIP)

	client, err := connect(externalIP, *privateKey)
	if err != nil {
		log.Fatal("Failed to connect: ", err)
	}

	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	if err := session.Run("/usr/bin/whoami && id"); err != nil {
		log.Fatal("Failed to run: " + err.Error())
	}
	fmt.Println(buf.String())
}
