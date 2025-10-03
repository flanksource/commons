// Package hash provides utility functions for generating cryptographic hashes
// and deterministic UUIDs from various data types.
//
// The package offers convenient methods to generate hashes from JSON-serializable
// objects and create deterministic UUIDs based on input data, which is useful
// for generating stable identifiers across distributed systems.
//
// Key features:
//   - JSON object hashing with MD5
//   - SHA256 string hashing
//   - Deterministic UUID generation from any JSON-serializable data
//
// Basic usage:
//
//	// Hash a struct to MD5
//	type User struct {
//		Name  string
//		Email string
//	}
//	user := User{Name: "John", Email: "john@example.com"}
//	hash, err := hash.JSONMD5Hash(user)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("MD5: %s\n", hash)
//
//	// Generate SHA256 hash
//	sha := hash.Sha256Hex("hello world")
//	fmt.Printf("SHA256: %s\n", sha)
//
//	// Create deterministic UUID
//	id, err := hash.DeterministicUUID(user)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("UUID: %s\n", id)
//
// Note: MD5 is used for non-cryptographic purposes like checksums and
// deterministic ID generation. For cryptographic security, use SHA256 or stronger.
package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
)

// JSONMD5Hash marshals the object into JSON and generates its MD5 hash
// in hexadecimal format. This is useful for creating checksums of
// structured data or generating cache keys.
//
// The object must be JSON-serializable. The resulting hash is deterministic
// for the same input data, making it suitable for comparison and deduplication.
//
// Note: MD5 is not cryptographically secure and should only be used for
// non-security purposes like checksums and identifiers.
//
// Example:
//
//	config := map[string]string{"host": "localhost", "port": "8080"}
//	hash, err := JSONMD5Hash(config)
//	// hash will be consistent for the same config values
func JSONMD5Hash(obj any) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	hash := md5.Sum(data)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash[:]), nil
}

// Sha256Hex computes the SHA256 hash of the input string and returns it
// as a hexadecimal string. SHA256 is cryptographically secure and suitable
// for security-sensitive applications.
//
// Example:
//
//	password := "secretpassword"
//	hash := Sha256Hex(password)
//	// hash = "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"
func Sha256Hex(in string) string {
	hash := sha256.New()
	hash.Write([]byte(in))
	hashVal := hash.Sum(nil)
	return hex.EncodeToString(hashVal[:])
}

// DeterministicUUID generates a UUID that is deterministic based on the input seed.
// The same seed will always produce the same UUID, which is useful for creating
// stable identifiers in distributed systems.
//
// The seed can be any JSON-serializable object. The function uses MD5 hashing
// internally to generate a 128-bit value that forms the UUID.
//
// Example:
//
//	// Generate UUID from user data
//	userData := map[string]string{
//		"email": "user@example.com",
//		"system": "production",
//	}
//	id, err := DeterministicUUID(userData)
//	// The same userData will always generate the same UUID
//
//	// Generate UUID from string
//	id2, err := DeterministicUUID("unique-resource-name")
func DeterministicUUID(seed any) (uuid.UUID, error) {
	byteHash, err := JSONMD5Hash(seed)
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.FromBytes([]byte(byteHash[0:16]))
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}
