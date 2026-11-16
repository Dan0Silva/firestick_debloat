package adb

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

const (
	rsaKeyBits          = 2048
	androidPubKeyWords  = rsaKeyBits / 32 // 64 uint32 words
	androidPubKeyBytes  = rsaKeyBits / 8  // 256 bytes
	androidStructLength = 4 + 4 + androidPubKeyBytes + androidPubKeyBytes + 4
)

// DefaultKeyPath returns the path where the local ADB RSA private key is stored.
// E.g. ~/.config/firestick_debloat/adbkey
func DefaultKeyPath() (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("cannot determine user config or home directory: %w", err)
		}
		cfgDir = filepath.Join(home, ".config")
	}
	return filepath.Join(cfgDir, "firestick_debloat", "adbkey"), nil
}

// LoadOrGenerateKey loads an RSA private key from keyPath, or generates and saves
// a new 2048-bit RSA key if the file does not exist yet.
func LoadOrGenerateKey(keyPath string) (*rsa.PrivateKey, error) {
	if keyPath == "" {
		var err error
		keyPath, err = DefaultKeyPath()
		if err != nil {
			return nil, err
		}
	}

	if data, err := os.ReadFile(keyPath); err == nil {
		return parsePrivateKeyPEM(data)
	}

	privKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	if err := savePrivateKeyPEM(keyPath, privKey); err != nil {
		return nil, err
	}

	return privKey, nil
}

func parsePrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM block in ADB private key")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	pkcs8Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}
	rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an RSA key")
	}
	return rsaKey, nil
}

func savePrivateKeyPEM(keyPath string, key *rsa.PrivateKey) error {
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	if err := os.WriteFile(keyPath, pemBytes, 0o600); err != nil {
		return fmt.Errorf("failed to write ADB private key: %w", err)
	}
	return nil
}

// SignToken signs the 20-byte ADB authentication token using PKCS#1 v1.5 with SHA-1
// DigestInfo framing, matching AOSP's RSA_sign(NID_sha1, token, 20, ...) behavior.
func SignToken(priv *rsa.PrivateKey, token []byte) ([]byte, error) {
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA1, token)
}

// EncodeAndroidPublicKey encodes an *rsa.PublicKey into the 524-byte Android RSAPublicKey
// binary format (base64-encoded with a trailing user@host identifier and null byte),
// required by adbd during AuthRSAPublicKey (first-time TV authorization prompt).
func EncodeAndroidPublicKey(pub *rsa.PublicKey, comment string) ([]byte, error) {
	if pub.N.BitLen() != rsaKeyBits {
		return nil, fmt.Errorf("android adb requires %d-bit RSA key, got %d-bit", rsaKeyBits, pub.N.BitLen())
	}

	if comment == "" {
		comment = "firestick_debloat@local"
	}

	// Calculate n0inv = -1 / N[0] mod 2^32
	two32 := new(big.Int).Lsh(big.NewInt(1), 32)
	n0 := new(big.Int).Mod(pub.N, two32)
	n0Inv := new(big.Int).ModInverse(n0, two32)
	if n0Inv == nil {
		return nil, errors.New("failed to compute modular inverse for RSA modulus")
	}
	n0Inv.Sub(two32, n0Inv)

	// Calculate rr = (2^2048)^2 mod N = 2^4096 mod N
	r := new(big.Int).Lsh(big.NewInt(1), rsaKeyBits)
	rr := new(big.Int).Mul(r, r)
	rr.Mod(rr, pub.N)

	buf := make([]byte, androidStructLength)
	// 1. modulus_size_words (uint32 LE) = 64
	binary.LittleEndian.PutUint32(buf[0:4], uint32(androidPubKeyWords))
	// 2. n0inv (uint32 LE)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(n0Inv.Uint64()))

	// 3. modulus (256 bytes in Little-Endian order)
	modBytes := bigIntToLittleEndian(pub.N, androidPubKeyBytes)
	copy(buf[8:8+androidPubKeyBytes], modBytes)

	// 4. rr (256 bytes in Little-Endian order)
	rrBytes := bigIntToLittleEndian(rr, androidPubKeyBytes)
	copy(buf[8+androidPubKeyBytes:8+2*androidPubKeyBytes], rrBytes)

	// 5. exponent (uint32 LE)
	binary.LittleEndian.PutUint32(buf[8+2*androidPubKeyBytes:], uint32(pub.E))

	b64 := base64.StdEncoding.EncodeToString(buf)
	payload := fmt.Sprintf("%s %s\x00", b64, comment)
	return []byte(payload), nil
}

func bigIntToLittleEndian(n *big.Int, size int) []byte {
	be := n.Bytes()
	le := make([]byte, size)
	for i := 0; i < len(be) && i < size; i++ {
		le[i] = be[len(be)-1-i]
	}
	return le
}
