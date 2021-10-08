/*
Elliptic curve cryptography is used to generate public keys and Bitcoin addresses.

The public key can be generated by multiplying the private key and the generator point on the Elliptic curve.

K = k * G

G is the generator point on the Elliptic curve.
k is a private key in the form of a randomly generated number.
K is a public key.

How to generate Bitcoin address

Pulick Key --> SHA256 --> RIPEMD160 --> Base58Check Encoding --> Bitcoin Address

Base58Check Encoding
1.            Payload
2.  Version + Payload
3. (Version + Payload) --> SHA256 --> SHA256 --> First 4bytes --> Checksum
3.  Version + Payload + Checksum
4. (Version + Payload + Checksum) --> Base58 encode

*/
package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"

	log "github.com/sirupsen/logrus"

	"github.com/mr-tron/base58"
	"golang.org/x/crypto/ripemd160"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
}

func NewWallet() *Wallet {
	curve := elliptic.P256()

	priKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pubK := append(priKey.PublicKey.X.Bytes(), priKey.PublicKey.Y.Bytes()...)

	return &Wallet{priKey, pubK}
}

func getChecksum(payload []byte) []byte {
	h1 := sha256.Sum256(payload)
	h2 := sha256.Sum256(h1[:])

	return h2[:checksumLength]
}

func HashPubKey(pubKey []byte) []byte {
	publicSha256 := sha256.Sum256(pubKey)
	ripemd160Hasher := ripemd160.New()
	_, err := ripemd160Hasher.Write(publicSha256[:])
	if err != nil {
		log.Panic(err)
	}

	return ripemd160Hasher.Sum(nil)
}

func (w *Wallet) getAddress() []byte {
	payload := HashPubKey(w.PublicKey)

	verPayload := append([]byte{version}, payload...)
	checksum := getChecksum(verPayload)

	fullPayload := append(verPayload, checksum...)
	address := Base58Encode(fullPayload)

	ich := binary.LittleEndian.Uint32(checksum)
	log.Infof("checksum : %v", ich)
	log.Infof("payload  : %x", payload)
	log.Infof("version  : %d", version)

	return address
}

func getPayload(addr []byte) (byte, []byte, []byte) {
	fullPayload := Base58Decode(addr)
	checksum := fullPayload[len(fullPayload)-checksumLength:]
	version := fullPayload[0]
	payload := fullPayload[1 : len(fullPayload)-checksumLength]

	return version, payload, checksum
}

func ValidateAddress(addr []byte) bool {
	version, payload, checksum := getPayload(addr)
	targetChecksum := getChecksum(append([]byte{version}, payload...))

	ich := binary.LittleEndian.Uint32(checksum)
	log.Infof("checksum : %d", ich)
	log.Infof("payload  : %x", payload)
	log.Infof("version  : %d", version)

	return bytes.Equal(checksum, targetChecksum)
}

func Base58Encode(input []byte) []byte {
	encode := base58.Encode(input)

	return []byte(encode)
}

func Base58Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input[:]))
	if err != nil {
		log.Panic(err)
	}

	return decode
}