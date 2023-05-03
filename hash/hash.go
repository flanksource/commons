package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
)

// JSONMD5Hash marshals the object into JSON and generates its md5 hash
// in hex format.
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

func Sha256Hex(in string) string {
	hash := sha256.New()
	hash.Write([]byte(in))
	hashVal := hash.Sum(nil)
	return hex.EncodeToString(hashVal[:])
}

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
