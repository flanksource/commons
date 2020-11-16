package validate

import "regexp"

var envVarNameRegexp *regexp.Regexp = regexp.MustCompile(`(?m)^[A-Z0-9\_]+$`)

// EnvVarName validates whether a string is a valid name for an environment variable.
// Note that technically the Unix specification says that some implementations may allow other characters,
// and that applications should tolerate wider character sets: https://pubs.opengroup.org/onlinepubs/007908799/xbd/envvar.html
// However, those env vars are uncommon and aren't generally accepted.
func EnvVarName(input string) (isValid bool) {
	return envVarNameRegexp.MatchString(input)
}

func LookupFile() {

}

func RsaPrivateKey() {

}

func RsaPublicKey() {

}

func EcsdaPrivateKey() {

}

func EcsdaPublicKey() {

}

func PrivateKey() {

}

func PublicKey() {

}

func X509Cert() {

}

func SSHKey() {

}
