package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	seedB64 := os.Getenv("ED25519_PRIVATE_SEED")
	if seedB64 == "" {
		log.Fatal("ED25519_PRIVATE_SEED missing")
	}
	seed, err := base64.StdEncoding.DecodeString(seedB64)
	if err != nil {
		log.Fatalf("invalid base64 seed: %v", err)
	}
	if len(seed) != 32 {
		log.Fatalf("seed must be 32 bytes (base64-encoded raw seed)")
	}
	priv := ed25519.NewKeyFromSeed(seed)
	if len(os.Args) < 2 {
		log.Fatal("usage: sign_checksums <checksums.txt>")
	}
	ckFile := os.Args[1]
	ck, err := ioutil.ReadFile(ckFile)
	if err != nil {
		log.Fatalf("failed reading checksums file: %v", err)
	}
	sig := ed25519.Sign(priv, ck)
	sigHex := hex.EncodeToString(sig)
	if err := ioutil.WriteFile(ckFile+".sig", []byte(sigHex), 0644); err != nil {
		log.Fatalf("failed writing signature file: %v", err)
	}
	log.Printf("signed %s -> %s.sig", ckFile, ckFile)
}
